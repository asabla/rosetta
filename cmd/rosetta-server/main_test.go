package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadServerConfigRequiresAuthenticatedTransport(t *testing.T) {
	if _, err := loadServerConfig(environment(nil)); err == nil {
		t.Fatal("expected missing authenticated transport to fail")
	}
}

func TestLoadServerConfigRejectsIncompleteTLS(t *testing.T) {
	values := map[string]string{tlsCertificateVariable: "/tmp/cert.pem"}
	if _, err := loadServerConfig(environment(values)); err == nil {
		t.Fatal("expected incomplete mTLS configuration to fail")
	}
}

func TestLoadServerConfigAllowsOnlyExplicitLoopbackHTTP(t *testing.T) {
	config, err := loadServerConfig(environment(map[string]string{
		"ROSETTA_ADDR":       "127.0.0.1:8080",
		insecureHTTPVariable: "true",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !config.insecureHTTP || config.tlsConfig != nil {
		t.Fatalf("unexpected insecure configuration: %#v", config)
	}
	for _, address := range []string{":8080", "0.0.0.0:8080", "localhost:8080"} {
		t.Run(address, func(t *testing.T) {
			_, err := loadServerConfig(environment(map[string]string{
				"ROSETTA_ADDR":       address,
				insecureHTTPVariable: "true",
			}))
			if err == nil {
				t.Fatalf("expected insecure address %q to fail", address)
			}
		})
	}
}

func TestMutualTLSRequiresTrustedClientCertificate(t *testing.T) {
	files, roots, clientCertificate := transportCertificates(t)
	config, err := loadServerConfig(environment(map[string]string{
		"ROSETTA_ADDR":          "127.0.0.1:0",
		tlsCertificateVariable: files.serverCertificate,
		tlsKeyVariable:         files.serverKey,
		clientCAVariable:       files.caCertificate,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if config.tlsConfig.MinVersion != tls.VersionTLS13 ||
		config.tlsConfig.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Fatalf("unexpected mTLS policy: %#v", config.tlsConfig)
	}

	listener, err := net.Listen("tcp", config.address)
	if err != nil {
		t.Fatal(err)
	}
	server := newServer(config)
	t.Cleanup(func() {
		_ = server.Close()
	})
	go func() {
		_ = server.Serve(tls.NewListener(listener, config.tlsConfig))
	}()

	baseTLS := &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots}
	unauthenticated := &http.Client{
		Timeout:   2 * time.Second,
		Transport: &http.Transport{TLSClientConfig: baseTLS.Clone()},
	}
	if response, err := unauthenticated.Get("https://" + listener.Addr().String() + "/healthz"); err == nil {
		_ = response.Body.Close()
		t.Fatal("expected a client without a certificate to be rejected")
	}

	authenticatedTLS := baseTLS.Clone()
	authenticatedTLS.Certificates = []tls.Certificate{clientCertificate}
	authenticated := &http.Client{
		Timeout:   2 * time.Second,
		Transport: &http.Transport{TLSClientConfig: authenticatedTLS},
	}
	response, err := authenticated.Get("https://" + listener.Addr().String() + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}
}

type certificateFiles struct {
	caCertificate     string
	serverCertificate string
	serverKey         string
}

func transportCertificates(t *testing.T) (certificateFiles, *x509.CertPool, tls.Certificate) {
	t.Helper()
	caTemplate, caKey, caPEM := certificateAuthority(t)
	serverCertificate, serverKey := signedCertificate(t, caTemplate, caKey, false)
	clientCertificate, clientKey := signedCertificate(t, caTemplate, caKey, true)
	directory := t.TempDir()
	files := certificateFiles{
		caCertificate:     writeCertificateFile(t, directory, "ca.pem", caPEM),
		serverCertificate: writeCertificateFile(t, directory, "server.pem", serverCertificate),
		serverKey:         writeCertificateFile(t, directory, "server-key.pem", serverKey),
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(caPEM) {
		t.Fatal("append test CA")
	}
	client, err := tls.X509KeyPair(clientCertificate, clientKey)
	if err != nil {
		t.Fatal(err)
	}
	return files, roots, client
}

func certificateAuthority(t *testing.T) (*x509.Certificate, *ecdsa.PrivateKey, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Rosetta test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	body, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return template, key, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: body})
}

func signedCertificate(
	t *testing.T,
	ca *x509.Certificate,
	caKey *ecdsa.PrivateKey,
	client bool,
) ([]byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	usage := x509.ExtKeyUsageServerAuth
	commonName := "rosetta-server"
	if client {
		usage = x509.ExtKeyUsageClientAuth
		commonName = "dataground"
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{usage},
	}
	if !client {
		template.DNSNames = []string{"localhost"}
		template.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
	}
	body, err := x509.CreateCertificate(rand.Reader, template, ca, &key.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	keyBody, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: body}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBody})
}

func writeCertificateFile(t *testing.T, directory, name string, body []byte) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func environment(values map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, found := values[name]
		return value, found
	}
}
