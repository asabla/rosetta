package compiler

import (
	"strings"
	"testing"
)

func TestCompileDeterministicYAML(t *testing.T) {
	b, h, err := Compile([]Grant{{Kind: "ConnectHost", Host: "example.com", Port: 443, Binary: "/bin/curl"}, {Kind: "ReadPath", Path: "/tmp"}})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{"version: 1", "filesystem_policy:", "network_policies:", "host: example.com", "path: /bin/curl"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s in\n%s", want, s)
		}
	}
	if !strings.HasPrefix(h, "sha256:") {
		t.Fatal(h)
	}
}
