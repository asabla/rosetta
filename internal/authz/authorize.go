package authz

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	cedar "github.com/cedar-policy/cedar-go"
	expast "github.com/cedar-policy/cedar-go/x/exp/ast"
	"github.com/cedar-policy/cedar-go/x/exp/schema"
	"github.com/cedar-policy/cedar-go/x/exp/schema/validate"
)

type CapabilityInput struct {
	AgentID          string
	Role             string
	AssignedRepo     string
	ApprovedTask     bool
	ApprovedBinaries []string
	Action           Capability
	ResourceID       string
	ResourceType     string
	Repo             string
	Host             string
	HostGroups       []string
	Secret           string
	ProductionSecret bool
	Path             string
	Binary           string
	Model            string
	ApprovalStatus   string
	Risk             string
	HumanApprovalID  string
}

type CapabilityDecision struct {
	Decision  string   `json:"decision"`
	Reasons   []string `json:"reasons"`
	PolicyIDs []string `json:"policyIds"`
}

type CedarAuthorizer struct {
	log      *slog.Logger
	policies *cedar.PolicySet
}

func NewCedarAuthorizer(log *slog.Logger) *CedarAuthorizer {
	a, err := NewCedarAuthorizerFromFS(log, findRepoFile("cedar/schema.cedarschema"), findRepoFile("cedar/policies"))
	if err != nil {
		log.Error("cedar_load_failed", "error", err)
		return &CedarAuthorizer{log: log, policies: cedar.NewPolicySet()}
	}
	return a
}

func findRepoFile(rel string) string {
	for _, prefix := range []string{".", "..", "../..", "../../.."} {
		candidate := filepath.Join(prefix, rel)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return rel
}

func NewCedarAuthorizerFromFS(log *slog.Logger, schemaPath, policyDir string) (*CedarAuthorizer, error) {
	if err := ValidateCedarFiles(schemaPath, policyDir); err != nil {
		return nil, err
	}
	ps := cedar.NewPolicySet()
	files, err := filepath.Glob(filepath.Join(policyDir, "*.cedar"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var p cedar.Policy
		if err := p.UnmarshalCedar(b); err != nil {
			return nil, fmt.Errorf("parse %s: %w", f, err)
		}
		p.SetFilename(f)
		id := cedar.PolicyID(strings.TrimSuffix(filepath.Base(f), filepath.Ext(f)))
		ps.Add(id, &p)
	}
	return &CedarAuthorizer{log: log, policies: ps}, nil
}

func ValidateCedarFiles(schemaPath, policyDir string) error {
	b, err := os.ReadFile(schemaPath)
	if err != nil {
		return err
	}
	var s schema.Schema
	s.SetFilename(schemaPath)
	if err := s.UnmarshalCedar(b); err != nil {
		return fmt.Errorf("parse schema: %w", err)
	}
	resolvedSchema, err := s.Resolve()
	if err != nil {
		return fmt.Errorf("resolve schema: %w", err)
	}
	validator := validate.New(resolvedSchema, validate.WithStrict())
	files, err := filepath.Glob(filepath.Join(policyDir, "*.cedar"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no cedar policies in %s", policyDir)
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		var p cedar.Policy
		if err := p.UnmarshalCedar(b); err != nil {
			return fmt.Errorf("parse policy %s: %w", f, err)
		}
		policyID := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
		if err := validator.Policy(policyID, (*expast.Policy)(p.AST())); err != nil {
			return fmt.Errorf("validate policy %s: %w", f, err)
		}
	}
	return nil
}

func (a *CedarAuthorizer) AuthorizeCapability(in CapabilityInput) CapabilityDecision {
	if in.AgentID == "" {
		in.AgentID = "agent"
	}
	if in.ResourceType == "" {
		in.ResourceType = resourceTypeFor(in.Action)
	}
	if in.ResourceID == "" {
		in.ResourceID = resourceIDFor(in)
	}
	entities := buildEntities(in)
	req := cedar.Request{Principal: uid("Agent", in.AgentID), Action: uid("Action", string(in.Action)), Resource: uid(in.ResourceType, in.ResourceID), Context: contextRecord(in)}
	decision, diagnostic := a.policies.IsAuthorized(entities, req)
	out := CapabilityDecision{Decision: "DENY"}
	if decision == cedar.Allow {
		out.Decision = "ALLOW"
	}
	for _, r := range diagnostic.Reasons {
		pid := string(r.PolicyID)
		out.PolicyIDs = append(out.PolicyIDs, pid)
		out.Reasons = append(out.Reasons, pid)
	}
	for _, e := range diagnostic.Errors {
		out.Reasons = append(out.Reasons, e.String())
	}
	sort.Strings(out.PolicyIDs)
	sort.Strings(out.Reasons)
	a.log.Info("cedar_decision", "principal", in.AgentID, "action", in.Action, "resource_type", in.ResourceType, "resource", in.ResourceID, "decision", out.Decision, "policy_ids", out.PolicyIDs, "reasons", out.Reasons)
	return out
}

func (a *CedarAuthorizer) IsAllowed(r Request) Decision {
	in := CapabilityInput{AgentID: r.Principal, Role: "code-reviewer", AssignedRepo: r.Context["repo"], Action: r.Action, Repo: r.Context["repo"], ApprovalStatus: r.Context["approval_status"], Risk: r.Context["risk"], HumanApprovalID: r.Context["human_approval_id"]}
	if in.AssignedRepo == "" {
		in.AssignedRepo = "default"
		in.Repo = "default"
	}
	switch r.Action {
	case CreateSandbox:
		in.ResourceType, in.ResourceID = "Workspace", "workspace"
	case ConnectHost:
		in.Host, in.ResourceID, in.ResourceType, in.ApprovedTask, in.HostGroups = r.Context["host"], r.Context["host"], "Host", true, []string{"github"}
		if in.ApprovalStatus == "" {
			in.ApprovalStatus = "approved"
		}
	case UseSecret:
		in.Secret, in.ResourceID, in.ResourceType = r.Context["secret"], r.Context["secret"], "Secret"
		in.ProductionSecret = r.Context["production"] == "true"
	case ReadPath, WritePath:
		in.Path, in.ResourceID, in.ResourceType = r.Context["path"], r.Context["path"], "Path"
	case RunBinary:
		in.Binary, in.ResourceID, in.ResourceType = r.Context["binary"], r.Context["binary"], "Binary"
		in.ApprovedBinaries = []string{r.Context["binary"]}
	case UseModel:
		in.Model, in.ResourceID, in.ResourceType = r.Context["model"], r.Context["model"], "Model"
	}
	out := a.AuthorizeCapability(in)
	return Decision{Allowed: out.Decision == "ALLOW", Reason: strings.Join(out.Reasons, ",")}
}

func buildEntities(in CapabilityInput) cedar.EntityMap {
	m := cedar.EntityMap{}
	approved := make([]cedar.Value, 0, len(in.ApprovedBinaries))
	for _, b := range in.ApprovedBinaries {
		approved = append(approved, cedar.String(b))
	}
	m[uid("Agent", in.AgentID)] = cedar.Entity{UID: uid("Agent", in.AgentID), Attributes: rec(map[string]cedar.Value{"role": cedar.String(in.Role), "assigned_repo": cedar.String(in.AssignedRepo), "approved_task": cedar.Boolean(in.ApprovedTask), "approved_binaries": cedar.NewSet(approved...)})}
	for _, g := range in.HostGroups {
		m[uid("HostGroup", g)] = cedar.Entity{UID: uid("HostGroup", g)}
	}
	parents := []cedar.EntityUID{}
	for _, g := range in.HostGroups {
		parents = append(parents, uid("HostGroup", g))
	}
	if in.Host != "" {
		m[uid("Host", in.ResourceID)] = cedar.Entity{UID: uid("Host", in.ResourceID), Parents: cedar.NewEntityUIDSet(parents...), Attributes: rec(map[string]cedar.Value{"hostname": cedar.String(in.Host)})}
	}
	if in.Secret != "" || in.ResourceType == "Secret" {
		name := in.Secret
		if name == "" {
			name = in.ResourceID
		}
		m[uid("Secret", in.ResourceID)] = cedar.Entity{UID: uid("Secret", in.ResourceID), Attributes: rec(map[string]cedar.Value{"name": cedar.String(name), "production": cedar.Boolean(in.ProductionSecret)})}
	}
	if in.Path != "" || in.ResourceType == "Path" {
		m[uid("Path", in.ResourceID)] = cedar.Entity{UID: uid("Path", in.ResourceID), Attributes: rec(map[string]cedar.Value{"path": cedar.String(in.Path), "repo": cedar.String(in.Repo)})}
	}
	if in.Binary != "" || in.ResourceType == "Binary" {
		m[uid("Binary", in.ResourceID)] = cedar.Entity{UID: uid("Binary", in.ResourceID), Attributes: rec(map[string]cedar.Value{"path": cedar.String(in.Binary)})}
	}
	if in.Model != "" || in.ResourceType == "Model" {
		m[uid("Model", in.ResourceID)] = cedar.Entity{UID: uid("Model", in.ResourceID), Attributes: rec(map[string]cedar.Value{"name": cedar.String(in.Model)})}
	}
	if in.ResourceType == "Workspace" {
		m[uid("Workspace", in.ResourceID)] = cedar.Entity{UID: uid("Workspace", in.ResourceID)}
	}
	return m
}

func contextRecord(in CapabilityInput) cedar.Record {
	return rec(map[string]cedar.Value{"repo": cedar.String(in.Repo), "approval_status": cedar.String(in.ApprovalStatus), "risk": cedar.String(in.Risk), "human_approval_id": cedar.String(in.HumanApprovalID)})
}
func rec(m map[string]cedar.Value) cedar.Record {
	rm := cedar.RecordMap{}
	for k, v := range m {
		rm[cedar.String(k)] = v
	}
	return cedar.NewRecord(rm)
}
func uid(t, id string) cedar.EntityUID {
	return cedar.NewEntityUID(cedar.EntityType(t), cedar.String(id))
}
func resourceTypeFor(a Capability) string {
	switch a {
	case ConnectHost:
		return "Host"
	case UseSecret:
		return "Secret"
	case ReadPath, WritePath:
		return "Path"
	case RunBinary:
		return "Binary"
	case UseModel:
		return "Model"
	default:
		return "Workspace"
	}
}
func resourceIDFor(in CapabilityInput) string {
	switch in.Action {
	case ConnectHost:
		return in.Host
	case UseSecret:
		return in.Secret
	case ReadPath, WritePath:
		return in.Path
	case RunBinary:
		return in.Binary
	case UseModel:
		return in.Model
	default:
		return "workspace"
	}
}
