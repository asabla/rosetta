package rosetta

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

var targetContracts = []TargetContractInfo{
	{Target: TargetOpenShell, Version: "rosetta/openshell-policy-v1", Maturity: "supported"},
	{Target: TargetOpenCode, Version: "rosetta/opencode-permissions-v1", Maturity: "supported"},
	{Target: TargetCodex, Version: "rosetta/codex-permissions-v1beta", Maturity: "preview"},
	{Target: TargetClaude, Version: "rosetta/claude-code-settings-v1beta", Maturity: "preview"},
}

// TargetContracts returns a copy of the target output contracts implemented by
// this compiler version.
func TargetContracts() []TargetContractInfo {
	return append([]TargetContractInfo(nil), targetContracts...)
}

func targetContractVersion(target string) string {
	for _, contract := range targetContracts {
		if contract.Target == target {
			return contract.Version
		}
	}
	return ""
}

// validateArtifactContract is a defense-in-depth check over Rosetta's emitted
// target subset. It is intentionally narrower than each upstream schema.
func validateArtifactContract(artifact Artifact) error {
	switch artifact.Target {
	case TargetOpenShell:
		return validateOpenShellArtifact(artifact.Content)
	case TargetOpenCode:
		return validateOpenCodeArtifact(artifact.Content)
	case TargetCodex:
		return validateCodexArtifact(artifact.Content)
	case TargetClaude:
		return validateClaudeArtifact(artifact.Content)
	default:
		return fmt.Errorf("unsupported artifact target %q", artifact.Target)
	}
}

func validateOpenShellArtifact(content string) error {
	var policy struct {
		Version          int `yaml:"version"`
		FilesystemPolicy struct {
			IncludeWorkdir bool     `yaml:"include_workdir"`
			ReadOnly       []string `yaml:"read_only"`
			ReadWrite      []string `yaml:"read_write"`
		} `yaml:"filesystem_policy"`
		Landlock struct {
			Compatibility string `yaml:"compatibility"`
		} `yaml:"landlock"`
		Process struct {
			RunAsUser  string `yaml:"run_as_user"`
			RunAsGroup string `yaml:"run_as_group"`
		} `yaml:"process"`
		NetworkPolicies map[string]struct {
			Name      string `yaml:"name"`
			Endpoints []struct {
				Host        string `yaml:"host"`
				Port        int    `yaml:"port"`
				Path        string `yaml:"path"`
				Protocol    string `yaml:"protocol"`
				Enforcement string `yaml:"enforcement"`
				Access      string `yaml:"access"`
			} `yaml:"endpoints"`
			Binaries []struct {
				Path string `yaml:"path"`
			} `yaml:"binaries"`
		} `yaml:"network_policies"`
	}
	decoder := yaml.NewDecoder(strings.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&policy); err != nil {
		return fmt.Errorf("decode OpenShell policy: %w", err)
	}
	if policy.Version != 1 {
		return errors.New("OpenShell policy version must be 1")
	}
	if policy.Landlock.Compatibility != "hard_requirement" {
		return errors.New("OpenShell policy must require Landlock")
	}
	if policy.FilesystemPolicy.IncludeWorkdir {
		return errors.New("OpenShell policy must not grant an implicit writable workdir")
	}
	if policy.Process.RunAsUser == "" || policy.Process.RunAsGroup == "" || policy.Process.RunAsUser == "root" || policy.Process.RunAsUser == "0" || policy.Process.RunAsGroup == "root" || policy.Process.RunAsGroup == "0" {
		return errors.New("OpenShell policy must use an explicit non-root identity")
	}
	for _, value := range append(policy.FilesystemPolicy.ReadOnly, policy.FilesystemPolicy.ReadWrite...) {
		if !isPOSIXAbsolute(value) {
			return fmt.Errorf("OpenShell path %q is not absolute", value)
		}
	}
	for _, value := range policy.FilesystemPolicy.ReadWrite {
		if value == "/" {
			return errors.New("OpenShell policy must not make the filesystem root writable")
		}
	}
	for name, network := range policy.NetworkPolicies {
		if len(network.Endpoints) == 0 || len(network.Binaries) == 0 {
			return fmt.Errorf("OpenShell network policy %q requires endpoints and binaries", name)
		}
		for _, endpoint := range network.Endpoints {
			if endpoint.Enforcement != "enforce" {
				return fmt.Errorf("OpenShell network policy %q must enforce endpoints", name)
			}
		}
		for _, binary := range network.Binaries {
			if !isPOSIXAbsolute(binary.Path) {
				return fmt.Errorf("OpenShell network policy %q has a non-absolute binary", name)
			}
		}
	}
	return nil
}

func validateOpenCodeArtifact(content string) error {
	var document map[string]any
	if err := json.Unmarshal([]byte(content), &document); err != nil {
		return fmt.Errorf("decode OpenCode configuration: %w", err)
	}
	permissions, ok := document["permission"].(map[string]any)
	if !ok || permissions["*"] != "deny" {
		return errors.New("OpenCode permissions must begin from a global deny")
	}
	for key, value := range permissions {
		rules, ok := value.(map[string]any)
		if !ok || key == "*" {
			continue
		}
		if rules["*"] != "deny" {
			return fmt.Errorf("OpenCode granular permission %q requires a catch-all deny", key)
		}
	}
	return nil
}

func validateCodexArtifact(content string) error {
	var document map[string]any
	if err := toml.Unmarshal([]byte(content), &document); err != nil {
		return fmt.Errorf("decode Codex configuration: %w", err)
	}
	profile, ok := document["default_permissions"].(string)
	if !ok || profile == "" || strings.HasPrefix(profile, ":") {
		return errors.New("Codex output must select a generated custom permission profile")
	}
	profiles, ok := document["permissions"].(map[string]any)
	if !ok {
		return errors.New("Codex output is missing permission profiles")
	}
	entry, ok := profiles[profile].(map[string]any)
	if !ok {
		return errors.New("Codex default permission profile is undefined")
	}
	if _, inherited := entry["extends"]; inherited {
		return errors.New("Codex generated profile must not inherit broader filesystem access")
	}
	filesystem, ok := entry["filesystem"].(map[string]any)
	if !ok || filesystem[":minimal"] != "read" {
		return errors.New("Codex generated profile must grant only the minimal runtime baseline by default")
	}
	roots, ok := filesystem[":workspace_roots"].(map[string]any)
	if !ok || roots["."] != "deny" {
		return errors.New("Codex generated profile must deny workspace roots by default")
	}
	return nil
}

func validateClaudeArtifact(content string) error {
	var document map[string]any
	if err := json.Unmarshal([]byte(content), &document); err != nil {
		return fmt.Errorf("decode Claude Code settings: %w", err)
	}
	permissions, ok := document["permissions"].(map[string]any)
	if !ok || permissions["defaultMode"] != "dontAsk" || permissions["disableAutoMode"] != "disable" || permissions["disableBypassPermissionsMode"] != "disable" {
		return errors.New("Claude Code permissions must disable unapproved and bypass execution")
	}
	if allow, ok := permissions["allow"].([]any); ok {
		for _, rule := range allow {
			text, _ := rule.(string)
			if strings.HasPrefix(text, "Bash(") || strings.HasPrefix(text, "WebFetch(") {
				return errors.New("Claude Code output must not contain unsupported command or network grants")
			}
		}
	}
	sandbox, ok := document["sandbox"].(map[string]any)
	if !ok || sandbox["enabled"] != true || sandbox["failIfUnavailable"] != true || sandbox["allowUnsandboxedCommands"] != false || sandbox["autoAllowBashIfSandboxed"] != false {
		return errors.New("Claude Code sandbox must fail closed")
	}
	filesystem, ok := sandbox["filesystem"].(map[string]any)
	if !ok || !stringArrayContains(filesystem["denyRead"], "/") {
		return errors.New("Claude Code sandbox must deny reads from the filesystem root")
	}
	network, ok := sandbox["network"].(map[string]any)
	if !ok || stringArrayLength(network["allowedDomains"]) != 0 {
		return errors.New("Claude Code output must keep network access disabled")
	}
	return nil
}

func stringArrayContains(value any, expected string) bool {
	items, ok := value.([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		if item == expected {
			return true
		}
	}
	return false
}

func stringArrayLength(value any) int {
	items, ok := value.([]any)
	if !ok {
		return -1
	}
	return len(items)
}
