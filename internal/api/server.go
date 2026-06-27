package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/asabla/rosetta/internal/authz"
	"github.com/asabla/rosetta/internal/compiler"
	"github.com/asabla/rosetta/internal/openshell"
	"github.com/asabla/rosetta/internal/store"
	"github.com/asabla/rosetta/internal/validate"
	"github.com/google/uuid"
)

type Server struct {
	Store *store.Store
	Auth  authz.Authorizer
	Shell openshell.Adapter
	Dir   string
}

type createReq struct {
	Principal string `json:"principal"`
	DryRun    bool   `json:"dry_run"`
}
type netReq struct {
	Principal, Host, Binary string
	Port                    int
	DryRun                  bool `json:"dry_run"`
}
type secReq struct {
	Principal, Secret string
	LongLived         bool `json:"long_lived"`
	DryRun            bool `json:"dry_run"`
}

func (s *Server) Routes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("POST /sandboxes", s.create)
	m.HandleFunc("POST /sandboxes/{id}/capabilities/network", s.network)
	m.HandleFunc("POST /sandboxes/{id}/capabilities/secrets", s.secret)
	m.HandleFunc("GET /sandboxes/{id}/policy", s.policy)
	return m
}
func respond(w http.ResponseWriter, code int, v any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func (s *Server) emit(id string, grants []compiler.Grant) (string, string, []byte, error) {
	b, h, err := compiler.Compile(grants)
	if err != nil {
		return "", "", nil, err
	}
	_ = os.MkdirAll(s.Dir, 0755)
	p := filepath.Join(s.Dir, id+".yaml")
	return p, h, b, os.WriteFile(p, b, 0644)
}
func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	var q createReq
	_ = json.NewDecoder(r.Body).Decode(&q)
	d := s.Auth.IsAllowed(authz.Request{Principal: q.Principal, Action: authz.CreateSandbox, Resource: "sandbox"})
	if !d.Allowed {
		respond(w, 403, d)
		return
	}
	id := uuid.NewString()
	p, h, b, err := s.emit(id, nil)
	if err != nil {
		respond(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if !q.DryRun {
		if err = s.Store.Create(id, q.Principal, p, h, nil); err != nil {
			respond(w, 500, map[string]string{"error": err.Error()})
			return
		}
		err = s.Shell.CreateSandbox(r.Context(), id, b, false)
	} else {
		err = s.Shell.CreateSandbox(r.Context(), id, b, true)
	}
	if err != nil {
		respond(w, 500, map[string]string{"error": err.Error()})
		return
	}
	respond(w, 201, map[string]any{"id": id, "policy_hash": h, "dry_run": q.DryRun})
}
func (s *Server) network(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var q netReq
	_ = json.NewDecoder(r.Body).Decode(&q)
	host, err := validate.NormalizeHost(q.Host)
	if err != nil {
		respond(w, 400, map[string]string{"error": err.Error()})
		return
	}
	d := s.Auth.IsAllowed(authz.Request{Principal: q.Principal, Action: authz.ConnectHost, Resource: id, Context: map[string]string{"host": host}})
	if !d.Allowed {
		respond(w, 403, d)
		return
	}
	_, _, grants, err := s.Store.Get(id)
	if err != nil {
		respond(w, 404, map[string]string{"error": "sandbox not found"})
		return
	}
	grants = append(grants, compiler.Grant{Kind: "ConnectHost", Host: host, Port: q.Port, Binary: q.Binary})
	p, h, b, err := s.emit(id, grants)
	if err != nil {
		respond(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if !q.DryRun {
		_ = s.Store.Update(id, p, h, grants)
	}
	_ = s.Shell.UpdatePolicy(r.Context(), id, b, q.DryRun)
	respond(w, 200, map[string]any{"policy_hash": h, "dry_run": q.DryRun})
}
func (s *Server) secret(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var q secReq
	_ = json.NewDecoder(r.Body).Decode(&q)
	if strings.TrimSpace(q.Secret) == "" {
		respond(w, 400, map[string]string{"error": "secret is required"})
		return
	}
	ctx := map[string]string{"secret": q.Secret}
	if q.LongLived {
		ctx["long_lived"] = "true"
	}
	d := s.Auth.IsAllowed(authz.Request{Principal: q.Principal, Action: authz.UseSecret, Resource: id, Context: ctx})
	if !d.Allowed {
		respond(w, 403, d)
		return
	}
	respond(w, 202, map[string]any{"accepted": true, "dry_run": q.DryRun, "note": "secret capability authorized but not embedded in OpenShell YAML"})
}
func (s *Server) policy(w http.ResponseWriter, r *http.Request) {
	p, h, _, err := s.Store.Get(r.PathValue("id"))
	if err != nil {
		respond(w, 404, map[string]string{"error": "sandbox not found"})
		return
	}
	b, err := os.ReadFile(p)
	if err != nil {
		respond(w, 500, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("x-policy-hash", h)
	w.Header().Set("content-type", "application/yaml")
	_, _ = w.Write(b)
}
