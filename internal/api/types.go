package api

import "github.com/asabla/rosetta"

// CompileRequest is the request body for POST /v1/compile.
type CompileRequest = rosetta.CompileRequest

// CompileResponse is the response body for POST /v1/compile.
type CompileResponse = rosetta.CompileResult

// CheckRequest is the request body for POST /v1/check.
type CheckRequest = rosetta.CheckRequest

// CheckResponse is the response body for POST /v1/check.
type CheckResponse = rosetta.CheckResult

// ExplainRequest is the request body for POST /v1/explain.
type ExplainRequest = rosetta.ExplainRequest

// ExplainResponse is the response body for POST /v1/explain.
type ExplainResponse = rosetta.ExplainResult

// CapabilitiesRequest is the request body for GET /v1/capabilities.
type CapabilitiesRequest = rosetta.CapabilitiesRequest

// CapabilitiesResponse is the response body for GET /v1/capabilities.
type CapabilitiesResponse = rosetta.CapabilitiesResult

// Diagnostic describes a validation or translation message.
type Diagnostic = rosetta.Diagnostic

// Artifact describes a generated target artifact.
type Artifact = rosetta.Artifact

// TargetsResponse is the response body for GET /v1/targets.
type TargetsResponse struct {
	Targets []string `json:"targets"`
}

// ErrorResponse is returned when the service cannot complete a request.
type ErrorResponse struct {
	Error string `json:"error"`
}
