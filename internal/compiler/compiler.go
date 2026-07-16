package compiler

import (
	"context"

	"rosetta/internal/rosetta"
)

const Version = rosetta.Version

// Targets returns the policy rendering targets supported by Rosetta.
func Targets() []string {
	return rosetta.Targets()
}

// Capabilities returns the compiler capabilities shared by the CLI and service.
func Capabilities() []string {
	result, err := rosetta.Capabilities(context.Background(), rosetta.CapabilitiesRequest{})
	if err != nil {
		return nil
	}
	return result.Capabilities
}

// Compile validates source policy text and renders it for the requested target.
func Compile(source, target string) (string, error) {
	result, err := rosetta.Compile(context.Background(), rosetta.CompileRequest{Source: source, Target: target})
	if err != nil {
		return "", err
	}
	return result.Output, nil
}

// Check validates source policy text.
func Check(source string) []error {
	result, err := rosetta.Check(context.Background(), rosetta.CheckRequest{Source: source})
	if err != nil {
		return []error{err}
	}
	if result.Valid {
		return nil
	}
	errs := make([]error, 0, len(result.Errors))
	for _, message := range result.Errors {
		errs = append(errs, errString(message))
	}
	return errs
}

// Explain describes how Rosetta would process the source policy text.
func Explain(source, target string) (string, error) {
	result, err := rosetta.Explain(context.Background(), rosetta.ExplainRequest{Source: source, Target: target})
	if err != nil {
		return "", err
	}
	return result.Explanation, nil
}

type errString string

func (e errString) Error() string { return string(e) }
