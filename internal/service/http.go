package service

import (
	"encoding/json"
	"net/http"

	"rosetta/internal/api"
	"rosetta/internal/compiler"
)

// NewHandler builds the Rosetta HTTP service router.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/compile", compileHandler)
	mux.HandleFunc("POST /v1/check", checkHandler)
	mux.HandleFunc("POST /v1/explain", explainHandler)
	mux.HandleFunc("GET /v1/capabilities", capabilitiesHandler)
	mux.HandleFunc("GET /v1/targets", targetsHandler)
	mux.HandleFunc("GET /v1/openapi.json", openAPIHandler)
	mux.HandleFunc("GET /healthz", healthzHandler)
	return mux
}

func compileHandler(w http.ResponseWriter, r *http.Request) {
	var req api.CompileRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	output, err := compiler.Compile(req.Source, req.Target)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, api.CompileResponse{Output: output, Target: targetOrDefault(req.Target)})
}

func checkHandler(w http.ResponseWriter, r *http.Request) {
	var req api.CheckRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	errs := compiler.Check(req.Source)
	if len(errs) == 0 {
		writeJSON(w, http.StatusOK, api.CheckResponse{Valid: true})
		return
	}
	messages := make([]string, 0, len(errs))
	for _, err := range errs {
		messages = append(messages, err.Error())
	}
	writeJSON(w, http.StatusBadRequest, api.CheckResponse{Valid: false, Errors: messages})
}

func explainHandler(w http.ResponseWriter, r *http.Request) {
	var req api.ExplainRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	explanation, err := compiler.Explain(req.Source, req.Target)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, api.ExplainResponse{Explanation: explanation})
}

func capabilitiesHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, api.CapabilitiesResponse{Version: compiler.Version, Capabilities: compiler.Capabilities()})
}

func targetsHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, api.TargetsResponse{Targets: compiler.Targets()})
}

func openAPIHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(OpenAPISchema())
}

func healthzHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, api.ErrorResponse{Error: err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func targetOrDefault(target string) string {
	if target != "" {
		return target
	}
	return compiler.Targets()[0]
}
