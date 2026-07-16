package service

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/asabla/rosetta"
	"github.com/asabla/rosetta/internal/api"
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
	var req rosetta.CompileRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := rosetta.Compile(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func checkHandler(w http.ResponseWriter, r *http.Request) {
	var req rosetta.CheckRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := rosetta.Check(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	status := http.StatusOK
	if !result.Valid {
		status = http.StatusBadRequest
	}
	writeJSON(w, status, result)
}

func explainHandler(w http.ResponseWriter, r *http.Request) {
	var req rosetta.ExplainRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := rosetta.Explain(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func capabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	result, err := rosetta.Capabilities(r.Context(), rosetta.CapabilitiesRequest{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func targetsHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, api.TargetsResponse{Targets: rosetta.Targets()})
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
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, errors.New("request body must contain exactly one JSON object"))
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
