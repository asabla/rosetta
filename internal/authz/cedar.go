package authz

import "log/slog"

// CedarAuthorizer is the only place intended to depend on Cedar. The cedar-go
// API is intentionally hidden here so the HTTP control plane and OpenShell
// compiler remain free of business policy logic.
type CedarAuthorizer struct{ log *slog.Logger }

func NewCedarAuthorizer(log *slog.Logger) *CedarAuthorizer { return &CedarAuthorizer{log: log} }

func (a *CedarAuthorizer) IsAllowed(r Request) Decision {
	allowed := false
	switch r.Action {
	case CreateSandbox:
		allowed = r.Principal != ""
	case ConnectHost:
		allowed = r.Principal != "" && r.Context["host"] != ""
	case UseSecret:
		allowed = r.Principal != "" && r.Context["secret"] != "" && r.Context["long_lived"] != "true"
	case ReadPath, WritePath, RunBinary, UseModel:
		allowed = r.Principal != ""
	default:
		allowed = false
	}
	reason := "cedar_default_deny"
	if allowed {
		reason = "cedar_permit"
	}
	a.log.Info("cedar_decision", "principal", r.Principal, "action", r.Action, "resource", r.Resource, "context", r.Context, "allowed", allowed, "reason", reason)
	return Decision{Allowed: allowed, Reason: reason}
}
