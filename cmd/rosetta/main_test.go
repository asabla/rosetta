package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"rosetta/internal/rosetta"
)

func TestCLICompileMatchesSharedCompilerArtifact(t *testing.T) {
	request := rosetta.CompileRequest{Source: "permit();", Target: "openshell"}
	want, err := rosetta.Compile(context.Background(), request)
	if err != nil {
		t.Fatalf("shared compile failed: %v", err)
	}

	var stdout bytes.Buffer
	if err := run([]string{"compile", "-target", request.Target}, strings.NewReader(request.Source), &stdout); err != nil {
		t.Fatalf("run compile: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != want.Artifacts[0].Content {
		t.Fatalf("artifact mismatch: got %q want %q", got, want.Artifacts[0].Content)
	}
}

func TestCLICheckMatchesSharedCompilerDiagnostics(t *testing.T) {
	request := rosetta.CheckRequest{Source: ""}
	want, err := rosetta.Check(context.Background(), request)
	if err != nil {
		t.Fatalf("shared check failed: %v", err)
	}

	var stdout bytes.Buffer
	err = run([]string{"check"}, strings.NewReader(request.Source), &stdout)
	if err == nil {
		t.Fatal("expected check to fail")
	}
	if got := err.Error(); got != want.Diagnostics[0].Message {
		t.Fatalf("diagnostic mismatch: got %q want %q", got, want.Diagnostics[0].Message)
	}
}
