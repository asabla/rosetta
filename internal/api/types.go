package api

// CompileRequest is the request body for POST /v1/compile.
type CompileRequest struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// CompileResponse is the response body for POST /v1/compile.
type CompileResponse struct {
	Output string `json:"output"`
	Target string `json:"target"`
}

// CheckRequest is the request body for POST /v1/check.
type CheckRequest struct {
	Source string `json:"source"`
}

// CheckResponse is the response body for POST /v1/check.
type CheckResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// ExplainRequest is the request body for POST /v1/explain.
type ExplainRequest struct {
	Source string `json:"source"`
	Target string `json:"target,omitempty"`
}

// ExplainResponse is the response body for POST /v1/explain.
type ExplainResponse struct {
	Explanation string `json:"explanation"`
}

// CapabilitiesResponse is the response body for GET /v1/capabilities.
type CapabilitiesResponse struct {
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}

// TargetsResponse is the response body for GET /v1/targets.
type TargetsResponse struct {
	Targets []string `json:"targets"`
}

// ErrorResponse is returned when the service cannot complete a request.
type ErrorResponse struct {
	Error string `json:"error"`
}
