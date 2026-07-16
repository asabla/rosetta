package rosetta

import (
	"context"
	"errors"
	"fmt"
	"net"
	pathpkg "path"
	"slices"
	"sort"
	"strings"
	"sync"

	cedar "github.com/cedar-policy/cedar-go"
	expast "github.com/cedar-policy/cedar-go/x/exp/ast"
	cedarschema "github.com/cedar-policy/cedar-go/x/exp/schema"
	"github.com/cedar-policy/cedar-go/x/exp/schema/resolved"
	"github.com/cedar-policy/cedar-go/x/exp/schema/validate"
)

const CatalogVersion = "rosetta/v0.5"

const (
	MaxSourceBytes   = 2 << 20
	MaxCapabilities  = 10_000
	MaxSelectorBytes = 4096
)

var targets = []string{TargetOpenShell, TargetOpenCode, TargetCodex, TargetClaude}

var (
	schemaOnce     sync.Once
	resolvedSchema *resolved.Schema
	resolvedError  error
)

// Targets returns a copy of the stable target identifiers.
func Targets() []string { return append([]string(nil), targets...) }

func Capabilities(ctx context.Context, _ CapabilitiesRequest) (*CapabilitiesResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &CapabilitiesResult{
		Version:      Version,
		Capabilities: []string{"cedar-parse", "schema-validation", "authorize", "compile", "check", "explain", "deterministic-artifacts"},
		Targets:      Targets(),
	}, nil
}

// Check parses Cedar and validates every policy against the Rosetta schema.
func Check(ctx context.Context, req CheckRequest) (*CheckResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	mode, err := normalizeMode(req.Mode)
	if err != nil {
		return nil, err
	}
	_, diagnostics, err := parseAndValidate(req.Source, mode)
	if err != nil {
		diagnostic := Diagnostic{Severity: "error", Code: "cedar_invalid", Message: err.Error()}
		return &CheckResult{Valid: false, Diagnostics: append(diagnostics, diagnostic), Errors: []string{err.Error()}}, nil
	}
	return &CheckResult{Valid: true, Diagnostics: diagnostics}, nil
}

// Compile authorizes the requested catalog and renders a deterministic artifact.
func Compile(ctx context.Context, req CompileRequest) (*CompileResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	mode, err := normalizeMode(req.Mode)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(targets, req.Target) {
		return nil, fmt.Errorf("unsupported target %q", req.Target)
	}
	if err := validateCatalog(req.Catalog); err != nil {
		return nil, err
	}
	policies, diagnostics, err := parseAndValidate(req.Source, mode)
	if err != nil {
		return nil, err
	}
	decisions, selected, err := authorizeCatalog(ctx, policies, req.Catalog, req.Target)
	if err != nil {
		return nil, err
	}
	if err := validateDecisionIsolation(selected, decisions); err != nil {
		return nil, err
	}

	artifact, renderDiagnostics, err := render(req.Target, mode, selected, decisions, req.Options)
	if err != nil {
		return nil, err
	}
	diagnostics = append(diagnostics, renderDiagnostics...)
	return &CompileResult{
		Output:      artifact.Content,
		Target:      req.Target,
		Artifacts:   []Artifact{artifact},
		Decisions:   decisions,
		Diagnostics: diagnostics,
	}, nil
}

func validateDecisionIsolation(capabilities []Capability, decisions []Decision) error {
	allowed := make(map[string]bool, len(decisions))
	for _, decision := range decisions {
		allowed[decision.CapabilityID] = decision.Allowed
	}
	for i, left := range capabilities {
		for _, right := range capabilities[i+1:] {
			if allowed[left.ID] == allowed[right.ID] || !capabilitiesOverlap(left, right) {
				continue
			}
			permitted, denied := left, right
			if !allowed[left.ID] {
				permitted, denied = right, left
			}
			return fmt.Errorf("allowed capability %q overlaps denied capability %q; target output cannot preserve the deny", permitted.ID, denied.ID)
		}
	}
	return nil
}

func capabilitiesOverlap(left, right Capability) bool {
	if left.Kind != right.Kind {
		return false
	}
	switch left.Kind {
	case KindFilesystem:
		return pathContains(left.Selector, right.Selector) || pathContains(right.Selector, left.Selector)
	case KindTool:
		return left.Selector == right.Selector
	case KindCommand:
		return wildcardMatch(left.Selector, right.Selector) || wildcardMatch(right.Selector, left.Selector)
	case KindNetwork:
		if left.Port != right.Port {
			return false
		}
		hostsOverlap := wildcardMatch(left.Selector, right.Selector) || wildcardMatch(right.Selector, left.Selector)
		pathsOverlap := left.Path == "" || right.Path == "" || wildcardMatch(left.Path, right.Path) || wildcardMatch(right.Path, left.Path)
		return hostsOverlap && pathsOverlap
	default:
		return false
	}
}

func pathContains(parent, child string) bool {
	parent = strings.TrimSuffix(pathpkg.Clean(parent), "/")
	child = strings.TrimSuffix(pathpkg.Clean(child), "/")
	return parent == child || parent == "" || parent == "/" || strings.HasPrefix(child, parent+"/")
}

func wildcardMatch(pattern, value string) bool {
	rows := len(pattern) + 1
	cols := len(value) + 1
	dp := make([][]bool, rows)
	for i := range dp {
		dp[i] = make([]bool, cols)
	}
	dp[0][0] = true
	for i := 1; i < rows; i++ {
		if pattern[i-1] == '*' {
			dp[i][0] = dp[i-1][0]
		}
	}
	for i := 1; i < rows; i++ {
		for j := 1; j < cols; j++ {
			switch pattern[i-1] {
			case '*':
				dp[i][j] = dp[i-1][j] || dp[i][j-1]
			case '?':
				dp[i][j] = dp[i-1][j-1]
			default:
				dp[i][j] = dp[i-1][j-1] && pattern[i-1] == value[j-1]
			}
		}
	}
	return dp[len(pattern)][len(value)]
}

func Explain(ctx context.Context, req ExplainRequest) (*ExplainResult, error) {
	result, err := Compile(ctx, CompileRequest(req))
	if err != nil {
		return nil, err
	}
	allowed := 0
	for _, decision := range result.Decisions {
		if decision.Allowed {
			allowed++
		}
	}
	return &ExplainResult{
		Explanation: fmt.Sprintf("Cedar allowed %d of %d catalogued capabilities for %s; Rosetta rendered one fail-closed %s artifact.", allowed, len(result.Decisions), req.Catalog.Principal.ID, req.Target),
		Decisions:   result.Decisions,
		Diagnostics: result.Diagnostics,
	}, nil
}

func parseAndValidate(source, mode string) (*cedar.PolicySet, []Diagnostic, error) {
	if strings.TrimSpace(source) == "" {
		return nil, nil, errors.New("source is required")
	}
	if len(source) > MaxSourceBytes {
		return nil, nil, fmt.Errorf("source exceeds %d bytes", MaxSourceBytes)
	}
	policies, err := cedar.NewPolicySetFromBytes("policy.cedar", []byte(source))
	if err != nil {
		return nil, nil, fmt.Errorf("parse Cedar policy: %w", err)
	}
	policyCount := 0
	for range policies.All() {
		policyCount++
	}
	if policyCount == 0 {
		return nil, nil, errors.New("at least one Cedar policy is required")
	}
	resolved, err := loadSchema()
	if err != nil {
		return nil, nil, err
	}
	options := []validate.Option{validate.WithStrict()}
	var diagnostics []Diagnostic
	if mode == ModePermissive {
		diagnostics = append(diagnostics, permissiveDiagnostic())
	}
	validator := validate.New(resolved, options...)
	ids := make([]string, 0, policyCount)
	for id := range policies.All() {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	for _, id := range ids {
		policy := policies.Get(cedar.PolicyID(id))
		if err := validator.Policy(id, (*expast.Policy)(policy.AST())); err != nil {
			return nil, diagnostics, fmt.Errorf("validate Cedar policy: %w", err)
		}
	}
	return policies, diagnostics, nil
}

func loadSchema() (*resolved.Schema, error) {
	schemaOnce.Do(func() {
		var schema cedarschema.Schema
		schema.SetFilename("rosetta.cedarschema")
		if err := schema.UnmarshalCedar([]byte(CedarSchema)); err != nil {
			resolvedError = fmt.Errorf("parse embedded Cedar schema: %w", err)
			return
		}
		resolvedSchema, resolvedError = schema.Resolve()
		if resolvedError != nil {
			resolvedError = fmt.Errorf("resolve embedded Cedar schema: %w", resolvedError)
		}
	})
	return resolvedSchema, resolvedError
}

func authorizeCatalog(ctx context.Context, policies *cedar.PolicySet, catalog Catalog, target string) ([]Decision, []Capability, error) {
	principalType := catalog.Principal.Type
	if principalType == "" {
		principalType = "Rosetta::Principal"
	}
	principal := cedar.NewEntityUID(cedar.EntityType(principalType), cedar.String(catalog.Principal.ID))
	entities := cedar.EntityMap{}
	entities[principal] = cedar.Entity{
		UID:        principal,
		Parents:    cedar.NewEntityUIDSet(),
		Attributes: cedar.NewRecord(cedar.RecordMap{"roles": stringSet(catalog.Principal.Roles)}),
		Tags:       cedar.NewRecord(nil),
	}

	selected := make([]Capability, 0, len(catalog.Capabilities))
	for _, capability := range catalog.Capabilities {
		if len(capability.Targets) > 0 && !slices.Contains(capability.Targets, target) {
			continue
		}
		selected = append(selected, capability)
		uid := cedar.NewEntityUID("Rosetta::Capability", cedar.String(capability.ID))
		attrs := cedar.RecordMap{
			"kind":     cedar.String(capability.Kind),
			"selector": cedar.String(capability.Selector),
			"access":   cedar.String(capability.Access),
			"binaries": stringSet(capability.Binaries),
			"targets":  stringSet(capability.Targets),
		}
		if capability.Port != 0 {
			attrs["port"] = cedar.Long(capability.Port)
		}
		if capability.Protocol != "" {
			attrs["protocol"] = cedar.String(capability.Protocol)
		}
		if capability.Path != "" {
			attrs["path"] = cedar.String(capability.Path)
		}
		if capability.Server != "" {
			attrs["server"] = cedar.String(capability.Server)
		}
		entities[uid] = cedar.Entity{UID: uid, Parents: cedar.NewEntityUIDSet(), Attributes: cedar.NewRecord(attrs), Tags: cedar.NewRecord(nil)}
	}
	resolved, err := loadSchema()
	if err != nil {
		return nil, nil, err
	}
	validator := validate.New(resolved, validate.WithStrict())
	if err := validator.Entities(entities); err != nil {
		return nil, nil, fmt.Errorf("validate generated Cedar entities: %w", err)
	}

	decisions := make([]Decision, 0, len(selected))
	for _, capability := range selected {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		request := cedar.Request{
			Principal: principal,
			Action:    cedar.NewEntityUID("Rosetta::Action", cedar.String(capability.Action)),
			Resource:  cedar.NewEntityUID("Rosetta::Capability", cedar.String(capability.ID)),
			Context:   cedar.NewRecord(nil),
		}
		if err := validator.Request(request); err != nil {
			return nil, nil, fmt.Errorf("validate generated Cedar request for %q: %w", capability.ID, err)
		}
		decision, diagnostic := cedar.Authorize(policies, entities, request)
		if len(diagnostic.Errors) > 0 {
			return nil, nil, fmt.Errorf("authorize capability %q: %s", capability.ID, diagnostic.Errors[0].String())
		}
		policyIDs := make([]string, 0, len(diagnostic.Reasons))
		for _, reason := range diagnostic.Reasons {
			policyIDs = append(policyIDs, string(reason.PolicyID))
		}
		sort.Strings(policyIDs)
		decisions = append(decisions, Decision{CapabilityID: capability.ID, Allowed: decision == cedar.Allow, PolicyIDs: policyIDs})
	}
	return decisions, selected, nil
}

func validateCatalog(catalog Catalog) error {
	if catalog.Version != CatalogVersion {
		return fmt.Errorf("catalog version must be %q", CatalogVersion)
	}
	if catalog.Principal.ID == "" {
		return errors.New("catalog principal id is required")
	}
	if err := validatePlainString("catalog principal id", catalog.Principal.ID, 1024); err != nil {
		return err
	}
	if len(catalog.Capabilities) > MaxCapabilities {
		return fmt.Errorf("catalog exceeds %d capabilities", MaxCapabilities)
	}
	if catalog.Principal.Type != "" && catalog.Principal.Type != "Rosetta::Principal" {
		return errors.New("catalog principal type must be Rosetta::Principal")
	}
	seen := map[string]struct{}{}
	for i, capability := range catalog.Capabilities {
		prefix := fmt.Sprintf("catalog capability %d", i)
		if capability.ID == "" || capability.Selector == "" {
			return fmt.Errorf("%s requires id and selector", prefix)
		}
		if err := validatePlainString("capability id", capability.ID, 1024); err != nil {
			return fmt.Errorf("%s: %w", prefix, err)
		}
		if _, exists := seen[capability.ID]; exists {
			return fmt.Errorf("duplicate capability id %q", capability.ID)
		}
		seen[capability.ID] = struct{}{}
		if err := validateCapability(capability); err != nil {
			return fmt.Errorf("capability %q: %w", capability.ID, err)
		}
	}
	return nil
}

func validateCapability(capability Capability) error {
	if err := validatePlainString("selector", capability.Selector, MaxSelectorBytes); err != nil {
		return err
	}
	wantAction := map[string][]string{
		KindFilesystem: {"read", "write"},
		KindTool:       {"use"},
		KindCommand:    {"execute"},
		KindNetwork:    {"connect"},
	}
	actions, ok := wantAction[capability.Kind]
	if !ok {
		return fmt.Errorf("unsupported kind %q", capability.Kind)
	}
	if !slices.Contains(actions, capability.Action) {
		return fmt.Errorf("action %q is invalid for %s", capability.Action, capability.Kind)
	}
	for _, target := range capability.Targets {
		if !slices.Contains(targets, target) {
			return fmt.Errorf("unsupported target %q", target)
		}
	}
	if len(uniqueSorted(capability.Targets)) != len(capability.Targets) {
		return errors.New("targets must not contain duplicates")
	}
	if capability.Kind == KindTool && strings.ContainsAny(capability.Selector, "*?") {
		return errors.New("tool selector must name one exact tool")
	}
	if capability.Kind == KindFilesystem {
		if strings.ContainsAny(capability.Selector, "*?[\\") {
			return errors.New("filesystem selector must name a directory root without glob syntax")
		}
		clean := pathpkg.Clean(capability.Selector)
		if clean == "." && capability.Selector != "." {
			return errors.New("filesystem selector is empty after normalization")
		}
		for _, part := range strings.Split(capability.Selector, "/") {
			if part == ".." {
				return errors.New("filesystem selector must not contain traversal")
			}
		}
	}
	if capability.Kind == KindNetwork {
		if net.ParseIP(capability.Selector) == nil && !validHostname(capability.Selector) {
			return errors.New("network selector must be a hostname or IP address")
		}
		if capability.Port < 1 || capability.Port > 65535 {
			return errors.New("network port must be between 1 and 65535")
		}
		if capability.Path != "" {
			if err := validatePlainString("network path", capability.Path, MaxSelectorBytes); err != nil {
				return err
			}
			if !strings.HasPrefix(capability.Path, "/") || strings.ContainsAny(capability.Path, "?#") {
				return errors.New("network path must be an absolute path pattern without query or fragment")
			}
		}
	}
	return nil
}

func validatePlainString(name, value string, maxBytes int) error {
	if len(value) > maxBytes {
		return fmt.Errorf("%s exceeds %d bytes", name, maxBytes)
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("%s must not contain control characters", name)
		}
	}
	return nil
}

func validHostname(host string) bool {
	if len(host) == 0 || len(host) > 253 || strings.ContainsAny(host, "/\\ ") {
		return false
	}
	labels := strings.Split(host, ".")
	if host == "*" || host == "**" || len(labels) < 2 && strings.Contains(host, "*") {
		return false
	}
	for index, label := range labels {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		if index > 0 && strings.Contains(label, "*") {
			return false
		}
		for _, r := range label {
			if !(r == '-' || r == '*' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
				return false
			}
		}
	}
	return true
}

func stringSet(values []string) cedar.Set {
	items := make([]cedar.Value, 0, len(values))
	for _, value := range values {
		items = append(items, cedar.String(value))
	}
	return cedar.NewSet(items...)
}

func normalizeMode(mode string) (string, error) {
	if mode == "" {
		return ModeStrict, nil
	}
	if mode != ModeStrict && mode != ModePermissive {
		return "", fmt.Errorf("unsupported mode %q", mode)
	}
	return mode, nil
}

func permissiveDiagnostic() Diagnostic {
	return Diagnostic{Severity: "warning", Code: "permissive_mode", Message: "permissive mode may omit unsupported capabilities only when omission safely denies access", Recoverable: true}
}
