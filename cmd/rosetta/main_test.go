package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/asabla/rosetta"
)

func TestCLICompileMatchesSharedCompilerArtifact(t *testing.T) {
	request := rosetta.CompileRequest{
		Source: "permit(principal, action, resource);",
		Target: "openshell",
		Catalog: rosetta.Catalog{
			Version:      rosetta.CatalogVersion,
			Principal:    rosetta.EntityRef{ID: "agent"},
			Capabilities: []rosetta.Capability{{ID: "workspace", Kind: "filesystem", Action: "read", Selector: "/workspace"}},
		},
	}
	want, err := rosetta.Compile(context.Background(), request)
	if err != nil {
		t.Fatalf("shared compile failed: %v", err)
	}

	catalog, err := json.Marshal(request.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	path := t.TempDir() + "/catalog.json"
	if err := os.WriteFile(path, catalog, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if err := run([]string{"compile", "-target", request.Target, "-catalog", path}, strings.NewReader(request.Source), &stdout); err != nil {
		t.Fatalf("run compile: %v", err)
	}
	if got := stdout.String(); got != want.Artifacts[0].Content {
		t.Fatalf("artifact mismatch: got %q want %q", got, want.Artifacts[0].Content)
	}
}

func TestCLIVersion(t *testing.T) {
	var stdout bytes.Buffer
	if err := run([]string{"version"}, strings.NewReader(""), &stdout); err != nil {
		t.Fatal(err)
	}
	if got, want := stdout.String(), "rosetta "+rosetta.Version+"\n"; got != want {
		t.Fatalf("version mismatch: got %q want %q", got, want)
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
