package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/asabla/rosetta"
)

func TestCLIAndServiceCompileEquivalentArtifacts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping process E2E test")
	}
	root := repositoryRoot(t)
	binDir := t.TempDir()
	cli := filepath.Join(binDir, executable("rosetta"))
	server := filepath.Join(binDir, executable("rosetta-server"))
	build(t, root, cli, "./cmd/rosetta")
	build(t, root, server, "./cmd/rosetta-server")

	policy, err := os.ReadFile(filepath.Join(root, "examples", "developer.cedar"))
	if err != nil {
		t.Fatal(err)
	}
	catalogPath := filepath.Join(root, "examples", "catalog.json")
	catalogBody, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	var catalog rosetta.Catalog
	if err := json.Unmarshal(catalogBody, &catalog); err != nil {
		t.Fatal(err)
	}
	optionsPath := filepath.Join(root, "examples", "options.json")
	optionsBody, err := os.ReadFile(optionsPath)
	if err != nil {
		t.Fatal(err)
	}
	var options rosetta.TargetOptions
	if err := json.Unmarshal(optionsBody, &options); err != nil {
		t.Fatal(err)
	}

	address := freeAddress(t)
	process := exec.Command(server)
	process.Env = append(os.Environ(), "ROSETTA_ADDR="+address)
	var serverLog bytes.Buffer
	process.Stdout = &serverLog
	process.Stderr = &serverLog
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = process.Process.Kill()
		_, _ = process.Process.Wait()
	})
	waitForHealth(t, "http://"+address+"/healthz", &serverLog)

	client := &http.Client{Timeout: 10 * time.Second}
	for _, target := range rosetta.Targets() {
		t.Run(target, func(t *testing.T) {
			mode := rosetta.ModeStrict
			args := []string{"compile", "--target", target, "--catalog", catalogPath}
			request := rosetta.CompileRequest{Source: string(policy), Target: target, Catalog: catalog}
			if target == rosetta.TargetClaude {
				mode = rosetta.ModePermissive
				args = append(args, "--mode", mode)
				request.Mode = mode
			}
			if target == rosetta.TargetCodex {
				args = append(args, "--options", optionsPath)
				request.Options = options
			}
			command := exec.Command(cli, args...)
			command.Stdin = bytes.NewReader(policy)
			var cliOutput, cliDiagnostics bytes.Buffer
			command.Stdout = &cliOutput
			command.Stderr = &cliDiagnostics
			if err := command.Run(); err != nil {
				t.Fatalf("CLI compile failed: %v\n%s", err, cliDiagnostics.String())
			}

			requestBody, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			response, err := client.Post("http://"+address+"/v1/compile", "application/json", bytes.NewReader(requestBody))
			if err != nil {
				t.Fatal(err)
			}
			body, readErr := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if readErr != nil {
				t.Fatal(readErr)
			}
			if response.StatusCode != http.StatusOK {
				t.Fatalf("service compile returned %s: %s", response.Status, body)
			}
			var result rosetta.CompileResult
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(cliOutput.String()) != strings.TrimSpace(result.Output) {
				t.Fatalf("CLI and service artifacts differ\nCLI:\n%s\nservice:\n%s", cliOutput.String(), result.Output)
			}
			if target == rosetta.TargetClaude && !strings.Contains(cliDiagnostics.String(), "capability_omitted") {
				t.Fatalf("expected permissive diagnostics, got %q", cliDiagnostics.String())
			}
		})
	}
}

func build(t *testing.T, root, output, pkg string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "build", "-trimpath", "-o", output, pkg)
	command.Dir = root
	if body, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", pkg, err, body)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	return filepath.Dir(filepath.Dir(file))
}

func executable(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func freeAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return address
}

func waitForHealth(t *testing.T, url string, log *bytes.Buffer) {
	t.Helper()
	client := &http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get(url)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server did not become healthy: %s", fmt.Sprint(log))
}
