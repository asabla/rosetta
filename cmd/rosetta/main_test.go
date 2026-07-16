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
	var stderr bytes.Buffer
	if err := run([]string{"compile", "-target", request.Target, "-catalog", path}, strings.NewReader(request.Source), &stdout, &stderr); err != nil {
		t.Fatalf("run compile: %v", err)
	}
	if got := stdout.String(); got != want.Artifacts[0].Content {
		t.Fatalf("artifact mismatch: got %q want %q", got, want.Artifacts[0].Content)
	}
}

func TestCLIVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"version"}, strings.NewReader(""), &stdout, &stderr); err != nil {
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
	var stderr bytes.Buffer
	err = run([]string{"check"}, strings.NewReader(request.Source), &stdout, &stderr)
	if err == nil {
		t.Fatal("expected check to fail")
	}
	if got := err.Error(); got != want.Diagnostics[0].Message {
		t.Fatalf("diagnostic mismatch: got %q want %q", got, want.Diagnostics[0].Message)
	}
}

func TestCLIPermissiveCompileReportsDiagnostics(t *testing.T) {
	request := rosetta.CompileRequest{
		Source: "permit(principal, action, resource);",
		Target: rosetta.TargetClaude,
		Mode:   rosetta.ModePermissive,
		Catalog: rosetta.Catalog{
			Version:      rosetta.CatalogVersion,
			Principal:    rosetta.EntityRef{ID: "agent"},
			Capabilities: []rosetta.Capability{{ID: "command", Kind: rosetta.KindCommand, Action: "execute", Selector: "go test"}},
		},
	}
	catalog, err := json.Marshal(request.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	path := t.TempDir() + "/catalog.json"
	if err := os.WriteFile(path, catalog, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"compile", "-target", request.Target, "-mode", request.Mode, "-catalog", path}, strings.NewReader(request.Source), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "capability_omitted") {
		t.Fatalf("expected omission diagnostic on stderr, got %q", stderr.String())
	}
}

func TestCLICompileJSONIncludesMetadataAndDiagnostics(t *testing.T) {
	request := rosetta.CompileRequest{
		Source: "permit(principal, action, resource);",
		Target: rosetta.TargetClaude,
		Mode:   rosetta.ModePermissive,
		Catalog: rosetta.Catalog{
			Version:      rosetta.CatalogVersion,
			Principal:    rosetta.EntityRef{ID: "agent"},
			Capabilities: []rosetta.Capability{{ID: "command", Kind: rosetta.KindCommand, Action: "execute", Selector: "go test"}},
		},
	}
	catalog, err := json.Marshal(request.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	path := t.TempDir() + "/catalog.json"
	if err := os.WriteFile(path, catalog, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"compile", "-format", "json", "-target", request.Target, "-mode", request.Mode, "-catalog", path}, strings.NewReader(request.Source), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var result rosetta.CompileResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode JSON output: %v", err)
	}
	if result.Metadata.InputSHA256 == "" || result.Metadata.TargetContractVersion == "" {
		t.Fatalf("missing compile metadata: %#v", result.Metadata)
	}
	if len(result.Diagnostics) == 0 {
		t.Fatal("JSON result omitted diagnostics")
	}
	if stderr.Len() != 0 {
		t.Fatalf("JSON diagnostics were duplicated on stderr: %q", stderr.String())
	}
}

func TestCLISourceReadIsBounded(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"check"}, strings.NewReader(strings.Repeat("x", rosetta.MaxSourceBytes+1)), &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "source exceeds") {
		t.Fatalf("expected bounded source error, got %v", err)
	}
}
