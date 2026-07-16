package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"rosetta/internal/rosetta"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: rosetta <compile|check|explain|targets|capabilities>")
	}
	switch args[0] {
	case "compile":
		fs := flag.NewFlagSet("compile", flag.ContinueOnError)
		target := fs.String("target", "", "rendering target")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		source, err := io.ReadAll(stdin)
		if err != nil {
			return err
		}
		output, err := rosetta.Compile(context.Background(), rosetta.CompileRequest{Source: string(source), Target: *target})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, output.Output)
		return err
	case "check":
		source, err := io.ReadAll(stdin)
		if err != nil {
			return err
		}
		result, err := rosetta.Check(context.Background(), rosetta.CheckRequest{Source: string(source)})
		if err != nil {
			return err
		}
		if !result.Valid {
			return errors.New(result.Errors[0])
		}
		_, err = fmt.Fprintln(stdout, "ok")
		return err
	case "explain":
		source, err := io.ReadAll(stdin)
		if err != nil {
			return err
		}
		explanation, err := rosetta.Explain(context.Background(), rosetta.ExplainRequest{Source: string(source)})
		if err != nil {
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
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}
