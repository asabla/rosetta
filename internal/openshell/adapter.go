package openshell

import (
	"context"
	"log/slog"
)

type Adapter interface {
	CreateSandbox(context.Context, string, []byte, bool) error
	UpdatePolicy(context.Context, string, []byte, bool) error
}

type LoggingAdapter struct{ Log *slog.Logger }

func (a LoggingAdapter) CreateSandbox(ctx context.Context, id string, policy []byte, dry bool) error {
	a.Log.Info("openshell_adapter_call", "operation", "create_sandbox", "sandbox_id", id, "dry_run", dry, "policy_bytes", len(policy))
	return nil
}
func (a LoggingAdapter) UpdatePolicy(ctx context.Context, id string, policy []byte, dry bool) error {
	a.Log.Info("openshell_adapter_call", "operation", "update_policy", "sandbox_id", id, "dry_run", dry, "policy_bytes", len(policy))
	return nil
}
