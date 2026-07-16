package service

import (
	"encoding/json"

	"github.com/asabla/rosetta"
)

// OpenAPISchema returns the Rosetta service OpenAPI document.
func OpenAPISchema() []byte {
	document := map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "Rosetta API",
			"version":     rosetta.Version,
			"description": "Validate Cedar policies and compile catalogued capabilities into restrictive agent-runtime configuration.",
		},
		"paths": map[string]any{
			"/v1/compile":      map[string]any{"post": operation("compilePolicy", "Compile policy", "Compile Cedar decisions into one target artifact.", "CompilePolicyRequest", "CompilePolicyResponse")},
			"/v1/check":        map[string]any{"post": operation("checkPolicy", "Check policy", "Parse Cedar and validate it against the Rosetta profile.", "CheckPolicyRequest", "CheckPolicyResponse")},
			"/v1/explain":      map[string]any{"post": operation("explainPolicyTranslation", "Explain compilation", "Return the Cedar decisions behind a target artifact.", "CompilePolicyRequest", "ExplainPolicyResponse")},
			"/v1/capabilities": map[string]any{"get": readOperation("listTargetCapabilities", "List compiler capabilities", "CapabilitiesResponse")},
			"/v1/targets":      map[string]any{"get": readOperation("listSupportedTargets", "List supported targets", "TargetsResponse")},
			"/v1/openapi.json": map[string]any{"get": readOperation("getOpenAPISchema", "Fetch this OpenAPI document", "OpenAPISchema")},
			"/healthz":         map[string]any{"get": readOperation("getHealth", "Check service health", "HealthResponse")},
		},
		"components": map[string]any{"schemas": schemas()},
	}
	body, _ := json.MarshalIndent(document, "", "  ")
	return body
}

func operation(id, summary, description, request, response string) map[string]any {
	return map[string]any{
		"operationId": id,
		"summary":     summary,
		"description": description,
		"requestBody": map[string]any{
			"required": true,
			"content":  map[string]any{"application/json": map[string]any{"schema": ref(request)}},
		},
		"responses": responses(response),
	}
}

func readOperation(id, summary, response string) map[string]any {
	return map[string]any{"operationId": id, "summary": summary, "responses": responses(response)}
}

func responses(name string) map[string]any {
	return map[string]any{
		"200": map[string]any{"description": "OK", "content": map[string]any{"application/json": map[string]any{"schema": ref(name)}}},
		"400": map[string]any{"description": "Invalid request", "content": map[string]any{"application/json": map[string]any{"schema": ref("ErrorResponse")}}},
	}
}

func schemas() map[string]any {
	stringArray := func(description string) map[string]any {
		return map[string]any{"type": "array", "description": description, "items": map[string]any{"type": "string"}}
	}
	return map[string]any{
		"CompilePolicyRequest": object(
			"Cedar source, target, and the complete capability catalog to authorize.",
			[]string{"source", "target", "catalog"},
			map[string]any{
				"source":  map[string]any{"type": "string"},
				"target":  targetSchema(),
				"mode":    modeSchema(),
				"catalog": ref("Catalog"),
				"options": ref("TargetOptions"),
			},
		),
		"CompilePolicyResponse": object(
			"The deterministic artifact and Cedar decision trace.",
			[]string{"output", "target", "artifacts", "decisions"},
			map[string]any{
				"output":      map[string]any{"type": "string"},
				"target":      targetSchema(),
				"artifacts":   arrayOf("Artifact"),
				"decisions":   arrayOf("Decision"),
				"diagnostics": arrayOf("Diagnostic"),
			},
		),
		"CheckPolicyRequest": object("Cedar source to validate.", []string{"source"}, map[string]any{
			"source": map[string]any{"type": "string"},
			"mode":   modeSchema(),
		}),
		"CheckPolicyResponse": object("Cedar validation result.", []string{"valid"}, map[string]any{
			"valid":       map[string]any{"type": "boolean"},
			"diagnostics": arrayOf("Diagnostic"),
			"errors":      stringArray("Compatibility error messages."),
		}),
		"ExplainPolicyResponse": object("Human-readable compilation explanation and decisions.", []string{"explanation"}, map[string]any{
			"explanation": map[string]any{"type": "string"},
			"decisions":   arrayOf("Decision"),
			"diagnostics": arrayOf("Diagnostic"),
		}),
		"Catalog": object("Finite capability universe evaluated by Cedar.", []string{"version", "principal", "capabilities"}, map[string]any{
			"version":      map[string]any{"type": "string", "const": rosetta.CatalogVersion},
			"principal":    ref("EntityRef"),
			"capabilities": arrayOf("Capability"),
		}),
		"EntityRef": object("The Rosetta Cedar principal.", []string{"id"}, map[string]any{
			"type":  map[string]any{"type": "string", "default": "Rosetta::Principal"},
			"id":    map[string]any{"type": "string", "minLength": 1},
			"roles": stringArray("Principal role attributes available to Cedar."),
		}),
		"Capability": object("One typed operation Cedar must allow or deny. Filesystem selectors are directory roots without glob syntax.", []string{"id", "kind", "action", "selector"}, map[string]any{
			"id":       map[string]any{"type": "string", "minLength": 1},
			"kind":     map[string]any{"type": "string", "enum": []string{"filesystem", "tool", "command", "network"}},
			"action":   map[string]any{"type": "string", "enum": []string{"read", "write", "use", "execute", "connect"}},
			"selector": map[string]any{"type": "string", "minLength": 1},
			"targets":  stringArray("Targets for which this capability is relevant."),
			"access":   map[string]any{"type": "string"},
			"port":     map[string]any{"type": "integer", "minimum": 1, "maximum": 65535},
			"protocol": map[string]any{"type": "string"},
			"path":     map[string]any{"type": "string"},
			"binaries": stringArray("OpenShell executable paths allowed to reach an endpoint."),
			"server":   map[string]any{"type": "string", "description": "MCP server identifier for Codex tool capabilities."},
		}),
		"Decision": object("Cedar result for one catalog entry.", []string{"capabilityId", "allowed"}, map[string]any{
			"capabilityId": map[string]any{"type": "string"},
			"allowed":      map[string]any{"type": "boolean"},
			"policyIds":    stringArray("Cedar policies responsible for the decision."),
		}),
		"Artifact": object("Generated target file.", []string{"name", "mediaType", "target", "content", "encoding"}, map[string]any{
			"name":        map[string]any{"type": "string"},
			"pathHint":    map[string]any{"type": "string"},
			"mediaType":   map[string]any{"type": "string"},
			"target":      targetSchema(),
			"content":     map[string]any{"type": "string"},
			"encoding":    map[string]any{"type": "string", "const": "plain"},
			"description": map[string]any{"type": "string"},
		}),
		"Diagnostic": object("Machine-addressable compiler diagnostic.", []string{"severity", "code", "message"}, map[string]any{
			"severity":         map[string]any{"type": "string", "enum": []string{"error", "warning", "info"}},
			"code":             map[string]any{"type": "string"},
			"message":          map[string]any{"type": "string"},
			"details":          map[string]any{"type": "object", "additionalProperties": true},
			"sourceSpan":       ref("SourceSpan"),
			"target":           targetSchema(),
			"ruleId":           map[string]any{"type": "string"},
			"recoverable":      map[string]any{"type": "boolean"},
			"documentationUrl": map[string]any{"type": "string", "format": "uri"},
		}),
		"SourceSpan": object("One-based Cedar source range.", nil, map[string]any{
			"startLine":   map[string]any{"type": "integer"},
			"startColumn": map[string]any{"type": "integer"},
			"endLine":     map[string]any{"type": "integer"},
			"endColumn":   map[string]any{"type": "integer"},
		}),
		"TargetOptions": object("Target-specific rendering options.", nil, map[string]any{
			"openShell": ref("OpenShellOptions"),
			"codex":     ref("CodexOptions"),
		}),
		"OpenShellOptions": object("OpenShell hardening options.", nil, map[string]any{
			"includeWorkdir":        map[string]any{"type": "boolean"},
			"landlockCompatibility": map[string]any{"type": "string", "enum": []string{"hard_requirement", "best_effort"}},
			"runAsUser":             map[string]any{"type": "string"},
			"runAsGroup":            map[string]any{"type": "string"},
		}),
		"CodexOptions": object("Codex permission-profile options.", nil, map[string]any{
			"profileName":   map[string]any{"type": "string"},
			"workspaceRoot": map[string]any{"type": "string"},
			"mcpServers": map[string]any{
				"type":                 "object",
				"additionalProperties": ref("CodexMCPServer"),
			},
		}),
		"CodexMCPServer": object("A self-contained stdio or HTTP MCP transport. Set exactly one of command or url.", nil, map[string]any{
			"command":           map[string]any{"type": "string"},
			"args":              stringArray("Stdio server arguments."),
			"url":               map[string]any{"type": "string", "format": "uri"},
			"bearerTokenEnvVar": map[string]any{"type": "string", "description": "Environment variable containing an HTTP bearer token."},
		}),
		"CapabilitiesResponse": object("Compiler metadata.", []string{"version", "capabilities", "targets"}, map[string]any{
			"version":      map[string]any{"type": "string"},
			"capabilities": stringArray("Compiler features."),
			"targets":      stringArray("Supported target identifiers."),
		}),
		"TargetsResponse": object("Supported targets.", []string{"targets"}, map[string]any{"targets": stringArray("Supported target identifiers.")}),
		"ErrorResponse":   object("Request failure.", []string{"error"}, map[string]any{"error": map[string]any{"type": "string"}}),
		"OpenAPISchema":   map[string]any{"type": "object"},
		"HealthResponse":  object("Service health.", []string{"status"}, map[string]any{"status": map[string]any{"type": "string", "const": "ok"}}),
	}
}

func object(description string, required []string, properties map[string]any) map[string]any {
	result := map[string]any{"type": "object", "description": description, "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		result["required"] = required
	}
	return result
}

func targetSchema() map[string]any {
	return map[string]any{"type": "string", "enum": rosetta.Targets()}
}

func modeSchema() map[string]any {
	return map[string]any{"type": "string", "enum": []string{rosetta.ModeStrict, rosetta.ModePermissive}, "default": rosetta.ModeStrict}
}

func arrayOf(name string) map[string]any {
	return map[string]any{"type": "array", "items": ref(name)}
}

func ref(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}
