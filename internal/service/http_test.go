package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompileEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/compile", strings.NewReader(`{
        "source":"permit(principal, action, resource);",
        "target":"openshell",
        "catalog":{
            "version":"rosetta/v0.5",
            "principal":{"id":"agent"},
            "capabilities":[{"id":"workspace","kind":"filesystem","action":"read","selector":"/workspace"}]
        }
    }`))
	rec := httptest.NewRecorder()

	NewHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "filesystem_policy:") {
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
		`"ExplainPolicyResponse"`,
		`"Catalog"`,
		`"Capability"`,
		`"Decision"`,
		`"Diagnostic"`,
		`"Artifact"`,
		`"CompileMetadata"`,
		`"TargetContractInfo"`,
		`"SourceSpan"`,
		`"mediaType"`,
		`"encoding"`,
		`"TargetOptions"`,
		`"OpenShellOptions"`,
		`"CodexOptions"`,
		`"CodexMCPServer"`,
		`"targetContracts"`,
		`"targetContractVersion"`,
		`"inputSha256"`,
		`"artifactSha256"`,
		`"const": "rosetta/v0.5"`,
		`"version": "0.5.0"`,
	} {
		if !strings.Contains(string(doc), expected) {
			t.Fatalf("expected OpenAPI schema to include %s", expected)
		}
	}
}
