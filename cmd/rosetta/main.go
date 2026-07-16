package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/asabla/rosetta"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: rosetta <compile|check|explain|targets|capabilities|version>")
	}
	switch args[0] {
	case "compile":
		fs := flag.NewFlagSet("compile", flag.ContinueOnError)
		target := fs.String("target", "", "rendering target")
		mode := fs.String("mode", rosetta.ModeStrict, "compilation mode (strict|permissive)")
		catalogPath := fs.String("catalog", "", "path to a Rosetta capability catalog")
		optionsPath := fs.String("options", "", "path to target rendering options")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		source, err := io.ReadAll(stdin)
		if err != nil {
			return err
		}
		catalog, err := readCatalog(*catalogPath)
		if err != nil {
			return err
		}
		options, err := readOptions(*optionsPath)
		if err != nil {
			return err
		}
		output, err := rosetta.Compile(context.Background(), rosetta.CompileRequest{Source: string(source), Target: *target, Mode: *mode, Catalog: catalog, Options: options})
		if err != nil {
			return err
		}
		if err := writeDiagnostics(stderr, output.Diagnostics); err != nil {
			return err
		}
		_, err = io.WriteString(stdout, output.Output)
		return err
	case "check":
		fs := flag.NewFlagSet("check", flag.ContinueOnError)
		mode := fs.String("mode", rosetta.ModeStrict, "compilation mode (strict|permissive)")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		source, err := io.ReadAll(stdin)
		if err != nil {
			return err
		}
		result, err := rosetta.Check(context.Background(), rosetta.CheckRequest{Source: string(source), Mode: *mode})
		if err != nil {
			return err
		}
		if !result.Valid {
			if err := writeDiagnostics(stderr, result.Diagnostics); err != nil {
				return err
			}
			return errors.New(result.Errors[0])
		}
		_, err = fmt.Fprintln(stdout, "ok")
		return err
	case "explain":
		fs := flag.NewFlagSet("explain", flag.ContinueOnError)
		target := fs.String("target", "", "rendering target")
		mode := fs.String("mode", rosetta.ModeStrict, "compilation mode (strict|permissive)")
		catalogPath := fs.String("catalog", "", "path to a Rosetta capability catalog")
		optionsPath := fs.String("options", "", "path to target rendering options")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		source, err := io.ReadAll(stdin)
		if err != nil {
			return err
		}
		catalog, err := readCatalog(*catalogPath)
		if err != nil {
			return err
		}
		options, err := readOptions(*optionsPath)
		if err != nil {
			return err
		}
		explanation, err := rosetta.Explain(context.Background(), rosetta.ExplainRequest{Source: string(source), Target: *target, Mode: *mode, Catalog: catalog, Options: options})
		if err != nil {
			return err
		}
		if err := writeDiagnostics(stderr, explanation.Diagnostics); err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, explanation.Explanation)
		return err
	case "targets":
		for _, target := range rosetta.Targets() {
			if _, err := fmt.Fprintln(stdout, target); err != nil {
				return err
			}
		}
		return nil
	case "capabilities":
		capabilities, err := rosetta.Capabilities(context.Background(), rosetta.CapabilitiesRequest{})
		if err != nil {
			return err
		}
		for _, capability := range capabilities.Capabilities {
			if _, err := fmt.Fprintln(stdout, capability); err != nil {
				return err
			}
		}
		return nil
	case "version":
		_, err := fmt.Fprintf(stdout, "rosetta %s\n", rosetta.Version)
		return err
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func writeDiagnostics(out io.Writer, diagnostics []rosetta.Diagnostic) error {
	for _, diagnostic := range diagnostics {
		if _, err := fmt.Fprintf(out, "%s[%s]: %s\n", diagnostic.Severity, diagnostic.Code, diagnostic.Message); err != nil {
			return err
		}
	}
	return nil
}

func readCatalog(path string) (rosetta.Catalog, error) {
	if path == "" {
		return rosetta.Catalog{}, errors.New("--catalog is required")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return rosetta.Catalog{}, fmt.Errorf("read catalog: %w", err)
	}
	var catalog rosetta.Catalog
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&catalog); err != nil {
		return rosetta.Catalog{}, fmt.Errorf("decode catalog: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return rosetta.Catalog{}, errors.New("catalog must contain exactly one JSON object")
	}
	return catalog, nil
}

func readOptions(path string) (rosetta.TargetOptions, error) {
	if path == "" {
		return rosetta.TargetOptions{}, nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return rosetta.TargetOptions{}, fmt.Errorf("read options: %w", err)
	}
	var options rosetta.TargetOptions
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&options); err != nil {
		return rosetta.TargetOptions{}, fmt.Errorf("decode options: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return rosetta.TargetOptions{}, errors.New("options must contain exactly one JSON object")
	}
	return options, nil
}
