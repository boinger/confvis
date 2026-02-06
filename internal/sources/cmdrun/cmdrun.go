// Package cmdrun provides utilities for running CLI commands.
package cmdrun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Result contains the output from running a command.
type Result struct {
	Stdout []byte
	Stderr []byte
}

// Run executes a command with the given arguments and returns the output.
// The command parameter can be a simple command like "grype" or a compound
// command like "docker run anchore/grype".
//
// Parameters:
//   - ctx: Context for cancellation
//   - command: The base command (may include additional args, e.g. "docker run image")
//   - args: Additional arguments to append
//   - toolName: Name of the tool for error messages (e.g., "grype", "trivy")
func Run(ctx context.Context, command string, args []string, toolName string) (*Result, error) {
	// Parse command (may be "grype" or "docker run anchore/grype")
	cmdParts := strings.Fields(command)
	if len(cmdParts) == 0 {
		return &Result{}, fmt.Errorf("empty %s command", toolName)
	}

	cmdName := cmdParts[0]
	cmdArgs := append(cmdParts[1:], args...)

	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...) //#nosec G204 -- command from validated source defaults, not untrusted input
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return &Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, err
	}

	return &Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, nil
}

// FormatError formats an error with stderr output for better debugging.
func FormatError(err error, stderr []byte, toolName, action string) error {
	stderrStr := strings.TrimSpace(string(stderr))
	if stderrStr != "" {
		return fmt.Errorf("%s %s failed: %w: %s", toolName, action, err, stderrStr)
	}
	return fmt.Errorf("%s %s failed: %w", toolName, action, err)
}

// CheckAcceptableExitCode returns nil if the error is an acceptable exit code,
// otherwise returns a formatted error.
// acceptableCodes lists exit codes that are valid output (e.g., 1 for findings found).
func CheckAcceptableExitCode(err error, acceptableCodes []int, stderr []byte, toolName, action string) error {
	exitError, ok := err.(*exec.ExitError)
	if !ok {
		return fmt.Errorf("%s %s failed: %w", toolName, action, err)
	}

	exitCode := exitError.ExitCode()
	for _, code := range acceptableCodes {
		if exitCode == code {
			return nil
		}
	}

	return FormatError(err, stderr, toolName, action)
}

// ParseJSONOutput unmarshals JSON from command output with standard error handling.
// If stdout is empty or whitespace-only, returns an error.
func ParseJSONOutput[T any](stdout []byte, toolName string) (*T, error) {
	trimmed := bytes.TrimSpace(stdout)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("%s produced no output", toolName)
	}

	var result T
	if err := json.Unmarshal(trimmed, &result); err != nil {
		return nil, fmt.Errorf("parsing %s output: %w", toolName, err)
	}

	return &result, nil
}
