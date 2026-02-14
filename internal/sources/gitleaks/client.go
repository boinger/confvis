package gitleaks

import (
	"bytes"
	"context"

	"github.com/boinger/confvis/internal/sources/cmdrun"
)

// Compile-time interface compliance check.
var _ Fetcher = (*Client)(nil)

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
		if fatalErr := cmdrun.CheckAcceptableExitCode(err, []int{1}, result.Stderr, "gitleaks", "detect"); fatalErr != nil {
			return nil, fatalErr
		}
	}

	// Empty output means no leaks found
	stdout := bytes.TrimSpace(result.Stdout)
	if len(stdout) == 0 {
		return Report{}, nil
	}

	// Parse JSON output
	report, err := cmdrun.ParseJSONOutput[Report](stdout, "gitleaks")
	if err != nil {
		return nil, err
	}

	return *report, nil
}
