package authz

import (
	"io"
	"log/slog"
	"testing"
)

func TestCedarDecisionMapping(t *testing.T) {
	a := NewCedarAuthorizer(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if a.IsAllowed(Request{Action: ConnectHost}).Allowed {
		t.Fatal("default deny failed")
	}
	if !a.IsAllowed(Request{Principal: "agent", Action: ConnectHost, Context: map[string]string{"host": "example.com"}}).Allowed {
		t.Fatal("expected permit")
	}
	if a.IsAllowed(Request{Principal: "agent", Action: UseSecret, Context: map[string]string{"secret": "x", "long_lived": "true"}}).Allowed {
		t.Fatal("long secret allowed")
	}
}
