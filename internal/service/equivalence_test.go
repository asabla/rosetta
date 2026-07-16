package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/asabla/rosetta"
)

func TestHTTPCompileMatchesSharedCompilerAPI(t *testing.T) {
	request := rosetta.CompileRequest{
		Source: "permit(principal, action, resource);",
		Target: "openshell",
		Catalog: rosetta.Catalog{
			Version:      "rosetta/v0.5",
			Principal:    rosetta.EntityRef{ID: "agent"},
			Capabilities: []rosetta.Capability{{ID: "workspace", Kind: "filesystem", Action: "read", Selector: "/workspace"}},
		},
	}
	want, err := rosetta.Compile(context.Background(), request)
	if err != nil {
		t.Fatalf("shared compile failed: %v", err)
	}

	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	rec := httptest.NewRecorder()
	NewHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/compile", bytes.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var got rosetta.CompileResult
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !reflect.DeepEqual(got.Diagnostics, want.Diagnostics) {
		t.Fatalf("diagnostics mismatch: got %#v want %#v", got.Diagnostics, want.Diagnostics)
	}
	if !reflect.DeepEqual(got.Artifacts, want.Artifacts) {
		t.Fatalf("artifacts mismatch: got %#v want %#v", got.Artifacts, want.Artifacts)
	}
	if got.Metadata != want.Metadata {
		t.Fatalf("metadata mismatch: got %#v want %#v", got.Metadata, want.Metadata)
	}
}

func TestHTTPCheckMatchesSharedCompilerDiagnostics(t *testing.T) {
	request := rosetta.CheckRequest{Source: ""}
	want, err := rosetta.Check(context.Background(), request)
	if err != nil {
		t.Fatalf("shared check failed: %v", err)
	}

	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	rec := httptest.NewRecorder()
	NewHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/check", bytes.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}

	var got rosetta.CheckResult
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !reflect.DeepEqual(got.Diagnostics, want.Diagnostics) {
		t.Fatalf("diagnostics mismatch: got %#v want %#v", got.Diagnostics, want.Diagnostics)
	}
}
