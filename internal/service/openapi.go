package service

import "encoding/json"

// OpenAPISchema returns the Rosetta service OpenAPI document.
func OpenAPISchema() []byte {
	schema := map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "Rosetta API",
			"version":     "0.1.0",
			"description": "HTTP API for checking Cedar policies and translating them into supported agent runtime targets.",
		},
		"paths": map[string]any{
			"/v1/compile":      map[string]any{"post": operation("compilePolicy", "Compile policy input", "Translate Cedar policy source into a target-specific artifact.", "CompilePolicyRequest", "CompilePolicyResponse")},
			"/v1/check":        map[string]any{"post": operation("checkPolicy", "Check policy input", "Validate Cedar policy source and return diagnostics without producing artifacts.", "CheckPolicyRequest", "CheckPolicyResponse")},
			"/v1/explain":      map[string]any{"post": operation("explainPolicyTranslation", "Explain policy translation", "Describe how Rosetta translates Cedar policy source for a requested target.", "ExplainPolicyRequest", "ExplainPolicyResponse")},
			"/v1/capabilities": map[string]any{"get": operationWithoutRequest("listTargetCapabilities", "List target capabilities", "List Rosetta target capabilities and service features.", "TargetCapability")},
			"/v1/targets":      map[string]any{"get": operationWithoutRequest("listSupportedTargets", "List supported targets", "List target identifiers accepted by translation requests.", "TargetCapability")},
			"/v1/openapi.json": map[string]any{"get": operationWithoutRequest("getOpenAPISchema", "Fetch the OpenAPI schema", "Return the OpenAPI description for this Rosetta service.", "OpenAPISchema")},
			"/healthz":         map[string]any{"get": operationWithoutRequest("getHealth", "Health check", "Report whether the Rosetta service is available.", "HealthResponse")},
		},
		"components": map[string]any{
			"schemas": schemas(),
		},
	}
	body, _ := json.MarshalIndent(schema, "", "  ")
	return body
}

func operation(operationID, summary, description, requestName, responseName string) map[string]any {
	return map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": description,
		"requestBody": map[string]any{
			"required": true,
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": schemaRef(requestName),
				},
			},
		},
		"responses": successResponse(responseName),
	}
}

func operationWithoutRequest(operationID, summary, description, responseName string) map[string]any {
	return map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": description,
		"responses":   successResponse(responseName),
	}
}

func successResponse(responseName string) map[string]any {
	return map[string]any{
		"200": map[string]any{
			"description": "OK",
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": schemaRef(responseName),
				},
			},
		},
	}
}

func schemas() map[string]any {
	return map[string]any{
		"CompilePolicyRequest":  objectSchema("Request to translate Cedar policy source into a target-specific policy artifact.", []string{"source"}, propertyMap(policySourceProperty(), targetProperty(), targetOptionsProperty()), compileExamples()),
		"CompilePolicyResponse": objectSchema("Response containing generated policy artifacts for the requested target.", []string{"target", "artifacts"}, propertyMap(targetProperty(), artifactsProperty(), diagnosticsProperty()), nil),
		"CheckPolicyRequest":    objectSchema("Request to validate Cedar policy source without generating target artifacts.", []string{"source"}, propertyMap(policySourceProperty()), nil),
		"CheckPolicyResponse":   objectSchema("Response describing whether policy source is valid and any diagnostics found.", []string{"valid"}, propertyMap(boolProperty("valid", "Whether the supplied policy source passed validation."), diagnosticsProperty()), nil),
		"ExplainPolicyRequest":  objectSchema("Request to explain how Cedar policy source would be translated for a target.", []string{"source"}, propertyMap(policySourceProperty(), targetProperty(), targetOptionsProperty()), nil),
		"ExplainPolicyResponse": objectSchema("Response containing a human-readable explanation of the translation process.", []string{"explanation"}, propertyMap(stringProperty("explanation", "Explanation of validation, normalization, and target rendering decisions."), diagnosticsProperty()), nil),
		"TargetCapability":      objectSchema("Capability metadata for one Rosetta translation target.", []string{"target", "displayName", "description"}, propertyMap(targetProperty(), stringProperty("displayName", "Human-readable target name."), stringProperty("description", "Summary of the target integration and generated output."), arrayProperty("features", "Supported translation features for this target.", stringSchema("Feature name."))), nil),
		"Diagnostic":            diagnosticSchema(),
		"SourceSpan":            objectSchema("Source range associated with a diagnostic.", nil, propertyMap(intProperty("startLine", "One-based starting line for the source range."), intProperty("startColumn", "One-based starting column for the source range."), intProperty("endLine", "One-based ending line for the source range."), intProperty("endColumn", "One-based ending column for the source range.")), nil),
		"Artifact":              artifactSchema(),
		"PolicySource":          stringSchema("Raw Cedar policy source submitted to Rosetta."),
		"TargetOptions":         objectSchema("Target-specific options for Rosetta policy translation.", nil, propertyMap(refProperty("openShell", "OpenShell translation options.", "OpenShellOptions"), refProperty("openCode", "OpenCode translation options.", "OpenCodeOptions"), refProperty("codex", "Codex translation options.", "CodexOptions"), refProperty("claudeCode", "Claude Code translation options.", "ClaudeCodeOptions")), nil),
		"OpenShellOptions":      objectSchema("Options that control OpenShell policy artifact generation.", nil, propertyMap(stringProperty("profileName", "OpenShell profile name to embed in generated output.")), nil),
		"OpenCodeOptions":       objectSchema("Options that control OpenCode policy artifact generation.", nil, propertyMap(stringProperty("workspaceRoot", "Workspace root path used by OpenCode policies.")), nil),
		"CodexOptions":          objectSchema("Options that control Codex policy artifact generation.", nil, propertyMap(stringProperty("approvalMode", "Codex approval mode represented in generated policy output.")), nil),
		"ClaudeCodeOptions":     objectSchema("Options that control Claude Code policy artifact generation.", nil, propertyMap(stringProperty("settingsScope", "Claude Code settings scope for generated policy output.")), nil),
		"OpenAPISchema":         map[string]any{"type": "object", "description": "OpenAPI document describing the Rosetta HTTP API."},
		"HealthResponse":        objectSchema("Health check response for the Rosetta service.", []string{"status"}, propertyMap(stringProperty("status", "Service status value.")), nil),
	}
}

func diagnosticSchema() map[string]any {
	return objectSchema(
		"Validation or translation diagnostic emitted while processing policy source. The code field is stable enough for automation, while message remains human-readable.",
		[]string{"severity", "code", "message"},
		propertyMap(
			stringProperty("severity", "Diagnostic severity such as error, warning, or info."),
			stringProperty("code", "Stable diagnostic code for programmatic handling."),
			stringProperty("message", "Human-readable diagnostic text explaining the issue or warning."),
			objectProperty("details", "Additional structured diagnostic metadata."),
			refProperty("sourceSpan", "Source range associated with this diagnostic.", "SourceSpan"),
			stringProperty("target", "Target identifier related to this diagnostic, when applicable."),
			stringProperty("ruleId", "Rule identifier related to this diagnostic, when applicable."),
			boolProperty("recoverable", "Whether processing can recover from this diagnostic."),
			stringProperty("documentationUrl", "Documentation URL for remediation guidance."),
		),
		nil,
	)
}

func artifactSchema() map[string]any {
	return objectSchema(
		"Generated target artifact produced by a policy translation. Content is plain text by default when encoding is plain; use base64 or another explicit encoding for encoded content.",
		[]string{"name", "mediaType", "content", "encoding"},
		propertyMap(
			stringProperty("name", "Artifact file name or logical identifier."),
			stringProperty("pathHint", "Suggested relative output path for writing this artifact."),
			stringProperty("mediaType", "Media type for the artifact contents."),
			stringProperty("target", "Target identifier this artifact was generated for."),
			stringProperty("content", "Artifact contents generated for the target."),
			stringProperty("encoding", "Encoding for content, such as plain or base64."),
			stringProperty("description", "Human-readable summary of the generated artifact."),
		),
		nil,
	)
}

func objectSchema(description string, required []string, properties map[string]any, examples []any) map[string]any {
	schema := map[string]any{"type": "object", "description": description, "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	if len(examples) > 0 {
		schema["examples"] = examples
	}
	return schema
}

func propertyMap(properties ...map[string]any) map[string]any {
	merged := map[string]any{}
	for _, property := range properties {
		for name, schema := range property {
			merged[name] = schema
		}
	}
	return merged
}

func compileExamples() []any {
	return []any{
		map[string]any{"source": "permit(principal, action, resource);", "target": "openshell", "options": map[string]any{"openShell": map[string]any{"profileName": "default"}}},
		map[string]any{"source": "permit(principal, action, resource);", "target": "opencode", "options": map[string]any{"openCode": map[string]any{"workspaceRoot": "/workspace"}}},
		map[string]any{"source": "permit(principal, action, resource);", "target": "codex", "options": map[string]any{"codex": map[string]any{"approvalMode": "on-request"}}},
		map[string]any{"source": "permit(principal, action, resource);", "target": "claude-code", "options": map[string]any{"claudeCode": map[string]any{"settingsScope": "project"}}},
	}
}

func schemaRef(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

func refProperty(name, description, refName string) map[string]any {
	return map[string]any{name: map[string]any{"allOf": []any{schemaRef(refName)}, "description": description}}
}

func policySourceProperty() map[string]any {
	return refProperty("source", "Cedar policy source to process.", "PolicySource")
}

func targetOptionsProperty() map[string]any {
	return refProperty("options", "Optional target-specific translation settings.", "TargetOptions")
}

func targetProperty() map[string]any {
	return map[string]any{"target": map[string]any{"type": "string", "description": "Target identifier for generated policy output.", "examples": []any{"openshell", "opencode", "codex", "claude-code"}}}
}

func artifactsProperty() map[string]any {
	return arrayProperty("artifacts", "Generated target artifacts.", schemaRef("Artifact"))
}
func diagnosticsProperty() map[string]any {
	return arrayProperty("diagnostics", "Diagnostics emitted while processing the request.", schemaRef("Diagnostic"))
}

func arrayProperty(name, description string, items map[string]any) map[string]any {
	return map[string]any{name: map[string]any{"type": "array", "description": description, "items": items}}
}

func stringProperty(name, description string) map[string]any {
	return map[string]any{name: stringSchema(description)}
}
func boolProperty(name, description string) map[string]any {
	return map[string]any{name: map[string]any{"type": "boolean", "description": description}}
}
func intProperty(name, description string) map[string]any {
	return map[string]any{name: map[string]any{"type": "integer", "description": description}}
}
func objectProperty(name, description string) map[string]any {
	return map[string]any{name: map[string]any{"type": "object", "description": description, "additionalProperties": true}}
}
func stringSchema(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}
