package compiler

import (
	"errors"
	"fmt"
	"strings"
)

const Version = "0.1.0"

var targets = []string{"openshell"}

// Targets returns the policy rendering targets supported by Rosetta.
func Targets() []string {
	return append([]string(nil), targets...)
}

// Capabilities returns the compiler capabilities shared by the CLI and service.
func Capabilities() []string {
	return []string{"compile", "check", "explain", "capabilities", "targets", "openapi"}
}

// Compile validates source policy text and renders it for the requested target.
func Compile(source, target string) (string, error) {
	if err := validateSource(source); err != nil {
		return "", err
	}
	if target == "" {
		target = targets[0]
	}
	if !supportedTarget(target) {
		return "", fmt.Errorf("unsupported target %q", target)
	}
	return fmt.Sprintf("# target: %s\n%s", target, strings.TrimSpace(source)), nil
}

// Check validates source policy text.
func Check(source string) []error {
	if err := validateSource(source); err != nil {
		return []error{err}
	}
	return nil
}

// Explain describes how Rosetta would process the source policy text.
func Explain(source, target string) (string, error) {
	if err := validateSource(source); err != nil {
		return "", err
	}
	if target == "" {
		target = targets[0]
	}
	if !supportedTarget(target) {
		return "", fmt.Errorf("unsupported target %q", target)
	}
	return fmt.Sprintf("Rosetta validates Cedar policy input and renders it for the %s target.", target), nil
}

func validateSource(source string) error {
	if strings.TrimSpace(source) == "" {
		return errors.New("source is required")
	}
	return nil
}

func supportedTarget(target string) bool {
	for _, candidate := range targets {
		if candidate == target {
			return true
		}
	}
	return false
}
