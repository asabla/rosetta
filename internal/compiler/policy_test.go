package compiler

import (
	"strings"
	"testing"
)

func TestStableHashForSameInput(t *testing.T) {
	in := Input{SandboxID: "s1", TaskID: "t1", AgentID: "a1", CedarDecisionIDs: []string{"p2", "p1"}, Grants: []Grant{{Kind: "ConnectHost", Host: "GitHub.COM.", Port: 443, Binary: "/usr/bin/git"}, {Kind: "ReadPath", Path: "/tmp/repo"}}}
	_, h1, err := CompileInput(in)
	if err != nil {
		t.Fatal(err)
	}
	_, h2, err := CompileInput(in)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatalf("hash changed for same input: %s != %s", h1, h2)
	}
}

func TestChangedCapabilityChangesHash(t *testing.T) {
	_, h1, err := CompileInput(Input{Grants: []Grant{{Kind: "ConnectHost", Host: "github.com", Port: 443, Binary: "/usr/bin/git"}}})
	if err != nil {
		t.Fatal(err)
	}
	_, h2, err := CompileInput(Input{Grants: []Grant{{Kind: "ConnectHost", Host: "github.com", Port: 8443, Binary: "/usr/bin/git"}}})
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h2 {
		t.Fatal("hash did not change after capability changed")
	}
}

func TestDenyByDefaultEmptyPolicy(t *testing.T) {
	b, h, err := CompileInput(Input{SandboxID: "s1"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{"# rosetta_metadata:", "policy_hash", "version: 1", "include_workdir: false", "read_only: []", "read_write: []", "network_policies: {}"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in\n%s", want, s)
		}
	}
	if !strings.HasPrefix(h, "sha256:") {
		t.Fatal(h)
	}
}

func TestValidGitHubOnlyNetworkPolicy(t *testing.T) {
	b, _, err := CompileInput(Input{CedarDecisionIDs: []string{"code_reviewer_github_hosts"}, Grants: []Grant{{Kind: "ConnectHost", Host: "GitHub.COM.", Port: 443, Binary: "/usr/bin/git"}}})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{"network_policies:", "host: github.com", "port: 443", "enforcement: enforce", "access: full", "path: /usr/bin/git"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in\n%s", want, s)
		}
	}
}

func TestValidRepoReadWriteSplit(t *testing.T) {
	b, _, err := CompileInput(Input{Grants: []Grant{{Kind: "ReadPath", Path: "/tmp/repo"}, {Kind: "WritePath", Path: "/tmp/repo/.agent-tmp"}}})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{"read_only:", "- /tmp/repo", "read_write:", "- /tmp/repo/.agent-tmp"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in\n%s", want, s)
		}
	}
}

func TestInvalidPathRejection(t *testing.T) {
	if _, _, err := CompileInput(Input{Grants: []Grant{{Kind: "ReadPath", Path: "/etc/passwd"}}}); err == nil {
		t.Fatal("expected unsafe path rejection")
	}
}

func TestInvalidHostRejection(t *testing.T) {
	if _, _, err := CompileInput(Input{Grants: []Grant{{Kind: "ConnectHost", Host: "*.github.com", Port: 443, Binary: "/usr/bin/git"}}}); err == nil {
		t.Fatal("expected wildcard host rejection")
	}
}

func TestExplicitPortRequired(t *testing.T) {
	if _, _, err := CompileInput(Input{Grants: []Grant{{Kind: "ConnectHost", Host: "github.com", Binary: "/usr/bin/git"}}}); err == nil {
		t.Fatal("expected explicit port rejection")
	}
}

func TestDuplicateConflictingCapabilities(t *testing.T) {
	if _, _, err := CompileInput(Input{Grants: []Grant{{Kind: "ReadPath", Path: "/tmp/repo"}, {Kind: "WritePath", Path: "/tmp/repo"}}}); err == nil {
		t.Fatal("expected read/write conflict rejection")
	}
}
