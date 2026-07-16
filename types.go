package rosetta

// Version is the Rosetta API and compiler version.
const Version = "0.5.0"

const (
	ModeStrict     = "strict"
	ModePermissive = "permissive"

	TargetOpenShell = "openshell"
	TargetOpenCode  = "opencode"
	TargetCodex     = "codex"
	TargetClaude    = "claude-code"
)

const (
	KindFilesystem = "filesystem"
	KindTool       = "tool"
	KindCommand    = "command"
	KindNetwork    = "network"
)

// EntityRef identifies the Cedar principal used for static capability decisions.
// The type defaults to Rosetta::Principal.
type EntityRef struct {
	Type  string   `json:"type,omitempty"`
	ID    string   `json:"id"`
	Roles []string `json:"roles,omitempty"`
}

// Catalog enumerates the capabilities Cedar must decide before rendering.
// Rosetta never invents capabilities that are absent from this catalog.
type Catalog struct {
	Version      string       `json:"version"`
	Principal    EntityRef    `json:"principal"`
	Capabilities []Capability `json:"capabilities"`
}

// Capability is a target-neutral operation that can be represented by one or
// more supported policy formats. Filesystem selectors name directory roots;
// renderers add target-native recursive matching where required.
type Capability struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Action   string   `json:"action"`
	Selector string   `json:"selector"`
	Targets  []string `json:"targets,omitempty"`

	Access   string   `json:"access,omitempty"`
	Port     int      `json:"port,omitempty"`
	Protocol string   `json:"protocol,omitempty"`
	Path     string   `json:"path,omitempty"`
	Binaries []string `json:"binaries,omitempty"`
	Server   string   `json:"server,omitempty"`
}

// CompileRequest describes Cedar translation into a target artifact.
type CompileRequest struct {
	Source  string        `json:"source"`
	Target  string        `json:"target"`
	Mode    string        `json:"mode,omitempty"`
	Catalog Catalog       `json:"catalog"`
	Options TargetOptions `json:"options,omitempty"`
}

// CheckRequest describes Cedar source validation against the Rosetta profile.
type CheckRequest struct {
	Source string `json:"source"`
	Mode   string `json:"mode,omitempty"`
}

// ExplainRequest describes a request to explain a compilation.
type ExplainRequest = CompileRequest

// TargetOptions controls target details that cannot be inferred from Cedar.
type TargetOptions struct {
	OpenShell OpenShellOptions `json:"openShell,omitempty"`
	Codex     CodexOptions     `json:"codex,omitempty"`
}

type OpenShellOptions struct {
	IncludeWorkdir        bool   `json:"includeWorkdir,omitempty"`
	LandlockCompatibility string `json:"landlockCompatibility,omitempty"`
	RunAsUser             string `json:"runAsUser,omitempty"`
	RunAsGroup            string `json:"runAsGroup,omitempty"`
}

type CodexOptions struct {
	ProfileName   string                    `json:"profileName,omitempty"`
	WorkspaceRoot string                    `json:"workspaceRoot,omitempty"`
	MCPServers    map[string]CodexMCPServer `json:"mcpServers,omitempty"`
}

// CodexMCPServer defines the transport for an MCP server whose enabled tools
// are restricted by Cedar. Set exactly one of Command or URL.
type CodexMCPServer struct {
	Command           string   `json:"command,omitempty"`
	Args              []string `json:"args,omitempty"`
	URL               string   `json:"url,omitempty"`
	BearerTokenEnvVar string   `json:"bearerTokenEnvVar,omitempty"`
}

// CompileResult contains generated artifacts and the Cedar decisions behind them.
type CompileResult struct {
	Output      string       `json:"output"`
	Target      string       `json:"target"`
	Artifacts   []Artifact   `json:"artifacts"`
	Decisions   []Decision   `json:"decisions"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

type CheckResult struct {
	Valid       bool         `json:"valid"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
	Errors      []string     `json:"errors,omitempty"`
}

type ExplainResult struct {
	Explanation string       `json:"explanation"`
	Decisions   []Decision   `json:"decisions,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

type CapabilitiesRequest struct{}

type CapabilitiesResult struct {
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
	Targets      []string `json:"targets"`
}

// Decision records how Cedar resolved one catalog entry.
type Decision struct {
	CapabilityID string   `json:"capabilityId"`
	Allowed      bool     `json:"allowed"`
	PolicyIDs    []string `json:"policyIds,omitempty"`
}

type Diagnostic struct {
	Severity         string         `json:"severity"`
	Code             string         `json:"code"`
	Message          string         `json:"message"`
	Details          map[string]any `json:"details,omitempty"`
	SourceSpan       *SourceSpan    `json:"sourceSpan,omitempty"`
	Target           string         `json:"target,omitempty"`
	RuleID           string         `json:"ruleId,omitempty"`
	Recoverable      bool           `json:"recoverable,omitempty"`
	DocumentationURL string         `json:"documentationUrl,omitempty"`
}

type SourceSpan struct {
	StartLine   int `json:"startLine,omitempty"`
	StartColumn int `json:"startColumn,omitempty"`
	EndLine     int `json:"endLine,omitempty"`
	EndColumn   int `json:"endColumn,omitempty"`
}

type Artifact struct {
	Name        string `json:"name"`
	PathHint    string `json:"pathHint,omitempty"`
	MediaType   string `json:"mediaType"`
	Target      string `json:"target"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
	Description string `json:"description,omitempty"`
}
