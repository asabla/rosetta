package rosetta

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

func TestExampleArtifactsMatchGoldenFilesAndParse(t *testing.T) {
	policy, err := os.ReadFile(filepath.Join("examples", "developer.cedar"))
	if err != nil {
		t.Fatal(err)
	}
	catalogBody, err := os.ReadFile(filepath.Join("examples", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog Catalog
	if err := json.Unmarshal(catalogBody, &catalog); err != nil {
		t.Fatal(err)
	}
	optionsBody, err := os.ReadFile(filepath.Join("examples", "options.json"))
	if err != nil {
		t.Fatal(err)
	}
	var options TargetOptions
	if err := json.Unmarshal(optionsBody, &options); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		target string
		golden string
		parse  func([]byte) error
	}{
		{TargetOpenShell, "openshell.yaml", func(body []byte) error { return yaml.Unmarshal(body, &map[string]any{}) }},
		{TargetOpenCode, "opencode.json", func(body []byte) error { return json.Unmarshal(body, &map[string]any{}) }},
		{TargetCodex, "codex.toml", func(body []byte) error { return toml.Unmarshal(body, &map[string]any{}) }},
		{TargetClaude, "claude-code.json", func(body []byte) error { return json.Unmarshal(body, &map[string]any{}) }},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			mode := ModeStrict
			targetOptions := TargetOptions{}
			if tt.target == TargetClaude {
				mode = ModePermissive
			}
			if tt.target == TargetCodex {
				targetOptions = options
			}
			result, err := Compile(context.Background(), CompileRequest{Source: string(policy), Target: tt.target, Mode: mode, Catalog: catalog, Options: targetOptions})
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(filepath.Join("testdata", "golden", tt.golden))
			if err != nil {
				t.Fatal(err)
			}
			if result.Output != string(want) {
				t.Fatalf("artifact differs from %s\nwant:\n%s\ngot:\n%s", tt.golden, want, result.Output)
			}
			if err := tt.parse([]byte(result.Output)); err != nil {
				t.Fatalf("generated artifact does not parse: %v", err)
			}
		})
	}
}
