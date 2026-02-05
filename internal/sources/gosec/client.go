package gosec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/boinger/confvis/internal/sources/cmdrun"
)

// DefaultCommand is the default gosec command.
const DefaultCommand = "gosec"

// Client wraps gosec CLI execution.
type Client struct {
	command string
}

// NewClient creates a new Gosec CLI client.
// If command is empty, uses "gosec" as the default.
func NewClient(command string) *Client {
	if command == "" {
		command = DefaultCommand
	}
	return &Client{command: command}
}

// Scan runs gosec on the specified path and returns the parsed report.
func (c *Client) Scan(ctx context.Context, path string) (*Report, error) {
	// Build command arguments: gosec -fmt=json ./...
	// Note: gosec outputs JSON to stdout when -fmt=json is specified
	args := []string{
		"-fmt=json",
		"-quiet", // Suppress banner and other output
		path,
	}

	result, err := cmdrun.Run(ctx, c.command, args, "gosec")
	// Gosec returns exit code 1 when issues are found, which is not an error for us
	if err != nil {
		if fatalErr := checkGosecError(err, result.Stdout, result.Stderr); fatalErr != nil {
			return nil, fatalErr
		}
	}

	// Even if no issues are found, gosec outputs a valid JSON report
	stdout := bytes.TrimSpace(result.Stdout)
	if len(stdout) == 0 {
		// No output likely means an error occurred
		return nil, fmt.Errorf("gosec produced no output")
	}

	// Parse JSON output
	var report Report
	if err := json.Unmarshal(stdout, &report); err != nil {
		return nil, fmt.Errorf("parsing gosec output: %w", err)
	}

	return &report, nil
}

// checkGosecError returns nil if the error is acceptable (exit code 1 = issues found),
// otherwise returns a formatted error.
func checkGosecError(err error, stdout, stderr []byte) error {
	exitError, ok := err.(*exec.ExitError)
	if !ok {
		return fmt.Errorf("gosec scan failed: %w", err)
	}

	// Exit code 1 means issues were found - this is valid output, not an error
	// Check if stdout contains valid JSON (has "Issues" key)
	if exitError.ExitCode() == 1 && bytes.Contains(stdout, []byte("\"Issues\"")) {
		return nil
	}

	return cmdrun.FormatError(err, stderr, "gosec", "scan")
}
