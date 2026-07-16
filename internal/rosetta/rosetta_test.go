package rosetta

import (
	"context"
	"testing"
)

func TestCompilePermissiveModeReturnsWarning(t *testing.T) {
	result, err := Compile(context.Background(), CompileRequest{Source: "permit();", Target: "openshell", Mode: ModePermissive})
	if err != nil {
		t.Fatalf("compile permissive: %v", err)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Severity != "warning" {
		t.Fatalf("expected permissive warning diagnostic, got %#v", result.Diagnostics)
	}
}

func TestCompileRejectsUnknownMode(t *testing.T) {
	_, err := Compile(context.Background(), CompileRequest{Source: "permit();", Target: "openshell", Mode: "unsafe"})
	if err == nil {
		t.Fatal("expected unsupported mode error")
	}
}
