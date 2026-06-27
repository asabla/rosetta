package validate

import "testing"

func TestNormalizeHost(t *testing.T) {
	h, err := NormalizeHost("Example.COM.")
	if err != nil || h != "example.com" {
		t.Fatalf("%s %v", h, err)
	}
	if _, err := NormalizeHost("*.example.com"); err == nil {
		t.Fatal("wildcard accepted")
	}
}
func TestSafePath(t *testing.T) {
	if p, err := SafePath("/tmp/work"); err != nil || p != "/tmp/work" {
		t.Fatalf("%s %v", p, err)
	}
	for _, p := range []string{"../x", "/etc/passwd", "/proc/cpuinfo", "/root/.ssh"} {
		if _, err := SafePath(p); err == nil {
			t.Fatalf("accepted %s", p)
		}
	}
}
