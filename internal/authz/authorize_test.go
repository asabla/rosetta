package authz

import (
	"io"
	"log/slog"
	"testing"
)

func testAuthorizer(t *testing.T) *CedarAuthorizer {
	t.Helper()
	a, err := NewCedarAuthorizerFromFS(slog.New(slog.NewTextHandler(io.Discard, nil)), "../../cedar/schema.cedarschema", "../../cedar/policies")
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func baseInput(action Capability) CapabilityInput {
	return CapabilityInput{AgentID: "agent-1", Role: "code-reviewer", AssignedRepo: "rosetta", ApprovedTask: true, Action: action, Repo: "rosetta", ApprovalStatus: "approved"}
}

func TestCedarFilesValidateAgainstSchema(t *testing.T) {
	if err := ValidateCedarFiles("../../cedar/schema.cedarschema", "../../cedar/policies"); err != nil {
		t.Fatal(err)
	}
}

func TestAllowGitHubNetworkAccess(t *testing.T) {
	in := baseInput(ConnectHost)
	in.ResourceType, in.ResourceID, in.Host, in.HostGroups = "Host", "github.com", "github.com", []string{"github"}
	out := testAuthorizer(t).AuthorizeCapability(in)
	if out.Decision != "ALLOW" || len(out.PolicyIDs) == 0 {
		t.Fatalf("expected allow with policy id, got %+v", out)
	}
}

func TestDenyArbitraryInternetAccess(t *testing.T) {
	in := baseInput(ConnectHost)
	in.ResourceType, in.ResourceID, in.Host = "Host", "example.com", "example.com"
	out := testAuthorizer(t).AuthorizeCapability(in)
	if out.Decision != "DENY" {
		t.Fatalf("expected deny, got %+v", out)
	}
}

func TestAllowGitHubReadonlySecret(t *testing.T) {
	in := baseInput(UseSecret)
	in.ResourceType, in.ResourceID, in.Secret = "Secret", "github-readonly-token", "github-readonly-token"
	out := testAuthorizer(t).AuthorizeCapability(in)
	if out.Decision != "ALLOW" {
		t.Fatalf("expected allow, got %+v", out)
	}
}

func TestDenyProductionSecret(t *testing.T) {
	in := baseInput(UseSecret)
	in.ResourceType, in.ResourceID, in.Secret, in.ProductionSecret = "Secret", "prod-db-password", "prod-db-password", true
	in.Risk, in.HumanApprovalID = "", ""
	out := testAuthorizer(t).AuthorizeCapability(in)
	if out.Decision != "DENY" {
		t.Fatalf("expected deny, got %+v", out)
	}
}

func TestDenyWriteOutsideTempPaths(t *testing.T) {
	in := baseInput(WritePath)
	in.ResourceType, in.ResourceID, in.Path = "Path", "/workspace/rosetta/main.go", "/workspace/rosetta/main.go"
	out := testAuthorizer(t).AuthorizeCapability(in)
	if out.Decision != "DENY" {
		t.Fatalf("expected deny, got %+v", out)
	}
}

func TestDenyUnapprovedBinary(t *testing.T) {
	in := baseInput(RunBinary)
	in.ResourceType, in.ResourceID, in.Binary = "Binary", "/usr/bin/python3", "/usr/bin/python3"
	in.ApprovedBinaries = []string{"/usr/bin/git"}
	out := testAuthorizer(t).AuthorizeCapability(in)
	if out.Decision != "DENY" {
		t.Fatalf("expected deny, got %+v", out)
	}
}
