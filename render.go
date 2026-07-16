package rosetta

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func render(target, mode string, capabilities []Capability, decisions []Decision, options TargetOptions) (Artifact, []Diagnostic, error) {
	allowed := make(map[string]bool, len(decisions))
	for _, decision := range decisions {
		allowed[decision.CapabilityID] = decision.Allowed
	}
	selected := make([]Capability, 0, len(capabilities))
	for _, capability := range capabilities {
		if allowed[capability.ID] {
			selected = append(selected, capability)
		}
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].ID < selected[j].ID })

	supported := map[string]map[string]bool{
		TargetOpenShell: {KindFilesystem: true, KindNetwork: true},
		TargetOpenCode:  {KindFilesystem: true, KindTool: true, KindCommand: true, KindNetwork: true},
		TargetCodex:     {KindFilesystem: true, KindTool: true},
		TargetClaude:    {KindFilesystem: true, KindTool: true, KindCommand: true, KindNetwork: true},
	}
	var compatible []Capability
	var diagnostics []Diagnostic
	for _, capability := range selected {
		if supported[target][capability.Kind] {
			compatible = append(compatible, capability)
			continue
		}
		if mode == ModeStrict {
			return Artifact{}, nil, fmt.Errorf("target %s cannot safely represent allowed %s capability %q", target, capability.Kind, capability.ID)
		}
		diagnostics = append(diagnostics, Diagnostic{
			Severity:    "warning",
			Code:        "capability_omitted",
			Message:     fmt.Sprintf("omitted allowed capability %q because %s cannot represent %s without broadening access", capability.ID, target, capability.Kind),
			Target:      target,
			RuleID:      capability.ID,
			Recoverable: true,
		})
	}

	var artifact Artifact
	var err error
	switch target {
	case TargetOpenShell:
		artifact, err = renderOpenShell(compatible, options.OpenShell)
	case TargetOpenCode:
		artifact, err = renderOpenCode(compatible)
	case TargetCodex:
		artifact, err = renderCodex(compatible, options.Codex)
	case TargetClaude:
		artifact, err = renderClaude(compatible)
	default:
		err = fmt.Errorf("unsupported target %q", target)
	}
	return artifact, diagnostics, err
}

func renderOpenShell(capabilities []Capability, options OpenShellOptions) (Artifact, error) {
	compatibility := options.LandlockCompatibility
	if compatibility == "" {
		compatibility = "hard_requirement"
	}
	if compatibility != "hard_requirement" && compatibility != "best_effort" {
		return Artifact{}, fmt.Errorf("invalid OpenShell Landlock compatibility %q", compatibility)
	}
	user := options.RunAsUser
	if user == "" {
		user = "sandbox"
	}
	group := options.RunAsGroup
	if group == "" {
		group = "sandbox"
	}
	if user == "root" || user == "0" || group == "root" || group == "0" {
		return Artifact{}, errors.New("OpenShell process identity must not be root")
	}

	var readOnly, readWrite []string
	var networks []Capability
	for _, capability := range capabilities {
		switch capability.Kind {
		case KindFilesystem:
			if !filepath.IsAbs(capability.Selector) {
				return Artifact{}, fmt.Errorf("OpenShell filesystem capability %q requires an absolute path", capability.ID)
			}
			clean := filepath.Clean(capability.Selector)
			if capability.Action == "write" {
				if clean == string(filepath.Separator) {
					return Artifact{}, fmt.Errorf("OpenShell capability %q must not grant write access to the filesystem root", capability.ID)
				}
				readWrite = append(readWrite, clean)
			} else {
				readOnly = append(readOnly, clean)
			}
		case KindNetwork:
			if len(capability.Binaries) == 0 {
				return Artifact{}, fmt.Errorf("OpenShell network capability %q requires at least one binary", capability.ID)
			}
			for _, binary := range capability.Binaries {
				if !filepath.IsAbs(binary) {
					return Artifact{}, fmt.Errorf("OpenShell network capability %q binary %q must be absolute", capability.ID, binary)
				}
			}
			if capability.Protocol == "mcp" || capability.Protocol == "json-rpc" {
				return Artifact{}, fmt.Errorf("OpenShell capability %q requires explicit protocol rules that are not present in the v0.5 catalog", capability.ID)
			}
			if capability.Protocol != "" && capability.Protocol != "rest" && capability.Protocol != "websocket" && capability.Protocol != "graphql" {
				return Artifact{}, fmt.Errorf("OpenShell capability %q has invalid protocol %q", capability.ID, capability.Protocol)
			}
			if capability.Protocol != "" && capability.Access == "" {
				return Artifact{}, fmt.Errorf("OpenShell L7 capability %q requires an access preset", capability.ID)
			}
			if capability.Protocol == "" && capability.Access != "" {
				return Artifact{}, fmt.Errorf("OpenShell TCP capability %q must not set an L7 access preset", capability.ID)
			}
			if capability.Access != "" && capability.Access != "read-only" && capability.Access != "read-write" && capability.Access != "full" {
				return Artifact{}, fmt.Errorf("OpenShell capability %q has invalid access preset %q", capability.ID, capability.Access)
			}
			networks = append(networks, capability)
		}
	}
	readOnly = uniqueSorted(readOnly)
	readWrite = uniqueSorted(readWrite)
	writable := make(map[string]struct{}, len(readWrite))
	for _, path := range readWrite {
		writable[path] = struct{}{}
	}
	readOnly = slicesWithout(readOnly, writable)
	if len(readOnly)+len(readWrite) > 256 {
		return Artifact{}, errors.New("OpenShell supports at most 256 filesystem paths")
	}

	var out strings.Builder
	out.WriteString("version: 1\nfilesystem_policy:\n  include_workdir: ")
	out.WriteString(strconv.FormatBool(options.IncludeWorkdir))
	out.WriteString("\n  read_only:")
	writeYAMLList(&out, readOnly, 4)
	out.WriteString("\n  read_write:")
	writeYAMLList(&out, readWrite, 4)
	out.WriteString("\nlandlock:\n  compatibility: ")
	out.WriteString(yamlString(compatibility))
	out.WriteString("\nprocess:\n  run_as_user: ")
	out.WriteString(yamlString(user))
	out.WriteString("\n  run_as_group: ")
	out.WriteString(yamlString(group))
	out.WriteString("\nnetwork_policies:")
	if len(networks) == 0 {
		out.WriteString(" {}")
	} else {
		for _, capability := range networks {
			out.WriteString("\n  ")
			out.WriteString(yamlString(capability.ID))
			out.WriteString(":\n    name: ")
			out.WriteString(yamlString(capability.ID))
			out.WriteString("\n    endpoints:\n      - host: ")
			out.WriteString(yamlString(capability.Selector))
			out.WriteString("\n        port: ")
			out.WriteString(strconv.Itoa(capability.Port))
			if capability.Path != "" {
				out.WriteString("\n        path: ")
				out.WriteString(yamlString(capability.Path))
			}
			if capability.Protocol != "" {
				out.WriteString("\n        protocol: ")
				out.WriteString(yamlString(capability.Protocol))
			}
			out.WriteString("\n        enforcement: enforce")
			if capability.Access != "" {
				out.WriteString("\n        access: ")
				out.WriteString(yamlString(capability.Access))
			}
			out.WriteString("\n    binaries:")
			for _, binary := range uniqueSorted(capability.Binaries) {
				out.WriteString("\n      - path: ")
				out.WriteString(yamlString(binary))
			}
		}
	}
	out.WriteByte('\n')
	return textArtifact(TargetOpenShell, "policy.yaml", "policy.yaml", "application/yaml", out.String()), nil
}

func renderOpenCode(capabilities []Capability) (Artifact, error) {
	permissions := map[string]any{"*": "deny"}
	granular := map[string]map[string]string{}
	for _, capability := range capabilities {
		switch capability.Kind {
		case KindFilesystem:
			key := "read"
			if capability.Action == "write" {
				key = "edit"
			}
			pattern := directoryPattern(capability.Selector)
			addGranular(granular, key, pattern)
			if filepath.IsAbs(capability.Selector) || strings.HasPrefix(capability.Selector, "~/") || strings.HasPrefix(capability.Selector, "$HOME/") {
				addGranular(granular, "external_directory", pattern)
			}
		case KindTool:
			if strings.ContainsAny(capability.Selector, "() ") {
				return Artifact{}, fmt.Errorf("OpenCode tool capability %q has invalid tool name %q", capability.ID, capability.Selector)
			}
			permissions[capability.Selector] = "allow"
		case KindCommand:
			addGranular(granular, "bash", capability.Selector)
		case KindNetwork:
			scheme := "https"
			if capability.Port == 80 {
				scheme = "http"
			}
			pattern := scheme + "://" + capability.Selector
			if capability.Port != 443 && capability.Port != 80 {
				pattern += ":" + strconv.Itoa(capability.Port)
			}
			pattern += "/*"
			addGranular(granular, "webfetch", pattern)
		}
	}
	for key, rules := range granular {
		permissions[key] = rules
	}
	body := map[string]any{"$schema": "https://opencode.ai/config.json", "permission": permissions}
	content, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return Artifact{}, err
	}
	return textArtifact(TargetOpenCode, "opencode.json", "opencode.json", "application/json", string(content)+"\n"), nil
}

func renderCodex(capabilities []Capability, options CodexOptions) (Artifact, error) {
	profile := options.ProfileName
	if profile == "" {
		profile = "rosetta"
	}
	if !validTOMLBareKey(profile) {
		return Artifact{}, fmt.Errorf("invalid Codex profile name %q", profile)
	}
	workspaceFilesystem := map[string]string{".": "deny"}
	absoluteFilesystem := map[string]string{}
	mcp := map[string][]string{}
	for _, capability := range capabilities {
		switch capability.Kind {
		case KindFilesystem:
			access := "read"
			if capability.Action == "write" {
				access = "write"
			}
			if filepath.IsAbs(capability.Selector) {
				absoluteFilesystem[capability.Selector] = access
			} else {
				workspaceFilesystem[capability.Selector] = access
			}
		case KindTool:
			if capability.Server == "" {
				return Artifact{}, fmt.Errorf("Codex tool capability %q requires an MCP server", capability.ID)
			}
			if !validTOMLBareKey(capability.Server) {
				return Artifact{}, fmt.Errorf("Codex tool capability %q has invalid MCP server %q", capability.ID, capability.Server)
			}
			mcp[capability.Server] = append(mcp[capability.Server], capability.Selector)
		}
	}

	var out strings.Builder
	out.WriteString("default_permissions = ")
	out.WriteString(strconv.Quote(profile))
	out.WriteString("\n\n[permissions.")
	out.WriteString(profile)
	out.WriteString("]\nextends = \":read-only\"\n\n[permissions.")
	out.WriteString(profile)
	out.WriteString(".filesystem]\n\":minimal\" = \"read\"\n\":workspace_roots\" = { ")
	keys := sortedKeys(workspaceFilesystem)
	for i, key := range keys {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(strconv.Quote(key))
		out.WriteString(" = ")
		out.WriteString(strconv.Quote(workspaceFilesystem[key]))
	}
	out.WriteString(" }\n")
	for _, path := range sortedKeys(absoluteFilesystem) {
		out.WriteString(strconv.Quote(path))
		out.WriteString(" = ")
		out.WriteString(strconv.Quote(absoluteFilesystem[path]))
		out.WriteByte('\n')
	}
	if options.WorkspaceRoot != "" && options.WorkspaceRoot != "." {
		out.WriteString("\n[permissions.")
		out.WriteString(profile)
		out.WriteString(".workspace_roots]\n")
		out.WriteString(strconv.Quote(options.WorkspaceRoot))
		out.WriteString(" = true\n")
	}
	for _, server := range sortedKeys(mcp) {
		tools := uniqueSorted(mcp[server])
		definition, found := options.MCPServers[server]
		if !found {
			return Artifact{}, fmt.Errorf("Codex MCP server %q requires a transport definition in options.codex.mcpServers", server)
		}
		if (definition.Command == "") == (definition.URL == "") {
			return Artifact{}, fmt.Errorf("Codex MCP server %q must set exactly one of command or url", server)
		}
		if definition.Command != "" && definition.BearerTokenEnvVar != "" {
			return Artifact{}, fmt.Errorf("Codex stdio MCP server %q must not set bearerTokenEnvVar", server)
		}
		if err := validatePlainString("Codex MCP command", definition.Command, MaxSelectorBytes); err != nil {
			return Artifact{}, fmt.Errorf("Codex MCP server %q: %w", server, err)
		}
		for _, arg := range definition.Args {
			if err := validatePlainString("Codex MCP argument", arg, MaxSelectorBytes); err != nil {
				return Artifact{}, fmt.Errorf("Codex MCP server %q: %w", server, err)
			}
		}
		if definition.URL != "" {
			parsed, err := url.ParseRequestURI(definition.URL)
			if err != nil || parsed.Host == "" || parsed.Scheme != "https" && parsed.Scheme != "http" {
				return Artifact{}, fmt.Errorf("Codex MCP server %q has invalid HTTP URL", server)
			}
		}
		if definition.BearerTokenEnvVar != "" && !validEnvironmentName(definition.BearerTokenEnvVar) {
			return Artifact{}, fmt.Errorf("Codex MCP server %q has invalid bearer token environment variable", server)
		}
		out.WriteString("\n[mcp_servers.")
		out.WriteString(server)
		out.WriteString("]\n")
		if definition.Command != "" {
			out.WriteString("command = ")
			out.WriteString(strconv.Quote(definition.Command))
			out.WriteByte('\n')
			if len(definition.Args) > 0 {
				out.WriteString("args = [")
				for i, arg := range definition.Args {
					if i > 0 {
						out.WriteString(", ")
					}
					out.WriteString(strconv.Quote(arg))
				}
				out.WriteString("]\n")
			}
		} else {
			out.WriteString("url = ")
			out.WriteString(strconv.Quote(definition.URL))
			out.WriteByte('\n')
			if definition.BearerTokenEnvVar != "" {
				out.WriteString("bearer_token_env_var = ")
				out.WriteString(strconv.Quote(definition.BearerTokenEnvVar))
				out.WriteByte('\n')
			}
		}
		out.WriteString("enabled_tools = [")
		for i, tool := range tools {
			if i > 0 {
				out.WriteString(", ")
			}
			out.WriteString(strconv.Quote(tool))
		}
		out.WriteString("]\n")
	}
	return textArtifact(TargetCodex, "config.toml", ".codex/config.toml", "application/toml", out.String()), nil
}

func renderClaude(capabilities []Capability) (Artifact, error) {
	allow := make([]string, 0)
	readPaths := make([]string, 0)
	writePaths := make([]string, 0)
	domains := make([]string, 0)
	for _, capability := range capabilities {
		switch capability.Kind {
		case KindFilesystem:
			if capability.Action == "read" {
				allow = append(allow, "Read("+directoryPattern(capability.Selector)+")")
				readPaths = append(readPaths, capability.Selector)
			} else {
				allow = append(allow, "Edit("+directoryPattern(capability.Selector)+")")
				writePaths = append(writePaths, capability.Selector)
			}
		case KindTool:
			allow = append(allow, capability.Selector)
		case KindCommand:
			allow = append(allow, "Bash("+capability.Selector+")")
		case KindNetwork:
			domains = append(domains, capability.Selector)
			allow = append(allow, "WebFetch(domain:"+capability.Selector+")")
		}
	}
	body := map[string]any{
		"$schema": "https://json.schemastore.org/claude-code-settings.json",
		"permissions": map[string]any{
			"allow":                        uniqueSorted(allow),
			"defaultMode":                  "dontAsk",
			"disableAutoMode":              "disable",
			"disableBypassPermissionsMode": "disable",
		},
		"sandbox": map[string]any{
			"enabled":                  true,
			"failIfUnavailable":        true,
			"autoAllowBashIfSandboxed": false,
			"allowUnsandboxedCommands": false,
			"filesystem": map[string]any{
				"denyRead":   []string{"~/"},
				"allowRead":  uniqueSorted(readPaths),
				"allowWrite": uniqueSorted(writePaths),
			},
			"network": map[string]any{
				"allowedDomains": uniqueSorted(domains),
			},
		},
	}
	content, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return Artifact{}, err
	}
	return textArtifact(TargetClaude, "settings.json", ".claude/settings.json", "application/json", string(content)+"\n"), nil
}

func addGranular(all map[string]map[string]string, key, pattern string) {
	rules := all[key]
	if rules == nil {
		rules = map[string]string{"*": "deny"}
		all[key] = rules
	}
	rules[pattern] = "allow"
}

func directoryPattern(root string) string {
	root = strings.TrimSuffix(filepath.ToSlash(root), "/")
	if root == "" {
		return "/**"
	}
	return root + "/**"
}

func textArtifact(target, name, pathHint, mediaType, content string) Artifact {
	return Artifact{Name: name, PathHint: pathHint, MediaType: mediaType, Target: target, Content: content, Encoding: "plain"}
}

func writeYAMLList(out *strings.Builder, values []string, indent int) {
	if len(values) == 0 {
		out.WriteString(" []")
		return
	}
	for _, value := range values {
		out.WriteByte('\n')
		out.WriteString(strings.Repeat(" ", indent))
		out.WriteString("- ")
		out.WriteString(yamlString(value))
	}
}

func yamlString(value string) string {
	body, _ := json.Marshal(value)
	return string(body)
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func slicesWithout(values []string, excluded map[string]struct{}) []string {
	result := values[:0]
	for _, value := range values {
		if _, found := excluded[value]; !found {
			result = append(result, value)
		}
	}
	return result
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func validTOMLBareKey(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !(r == '-' || r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func validEnvironmentName(value string) bool {
	for index, r := range value {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || index > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return value != ""
}

// marshalOrderedJSON is retained as a test seam for encoders whose rule order
// is significant. encoding/json sorts string map keys, keeping catch-alls first.
func marshalOrderedJSON(value any) ([]byte, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
