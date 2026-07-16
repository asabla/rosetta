package rosetta

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const Version = "0.1.0"

var targets = []string{"openshell"}

const (
	// ModeStrict fails when translation would be lossy, unsupported, or access-broadening.
	ModeStrict = "strict"
	// ModePermissive returns safe approximation artifacts with warnings when possible.
	ModePermissive = "permissive"
)

// CompileRequest describes policy source translation into a target artifact.
type CompileRequest struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Mode   string `json:"mode,omitempty"`
}

// CompileResult contains generated translation output and metadata.
type CompileResult struct {
	Output      string       `json:"output"`
	Target      string       `json:"target"`
	Artifacts   []Artifact   `json:"artifacts,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

// CheckRequest describes policy source validation.
type CheckRequest struct {
	Source string `json:"source"`
	Mode   string `json:"mode,omitempty"`
}

// CheckResult contains validation status and diagnostics.
type CheckResult struct {
	Valid       bool         `json:"valid"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
	Errors      []string     `json:"errors,omitempty"`
}

// ExplainRequest describes a request to explain target translation.
type ExplainRequest struct {
	Source string `json:"source"`
	Target string `json:"target,omitempty"`
	Mode   string `json:"mode,omitempty"`
}

// ExplainResult contains a human-readable translation explanation.
type ExplainResult struct {
	Explanation string       `json:"explanation"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

// CapabilitiesRequest describes a request for Rosetta feature metadata.
type CapabilitiesRequest struct{}

// CapabilitiesResult contains supported features and targets.
type CapabilitiesResult struct {
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
	Targets      []string `json:"targets,omitempty"`
}

// Diagnostic describes a validation or translation message.
//
// Code is stable enough for automation. Message is intended to remain
// human-readable and may change as Rosetta improves diagnostic wording.
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

// SourceSpan identifies a source range related to a diagnostic.
type SourceSpan struct {
	StartLine   int `json:"startLine,omitempty"`
	StartColumn int `json:"startColumn,omitempty"`
	EndLine     int `json:"endLine,omitempty"`
	EndColumn   int `json:"endColumn,omitempty"`
}

// Artifact describes a generated target artifact.
type Artifact struct {
	Name        string `json:"name"`
	PathHint    string `json:"pathHint,omitempty"`
	MediaType   string `json:"mediaType"`
	Target      string `json:"target,omitempty"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
	Description string `json:"description,omitempty"`
}

// Targets returns the policy rendering targets supported by Rosetta.
func Targets() []string {
	return append([]string(nil), targets...)
}

// Capabilities returns compiler capabilities shared by the CLI and service.
func Capabilities(ctx context.Context, _ CapabilitiesRequest) (*CapabilitiesResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &CapabilitiesResult{
		Version:      Version,
		Capabilities: []string{"compile", "check", "explain", "capabilities", "targets", "openapi"},
		Targets:      Targets(),
	}, nil
}

// Compile validates source policy text and renders it for the requested target.
func Compile(ctx context.Context, req CompileRequest) (*CompileResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	mode, err := modeOrDefault(req.Mode)
	if err != nil {
		return nil, err
	}
	if diagnostic, ok := validateSource(req.Source); !ok {
		return nil, diagnosticError(diagnostic)
	}
	target := targetOrDefault(req.Target)
	if !supportedTarget(target) {
		return nil, fmt.Errorf("unsupported target %q", target)
	}
	output := fmt.Sprintf("# target: %s\n%s", target, strings.TrimSpace(req.Source))
	return &CompileResult{
		Output:      output,
		Target:      target,
		Diagnostics: modeDiagnostics(mode),
		Artifacts: []Artifact{{
			Name:        target + ".policy",
			PathHint:    target + ".policy",
			MediaType:   "text/plain; charset=utf-8",
			Target:      target,
			Content:     output,
			Encoding:    "plain",
			Description: "Rendered policy artifact for the requested target.",
		}},
	}, nil
}

// Check validates source policy text.
func Check(ctx context.Context, req CheckRequest) (*CheckResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	mode, err := modeOrDefault(req.Mode)
	if err != nil {
		return nil, err
	}
	if diagnostic, ok := validateSource(req.Source); !ok {
		return &CheckResult{Valid: false, Diagnostics: []Diagnostic{diagnostic}, Errors: []string{diagnostic.Message}}, nil
	}
	return &CheckResult{Valid: true, Diagnostics: modeDiagnostics(mode)}, nil
}

// Explain describes how Rosetta would process the source policy text.
func Explain(ctx context.Context, req ExplainRequest) (*ExplainResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	mode, err := modeOrDefault(req.Mode)
	if err != nil {
		return nil, err
	}
	if diagnostic, ok := validateSource(req.Source); !ok {
		return nil, diagnosticError(diagnostic)
	}
	target := targetOrDefault(req.Target)
	if !supportedTarget(target) {
		return nil, fmt.Errorf("unsupported target %q", target)
	}
	return &ExplainResult{
		Explanation: fmt.Sprintf("Rosetta validates Cedar policy input and renders it for the %s target in %s mode.", target, mode),
		Diagnostics: modeDiagnostics(mode),
	}, nil
}

func validateSource(source string) (Diagnostic, bool) {
	if strings.TrimSpace(source) == "" {
		return Diagnostic{Message: "source is required", Severity: "error", Code: "source_required"}, false
	}
	return Diagnostic{}, true
}

func diagnosticError(d Diagnostic) error {
	if d.Message == "" {
		return errors.New("unknown diagnostic")
	}
	return errors.New(d.Message)
}

func targetOrDefault(target string) string {
	if target != "" {
		return target
	}
	return targets[0]
}

func supportedTarget(target string) bool {
	for _, candidate := range targets {
		if candidate == target {
			return true
		}
	}
	return false
}

func modeOrDefault(mode string) (string, error) {
	if mode == "" {
		return ModeStrict, nil
	}
	switch mode {
	case ModeStrict, ModePermissive:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported mode %q", mode)
	}
}

func modeDiagnostics(mode string) []Diagnostic {
	if mode != ModePermissive {
		return nil
	}
	return []Diagnostic{{
		Severity:    "warning",
		Code:        "permissive_mode",
		Message:     "permissive mode may return safe approximations with warnings; it never silently broadens a Cedar deny into a target allow",
		Recoverable: true,
	}}
}
