package service

import "encoding/json"

// OpenAPISchema returns the Rosetta service OpenAPI document.
func OpenAPISchema() []byte {
	schema := map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":   "Rosetta API",
			"version": "0.1.0",
		},
		"paths": map[string]any{
			"/v1/compile":      map[string]any{"post": operation("Compile policy input", "CompileResponse")},
			"/v1/check":        map[string]any{"post": operation("Check policy input", "CheckResponse")},
			"/v1/explain":      map[string]any{"post": operation("Explain policy compilation", "ExplainResponse")},
			"/v1/capabilities": map[string]any{"get": operation("List service capabilities", "CapabilitiesResponse")},
			"/v1/targets":      map[string]any{"get": operation("List supported targets", "TargetsResponse")},
			"/v1/openapi.json": map[string]any{"get": operation("Fetch the OpenAPI schema", "object")},
			"/healthz":         map[string]any{"get": operation("Health check", "object")},
		},
	}
	body, _ := json.MarshalIndent(schema, "", "  ")
	return body
}

func operation(summary, responseName string) map[string]any {
	return map[string]any{
		"summary": summary,
		"responses": map[string]any{
			"200": map[string]any{
				"description": "OK",
				"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"title": responseName}}},
			},
		},
	}
}
