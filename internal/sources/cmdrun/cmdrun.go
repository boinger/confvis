// Package cmdrun provides utilities for running CLI commands.
package cmdrun

import (
	"bytes"
	"context"
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

	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
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
