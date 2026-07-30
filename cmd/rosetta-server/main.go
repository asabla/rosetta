package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/asabla/rosetta/internal/service"
)

const (
	defaultAddress        = "127.0.0.1:8080"
	insecureHTTPVariable  = "ROSETTA_INSECURE_HTTP"
	tlsCertificateVariable = "ROSETTA_TLS_CERT_FILE"
	tlsKeyVariable         = "ROSETTA_TLS_KEY_FILE"
	clientCAVariable       = "ROSETTA_CLIENT_CA_FILE"
)

type serverConfig struct {
	address      string
	insecureHTTP bool
	tlsConfig    *tls.Config
}

func main() {
	config, err := loadServerConfig(os.LookupEnv)
	if err != nil {
		log.Fatal(err)
	}
	server := newServer(config)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown failed: %v", err)
		}
	}()

	log.Printf("listening on %s", config.address)
	if config.insecureHTTP {
		err = server.ListenAndServe()
	} else {
		err = server.ListenAndServeTLS("", "")
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func newServer(config serverConfig) *http.Server {
	return &http.Server{
		Addr:              config.address,
		Handler:           service.NewHandler(),
		TLSConfig:         config.tlsConfig,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func loadServerConfig(lookup func(string) (string, bool)) (serverConfig, error) {
	address := environmentValue(lookup, "ROSETTA_ADDR")
	if address == "" {
		address = defaultAddress
	}
	insecureValue := environmentValue(lookup, insecureHTTPVariable)
	if insecureValue != "" && insecureValue != "true" {
		return serverConfig{}, fmt.Errorf("%s must be exactly true when enabled", insecureHTTPVariable)
	}
	insecureHTTP := insecureValue == "true"

	certificateFile := environmentValue(lookup, tlsCertificateVariable)
	keyFile := environmentValue(lookup, tlsKeyVariable)
	clientCAFile := environmentValue(lookup, clientCAVariable)
	tlsInputs := 0
	for _, value := range []string{certificateFile, keyFile, clientCAFile} {
		if value != "" {
			tlsInputs++
		}
	}
	if tlsInputs != 0 && tlsInputs != 3 {
		return serverConfig{}, errors.New("mTLS configuration requires the certificate, private key, and client CA files")
	}
	if tlsInputs == 0 {
		if !insecureHTTP {
			return serverConfig{}, fmt.Errorf("mTLS is required; set %s only for loopback development", insecureHTTPVariable)
		}
		if !isLoopbackAddress(address) {
			return serverConfig{}, errors.New("insecure HTTP is restricted to a numeric loopback address")
		}
		return serverConfig{address: address, insecureHTTP: true}, nil
	}
	if insecureHTTP {
		return serverConfig{}, errors.New("insecure HTTP and mTLS configuration are mutually exclusive")
	}

	certificate, err := tls.LoadX509KeyPair(certificateFile, keyFile)
	if err != nil {
		return serverConfig{}, errors.New("load server certificate")
	}
	clientCAPEM, err := os.ReadFile(clientCAFile)
	if err != nil {
		return serverConfig{}, errors.New("load client CA")
	}
	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(clientCAPEM) {
		return serverConfig{}, errors.New("client CA file does not contain a certificate")
	}
	return serverConfig{
		address: address,
		tlsConfig: &tls.Config{
			MinVersion:   tls.VersionTLS13,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    clientCAs,
			Certificates: []tls.Certificate{certificate},
		},
	}, nil
}

func environmentValue(lookup func(string) (string, bool), name string) string {
	value, _ := lookup(name)
	return value
}

func isLoopbackAddress(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return false
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}
