package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompileEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/compile", strings.NewReader(`{"source":"permit();","target":"openshell"}`))
	rec := httptest.NewRecorder()

	NewHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "# target: openshell") {
		t.Fatalf("expected compiled target in response, got %s", rec.Body.String())
	}
}

func TestCheckEndpointRejectsEmptySource(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/check", strings.NewReader(`{"source":""}`))
	rec := httptest.NewRecorder()

	NewHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestOpenAPIEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/openapi.json", nil)
	rec := httptest.NewRecorder()

	NewHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "/v1/compile") {
		t.Fatalf("expected OpenAPI response to include compile path")
	}
}

func TestOpenAPISchemaUsesOperationIDsAndDescribedSchemas(t *testing.T) {
	doc := OpenAPISchema()
	for _, expected := range []string{
		`"operationId": "compilePolicy"`,
		`"operationId": "checkPolicy"`,
		`"operationId": "explainPolicyTranslation"`,
		`"operationId": "listTargetCapabilities"`,
		`"operationId": "listSupportedTargets"`,
		`"operationId": "getOpenAPISchema"`,
		`"operationId": "getHealth"`,
		`"CompilePolicyRequest"`,
		`"CompilePolicyResponse"`,
		`"CheckPolicyRequest"`,
		`"CheckPolicyResponse"`,
		`"ExplainPolicyRequest"`,
		`"ExplainPolicyResponse"`,
		`"TargetCapability"`,
		`"Diagnostic"`,
		`"Artifact"`,
		`"PolicySource"`,
		`"TargetOptions"`,
		`"OpenShellOptions"`,
		`"OpenCodeOptions"`,
		`"CodexOptions"`,
		`"ClaudeCodeOptions"`,
		`"description"`,
		`"target": "openshell"`,
		`"target": "opencode"`,
		`"target": "codex"`,
		`"target": "claude-code"`,
	} {
		if !strings.Contains(string(doc), expected) {
			t.Fatalf("expected OpenAPI schema to include %s", expected)
		}
	}
}
