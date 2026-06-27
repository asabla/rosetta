package compiler

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/asabla/rosetta/internal/validate"
	"gopkg.in/yaml.v3"
)

type Grant struct {
	Kind, Host, Secret, Path, Mode, Binary, Model string
	Port                                          int
}

type Input struct {
	SandboxID        string
	TaskID           string
	AgentID          string
	CedarDecisionIDs []string
	Grants           []Grant
}

type Policy struct {
	Version    int                      `yaml:"version"`
	Filesystem *Filesystem              `yaml:"filesystem_policy"`
	Process    map[string]string        `yaml:"process"`
	Network    map[string]NetworkPolicy `yaml:"network_policies"`
}

type Filesystem struct {
	IncludeWorkdir bool     `yaml:"include_workdir"`
	ReadOnly       []string `yaml:"read_only"`
	ReadWrite      []string `yaml:"read_write"`
}

type NetworkPolicy struct {
	Name      string     `yaml:"name,omitempty"`
	Endpoints []Endpoint `yaml:"endpoints"`
	Binaries  []Binary   `yaml:"binaries"`
}

type Endpoint struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Protocol    string `yaml:"protocol,omitempty"`
	Enforcement string `yaml:"enforcement,omitempty"`
	Access      string `yaml:"access,omitempty"`
}

type Binary struct {
	Path string `yaml:"path"`
}

type metadata struct {
	SandboxID        string   `json:"sandbox_id,omitempty"`
	TaskID           string   `json:"task_id,omitempty"`
	AgentID          string   `json:"agent_id,omitempty"`
	PolicyHash       string   `json:"policy_hash"`
	CedarDecisionIDs []string `json:"cedar_decision_ids,omitempty"`
	Note             string   `json:"note"`
}

// Compile preserves the legacy call shape for existing callers. New code should
// use CompileInput so metadata can be carried in YAML comments without adding
// unsupported OpenShell policy fields.
func Compile(grants []Grant) ([]byte, string, error) {
	return CompileInput(Input{Grants: grants})
}

func CompileInput(in Input) ([]byte, string, error) {
	policy, err := buildPolicy(in.Grants)
	if err != nil {
		return nil, "", err
	}
	body, err := yaml.Marshal(policy)
	if err != nil {
		return nil, "", err
	}
	h := sha256.Sum256(body)
	hash := fmt.Sprintf("sha256:%x", h)
	meta := metadata{SandboxID: in.SandboxID, TaskID: in.TaskID, AgentID: in.AgentID, PolicyHash: hash, CedarDecisionIDs: sortedStrings(in.CedarDecisionIDs), Note: "metadata is stored in YAML comments because OpenShell policy schema only documents version, filesystem_policy, landlock, process, and network_policies as top-level fields"}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, "", err
	}
	var out bytes.Buffer
	out.WriteString("# rosetta_metadata: ")
	out.Write(metaBytes)
	out.WriteByte('\n')
	out.Write(body)
	return out.Bytes(), hash, nil
}

func buildPolicy(grants []Grant) (Policy, error) {
	p := Policy{Version: 1, Filesystem: &Filesystem{IncludeWorkdir: false, ReadOnly: []string{}, ReadWrite: []string{}}, Process: map[string]string{"run_as_user": "sandbox", "run_as_group": "sandbox"}, Network: map[string]NetworkPolicy{}}
	readPaths := map[string]struct{}{}
	writePaths := map[string]struct{}{}
	networkKeys := map[string]struct{}{}

	ordered := append([]Grant(nil), grants...)
	sort.Slice(ordered, func(i, j int) bool { return grantSortKey(ordered[i]) < grantSortKey(ordered[j]) })
	for _, g := range ordered {
		switch g.Kind {
		case "ReadPath":
			path, err := validate.SafePath(g.Path)
			if err != nil {
				return p, err
			}
			if _, exists := writePaths[path]; exists {
				return p, fmt.Errorf("conflicting filesystem capabilities for %s", path)
			}
			readPaths[path] = struct{}{}
		case "WritePath":
			path, err := validate.SafePath(g.Path)
			if err != nil {
				return p, err
			}
			if _, exists := readPaths[path]; exists {
				return p, fmt.Errorf("conflicting filesystem capabilities for %s", path)
			}
			writePaths[path] = struct{}{}
		case "ConnectHost":
			host, err := validate.NormalizeHost(g.Host)
			if err != nil {
				return p, err
			}
			if g.Port <= 0 || g.Port > 65535 {
				return p, errors.New("network capabilities require an explicit valid TCP port")
			}
			if strings.TrimSpace(g.Binary) == "" {
				return p, errors.New("network capabilities require an explicit binary")
			}
			key := fmt.Sprintf("%s|%d|%s", host, g.Port, g.Binary)
			if _, exists := networkKeys[key]; exists {
				return p, fmt.Errorf("duplicate network capability for %s:%d and %s", host, g.Port, g.Binary)
			}
			networkKeys[key] = struct{}{}
			policyKey := fmt.Sprintf("connect_%03d_%s_%d", len(p.Network), sanitizeKey(host), g.Port)
			p.Network[policyKey] = NetworkPolicy{Name: "cedar-granted-host", Endpoints: []Endpoint{{Host: host, Port: g.Port, Enforcement: "enforce", Access: "full"}}, Binaries: []Binary{{Path: g.Binary}}}
		case "UseSecret", "RunBinary", "UseModel":
			// Not represented in the documented OpenShell policy schema. These grants
			// remain in Rosetta metadata/audit records and are intentionally not emitted
			// as OpenShell fields.
		default:
			return p, fmt.Errorf("unsupported grant kind %q", g.Kind)
		}
	}
	p.Filesystem.ReadOnly = sortedKeys(readPaths)
	p.Filesystem.ReadWrite = sortedKeys(writePaths)
	return p, nil
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func grantSortKey(g Grant) string {
	return fmt.Sprintf("%s|%s|%05d|%s|%s", g.Kind, g.Host, g.Port, g.Path, g.Binary)
}

func sanitizeKey(s string) string {
	s = strings.NewReplacer(".", "_", ":", "_", "/", "_", "-", "_").Replace(s)
	return strings.Trim(s, "_")
}
