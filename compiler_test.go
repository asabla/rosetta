package rosetta

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

const testPolicy = `permit(principal, action, resource)
when { principal.roles.contains("developer") };

forbid(principal, action, resource)
when { resource.selector == "/workspace/secret" };`

func testCatalog() Catalog {
	return Catalog{
		Version:   CatalogVersion,
		Principal: EntityRef{ID: "agent", Roles: []string{"developer"}},
		Capabilities: []Capability{
			{ID: "src-read", Kind: KindFilesystem, Action: "read", Selector: "/workspace/src"},
			{ID: "secret", Kind: KindFilesystem, Action: "read", Selector: "/workspace/secret"},
			{ID: "src-write", Kind: KindFilesystem, Action: "write", Selector: "/workspace/src"},
			{ID: "git-status", Kind: KindCommand, Action: "execute", Selector: "git status"},
			{ID: "read-tool", Kind: KindTool, Action: "use", Selector: "Read", Server: "filesystem"},
			{ID: "github", Kind: KindNetwork, Action: "connect", Selector: "api.github.com", Port: 443, Protocol: "rest", Access: "read-only", Binaries: []string{"/usr/bin/gh"}, Targets: []string{TargetOpenShell}},
		},
	}
}

func testOptions() TargetOptions {
	return TargetOptions{Codex: CodexOptions{MCPServers: map[string]CodexMCPServer{
		"filesystem": {Command: "mcp-server-filesystem", Args: []string{"/workspace/src"}},
	}}}
}

func TestCompilePreservesCedarForbid(t *testing.T) {
	result, err := Compile(context.Background(), CompileRequest{Source: testPolicy, Target: TargetOpenShell, Catalog: testCatalog(), Mode: ModePermissive})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	for _, decision := range result.Decisions {
		if decision.CapabilityID == "secret" && decision.Allowed {
			t.Fatal("forbid must override permit")
		}
	}
	if strings.Contains(result.Output, "/workspace/secret") {
		t.Fatal("denied path leaked into OpenShell artifact")
	}
}

func TestCompileTargetsAreDeterministicAndRestrictive(t *testing.T) {
	tests := []struct {
		target string
		want   []string
	}{
		{TargetOpenShell, []string{"version: 1", `compatibility: "hard_requirement"`, "enforcement: enforce"}},
		{TargetOpenCode, []string{`"*": "deny"`, `"bash"`, `"git status": "allow"`}},
		{TargetCodex, []string{`default_permissions = "rosetta"`, `"." = "deny"`, `enabled_tools = ["Read"]`}},
		{TargetClaude, []string{`"defaultMode": "dontAsk"`, `"failIfUnavailable": true`, `"allowUnsandboxedCommands": false`}},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			mode := ModeStrict
			catalog := testCatalog()
			var options TargetOptions
			if tt.target == TargetOpenShell || tt.target == TargetCodex || tt.target == TargetClaude {
				mode = ModePermissive
			}
			if tt.target == TargetCodex {
				options = testOptions()
			}
			first, err := Compile(context.Background(), CompileRequest{Source: testPolicy, Target: tt.target, Catalog: catalog, Mode: mode, Options: options})
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			second, err := Compile(context.Background(), CompileRequest{Source: testPolicy, Target: tt.target, Catalog: catalog, Mode: mode, Options: options})
			if err != nil {
				t.Fatalf("compile twice: %v", err)
			}
			if first.Output != second.Output {
				t.Fatal("artifact is not deterministic")
			}
			for _, want := range tt.want {
				if !strings.Contains(first.Output, want) {
					t.Fatalf("artifact missing %q:\n%s", want, first.Output)
				}
			}
		})
	}
}

func TestCompileMetadataIsDeterministicAndContentAddressed(t *testing.T) {
	req := CompileRequest{Source: testPolicy, Target: TargetOpenCode, Catalog: testCatalog()}
	first, err := Compile(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Compile(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if first.Metadata != second.Metadata {
		t.Fatalf("metadata is not deterministic: %#v != %#v", first.Metadata, second.Metadata)
	}
	if first.Metadata.CompilerVersion != Version || first.Metadata.CatalogVersion != CatalogVersion {
		t.Fatalf("unexpected compiler metadata: %#v", first.Metadata)
	}
	if first.Metadata.TargetContractVersion == "" || len(first.Metadata.InputSHA256) != 64 || len(first.Metadata.ArtifactSHA256) != 64 {
		t.Fatalf("incomplete compiler metadata: %#v", first.Metadata)
	}
	changed := req
	changed.Source += "\n"
	third, err := Compile(context.Background(), changed)
	if err != nil {
		t.Fatal(err)
	}
	if third.Metadata.InputSHA256 == first.Metadata.InputSHA256 {
		t.Fatal("input digest did not change with source bytes")
	}
	if third.Metadata.ArtifactSHA256 != first.Metadata.ArtifactSHA256 {
		t.Fatal("semantically equivalent source changed the deterministic artifact")
	}
}

func TestCapabilitiesDiscoverVersionedTargetContracts(t *testing.T) {
	result, err := Capabilities(context.Background(), CapabilitiesRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.TargetContracts) != len(Targets()) {
		t.Fatalf("got %d target contracts for %d targets", len(result.TargetContracts), len(Targets()))
	}
	for _, contract := range result.TargetContracts {
		if contract.Target == "" || contract.Version == "" || (contract.Maturity != "supported" && contract.Maturity != "preview") {
			t.Fatalf("invalid target contract: %#v", contract)
		}
	}
}

func TestOpenShellRejectsUncataloguedAndFailOpenOptions(t *testing.T) {
	catalog := Catalog{
		Version:      CatalogVersion,
		Principal:    EntityRef{ID: "agent"},
		Capabilities: []Capability{{ID: "workspace", Kind: KindFilesystem, Action: "write", Selector: "/workspace"}},
	}
	for _, options := range []OpenShellOptions{
		{IncludeWorkdir: true},
		{LandlockCompatibility: "best_effort"},
	} {
		_, err := Compile(context.Background(), CompileRequest{
			Source: "permit(principal, action, resource);", Target: TargetOpenShell, Catalog: catalog,
			Options: TargetOptions{OpenShell: options},
		})
		if err == nil {
			t.Fatalf("expected unsafe OpenShell options %#v to be rejected", options)
		}
	}
}

func TestStrictModeChecksCompleteNetworkSemantics(t *testing.T) {
	catalog := Catalog{
		Version:   CatalogVersion,
		Principal: EntityRef{ID: "agent"},
		Capabilities: []Capability{{
			ID: "network", Kind: KindNetwork, Action: "connect", Selector: "api.example.com", Port: 443,
			Protocol: "rest", Access: "read-only", Binaries: []string{"/usr/bin/curl"},
		}},
	}
	_, err := Compile(context.Background(), CompileRequest{
		Source: "permit(principal, action, resource);", Target: TargetOpenCode, Catalog: catalog,
	})
	if err == nil || !strings.Contains(err.Error(), "executable") {
		t.Fatalf("expected field-level representability error, got %v", err)
	}
}

func TestPermissiveModeSafelyOmitsUnrepresentableCapability(t *testing.T) {
	catalog := Catalog{
		Version:      CatalogVersion,
		Principal:    EntityRef{ID: "agent"},
		Capabilities: []Capability{{ID: "network", Kind: KindNetwork, Action: "connect", Selector: "api.example.com", Port: 443}},
	}
	result, err := Compile(context.Background(), CompileRequest{
		Source: "permit(principal, action, resource);", Target: TargetClaude, Mode: ModePermissive, Catalog: catalog,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Output, "api.example.com") {
		t.Fatal("unrepresentable network capability was rendered")
	}
	found := false
	for _, diagnostic := range result.Diagnostics {
		found = found || diagnostic.Code == "capability_omitted"
	}
	if !found {
		t.Fatal("expected capability_omitted diagnostic")
	}
}

func TestCodexProfileDoesNotInheritBroadReadAccess(t *testing.T) {
	catalog := Catalog{
		Version:      CatalogVersion,
		Principal:    EntityRef{ID: "agent"},
		Capabilities: []Capability{{ID: "workspace", Kind: KindFilesystem, Action: "read", Selector: "/workspace/src"}},
	}
	result, err := Compile(context.Background(), CompileRequest{
		Source: "permit(principal, action, resource);", Target: TargetCodex, Catalog: catalog,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Output, `extends = ":read-only"`) {
		t.Fatal("generated Codex profile must not inherit broad read access")
	}
}

func TestCompileRejectsAllowedCapabilityOverlappingDeny(t *testing.T) {
	tests := []struct {
		name         string
		capabilities []Capability
		policy       string
		target       string
	}{
		{
			name: "filesystem child deny",
			capabilities: []Capability{
				{ID: "workspace", Kind: KindFilesystem, Action: "write", Selector: "/workspace"},
				{ID: "secret", Kind: KindFilesystem, Action: "read", Selector: "/workspace/secret"},
			},
			policy: `permit(principal, action, resource) unless { resource.selector == "/workspace/secret" };`,
			target: TargetOpenShell,
		},
		{
			name: "command wildcard deny",
			capabilities: []Capability{
				{ID: "git", Kind: KindCommand, Action: "execute", Selector: "git *"},
				{ID: "push", Kind: KindCommand, Action: "execute", Selector: "git push"},
			},
			policy: `permit(principal, action, resource) unless { resource.selector == "git push" };`,
			target: TargetOpenCode,
		},
		{
			name: "intersecting command wildcards",
			capabilities: []Capability{
				{ID: "forced", Kind: KindCommand, Action: "execute", Selector: "git * --force"},
				{ID: "push", Kind: KindCommand, Action: "execute", Selector: "git push *"},
			},
			policy: `permit(principal, action, resource) unless { resource.selector == "git push *" };`,
			target: TargetOpenCode,
		},
		{
			name: "intersecting question wildcard",
			capabilities: []Capability{
				{ID: "git", Kind: KindCommand, Action: "execute", Selector: "git pu?h"},
				{ID: "push", Kind: KindCommand, Action: "execute", Selector: "git push"},
			},
			policy: `permit(principal, action, resource) unless { resource.selector == "git push" };`,
			target: TargetOpenCode,
		},
		{
			name: "intersecting network paths",
			capabilities: []Capability{
				{ID: "admin", Kind: KindNetwork, Action: "connect", Selector: "api.example.com", Port: 443, Path: "/v1/*/admin", Protocol: "rest", Access: "read-only", Binaries: []string{"/usr/bin/curl"}},
				{ID: "users", Kind: KindNetwork, Action: "connect", Selector: "api.example.com", Port: 443, Path: "/v1/users/*", Protocol: "rest", Access: "read-only", Binaries: []string{"/usr/bin/curl"}},
			},
			policy: `permit(principal, action, resource) unless { resource has path && resource.path == "/v1/users/*" };`,
			target: TargetOpenShell,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile(context.Background(), CompileRequest{
				Source: tt.policy, Target: tt.target,
				Catalog: Catalog{Version: CatalogVersion, Principal: EntityRef{ID: "agent"}, Capabilities: tt.capabilities},
			})
			if err == nil || !strings.Contains(err.Error(), "overlaps denied capability") {
				t.Fatalf("expected overlap rejection, got %v", err)
			}
		})
	}
}

func TestCompileAllowsDisjointCommandPatterns(t *testing.T) {
	result, err := Compile(context.Background(), CompileRequest{
		Source: `permit(principal, action, resource) unless { resource.selector == "npm publish *" };`,
		Target: TargetOpenCode,
		Catalog: Catalog{
			Version:   CatalogVersion,
			Principal: EntityRef{ID: "agent"},
			Capabilities: []Capability{
				{ID: "git", Kind: KindCommand, Action: "execute", Selector: "git * --force"},
				{ID: "publish", Kind: KindCommand, Action: "execute", Selector: "npm publish *"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Output, `"git * --force": "allow"`) || strings.Contains(result.Output, `"npm publish *": "allow"`) {
		t.Fatalf("unexpected OpenCode output:\n%s", result.Output)
	}
}

func TestGlobPatternsOverlap(t *testing.T) {
	tests := []struct {
		left  string
		right string
		want  bool
	}{
		{"git * --force", "git push *", true},
		{"git pu?h", "git push", true},
		{"git * --force", "npm publish *", false},
		{"ab*cd", "ab*ef", false},
		{"a*b*c", "ab*d*c", true},
		{"*", "", true},
		{"?", "", false},
		{"å?", "åb", true},
	}
	for _, tt := range tests {
		t.Run(tt.left+"|"+tt.right, func(t *testing.T) {
			budget := overlapBudget{remaining: MaxDecisionOverlapWork}
			got, err := globPatternsOverlap(context.Background(), tt.left, tt.right, &budget)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("globPatternsOverlap(%q, %q) = %v, want %v", tt.left, tt.right, got, tt.want)
			}
		})
	}
}

func TestDecisionIsolationHasDeterministicWorkLimit(t *testing.T) {
	const perDecision = 501
	capabilities := make([]Capability, 0, 2*perDecision)
	decisions := make([]Decision, 0, 2*perDecision)
	for i := range 2 * perDecision {
		id := fmt.Sprintf("tool-%04d", i)
		capabilities = append(capabilities, Capability{ID: id, Kind: KindTool, Action: "use", Selector: id})
		decisions = append(decisions, Decision{CapabilityID: id, Allowed: i < perDecision})
	}
	err := validateDecisionIsolation(context.Background(), capabilities, decisions)
	if err == nil || !strings.Contains(err.Error(), "work limit") {
		t.Fatalf("expected deterministic work-limit error, got %v", err)
	}
}

func TestDecisionIsolationHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := validateDecisionIsolation(ctx,
		[]Capability{
			{ID: "allowed", Kind: KindCommand, Action: "execute", Selector: "git *"},
			{ID: "denied", Kind: KindCommand, Action: "execute", Selector: "git push"},
		},
		[]Decision{{CapabilityID: "allowed", Allowed: true}, {CapabilityID: "denied"}},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestStrictModeRejectsUnsupportedAllowedCapability(t *testing.T) {
	_, err := Compile(context.Background(), CompileRequest{Source: "permit(principal, action, resource);", Target: TargetCodex, Catalog: testCatalog(), Options: testOptions()})
	if err == nil || !strings.Contains(err.Error(), "cannot safely represent") {
		t.Fatalf("expected representability error, got %v", err)
	}
}

func TestCodexToolRequiresSelfContainedMCPTransport(t *testing.T) {
	catalog := Catalog{
		Version:      CatalogVersion,
		Principal:    EntityRef{ID: "agent"},
		Capabilities: []Capability{{ID: "tool", Kind: KindTool, Action: "use", Selector: "read_file", Server: "filesystem"}},
	}
	_, err := Compile(context.Background(), CompileRequest{
		Source:  "permit(principal, action, resource);",
		Target:  TargetCodex,
		Catalog: catalog,
	})
	if err == nil || !strings.Contains(err.Error(), "transport definition") {
		t.Fatalf("expected missing transport error, got %v", err)
	}
}

func TestCheckRejectsInvalidCedar(t *testing.T) {
	result, err := Check(context.Background(), CheckRequest{Source: "permit("})
	if err != nil {
		t.Fatal(err)
	}
	if result.Valid || len(result.Diagnostics) == 0 {
		t.Fatalf("expected invalid result, got %#v", result)
	}
}

func TestCatalogRejectsTraversalAndDuplicates(t *testing.T) {
	catalog := testCatalog()
	catalog.Capabilities[0].Selector = "../secret"
	if err := validateCatalog(catalog); err == nil {
		t.Fatal("expected traversal error")
	}
	catalog = testCatalog()
	catalog.Capabilities[1].ID = catalog.Capabilities[0].ID
	if err := validateCatalog(catalog); err == nil {
		t.Fatal("expected duplicate id error")
	}
}

func TestDeniedUnsupportedCapabilityIsSafelyOmitted(t *testing.T) {
	catalog := Catalog{
		Version:      CatalogVersion,
		Principal:    EntityRef{ID: "agent"},
		Capabilities: []Capability{{ID: "command", Kind: KindCommand, Action: "execute", Selector: "git status"}},
	}
	result, err := Compile(context.Background(), CompileRequest{
		Source:  "forbid(principal, action, resource);",
		Target:  TargetCodex,
		Catalog: catalog,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Decisions) != 1 || result.Decisions[0].Allowed {
		t.Fatalf("expected explicit deny, got %#v", result.Decisions)
	}
}

func TestCatalogRejectsAmbiguousSelectors(t *testing.T) {
	tests := []Capability{
		{ID: "glob", Kind: KindFilesystem, Action: "read", Selector: "/workspace/**"},
		{ID: "tool-glob", Kind: KindTool, Action: "use", Selector: "mcp__github__*"},
		{ID: "host", Kind: KindNetwork, Action: "connect", Selector: "*", Port: 443},
		{ID: "control", Kind: KindCommand, Action: "execute", Selector: "git status\nrm -rf /"},
	}
	for _, capability := range tests {
		catalog := Catalog{Version: CatalogVersion, Principal: EntityRef{ID: "agent"}, Capabilities: []Capability{capability}}
		if err := validateCatalog(catalog); err == nil {
			t.Errorf("expected %s selector to be rejected", capability.ID)
		}
	}
}

func TestCompileIsSafeForConcurrentSDKUse(t *testing.T) {
	const workers = 32
	var group sync.WaitGroup
	errors := make(chan error, workers)
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := Compile(context.Background(), CompileRequest{Source: testPolicy, Target: TargetOpenCode, Catalog: testCatalog()})
			errors <- err
		}()
	}
	group.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func FuzzCheckNeverPanics(f *testing.F) {
	f.Add("permit(principal, action, resource);")
	f.Add("forbid(principal, action, resource);")
	f.Add("")
	f.Fuzz(func(t *testing.T, source string) {
		_, _ = Check(context.Background(), CheckRequest{Source: source})
	})
}

func FuzzCompileNeverBroadensDeniedInput(f *testing.F) {
	f.Add("/workspace/src")
	f.Add("../secret")
	f.Add("")
	f.Fuzz(func(t *testing.T, selector string) {
		catalog := Catalog{
			Version:      CatalogVersion,
			Principal:    EntityRef{ID: "agent"},
			Capabilities: []Capability{{ID: "candidate", Kind: KindFilesystem, Action: "read", Selector: selector}},
		}
		result, _ := Compile(context.Background(), CompileRequest{
			Source:  "forbid(principal, action, resource);",
			Target:  TargetOpenCode,
			Catalog: catalog,
		})
		if result != nil && strings.Contains(result.Output, `"`+selector+`": "allow"`) {
			t.Fatalf("denied selector was rendered as allowed: %q", selector)
		}
	})
}

func FuzzGlobIntersectionNeverMissesWitness(f *testing.F) {
	f.Add("git * --force", "git push *", "git push --force")
	f.Add("a?c", "ab*", "abc")
	f.Add("npm *", "git *", "git status")
	f.Fuzz(func(t *testing.T, left, right, witness string) {
		if len(left) > 64 || len(right) > 64 || len(witness) > 64 || !testGlobMatch(left, witness) || !testGlobMatch(right, witness) {
			return
		}
		budget := overlapBudget{remaining: MaxDecisionOverlapWork}
		overlaps, err := globPatternsOverlap(context.Background(), left, right, &budget)
		if err != nil {
			t.Fatal(err)
		}
		if !overlaps {
			t.Fatalf("missed witness %q shared by %q and %q", witness, left, right)
		}
	})
}

func testGlobMatch(pattern, value string) bool {
	patternRunes := []rune(pattern)
	valueRunes := []rune(value)
	dp := make([][]bool, len(patternRunes)+1)
	for i := range dp {
		dp[i] = make([]bool, len(valueRunes)+1)
	}
	dp[0][0] = true
	for i := 1; i <= len(patternRunes); i++ {
		if patternRunes[i-1] == '*' {
			dp[i][0] = dp[i-1][0]
		}
		for j := 1; j <= len(valueRunes); j++ {
			switch patternRunes[i-1] {
			case '*':
				dp[i][j] = dp[i-1][j] || dp[i][j-1]
			case '?':
				dp[i][j] = dp[i-1][j-1]
			default:
				dp[i][j] = dp[i-1][j-1] && patternRunes[i-1] == valueRunes[j-1]
			}
		}
	}
	return dp[len(patternRunes)][len(valueRunes)]
}
