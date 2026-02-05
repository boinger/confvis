package gitleaks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/boinger/confvis/internal/sources/cmdrun"
)

// DefaultCommand is the default gitleaks command.
const DefaultCommand = "gitleaks"

// Client wraps gitleaks CLI execution.
type Client struct {
	command string
}

// NewClient creates a new GitLeaks CLI client.
// If command is empty, uses "gitleaks" as the default.
func NewClient(command string) *Client {
	if command == "" {
		command = DefaultCommand
	}
	return &Client{command: command}
}

// Scan runs gitleaks on the specified path and returns the parsed report.
func (c *Client) Scan(ctx context.Context, path string) (Report, error) {
	// Build command arguments: gitleaks detect --source <path> --report-format json --report-path -
	// The --report-path - outputs to stdout
	args := []string{
		"detect",
		"--source", path,
		"--report-format", "json",
		"--report-path", "-",
		"--no-banner", // Suppress banner output
	}

	result, err := cmdrun.Run(ctx, c.command, args, "gitleaks")
	// Gitleaks returns exit code 1 when leaks are found, which is not an error for us
	if err != nil {
		if fatalErr := checkGitleaksError(err, result.Stderr); fatalErr != nil {
			return nil, fatalErr
		}
	}

	// Empty output means no leaks found
	stdout := bytes.TrimSpace(result.Stdout)
	if len(stdout) == 0 {
		return Report{}, nil
	}

	// Parse JSON output
	var report Report
	if err := json.Unmarshal(stdout, &report); err != nil {
		return nil, fmt.Errorf("parsing gitleaks output: %w", err)
	}

	return report, nil
}

// checkGitleaksError returns nil if the error is acceptable (exit code 1 = leaks found),
// otherwise returns a formatted error.
func checkGitleaksError(err error, stderr []byte) error {
	exitError, ok := err.(*exec.ExitError)
	if !ok {
		return fmt.Errorf("gitleaks scan failed: %w", err)
	}

	// Exit code 1 means leaks were found - this is valid output, not an error
	if exitError.ExitCode() == 1 {
		return nil
	}

	return cmdrun.FormatError(err, stderr, "gitleaks", "detect")
}
