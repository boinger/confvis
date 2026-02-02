package trivy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// DefaultCommand is the default trivy command.
const DefaultCommand = "trivy"

// Client wraps trivy CLI execution.
type Client struct {
	command string
}

// NewClient creates a new Trivy CLI client.
// If command is empty, uses "trivy" as the default.
func NewClient(command string) *Client {
	if command == "" {
		command = DefaultCommand
	}
	return &Client{command: command}
}

// Scan runs trivy fs on the specified path and returns the parsed report.
func (c *Client) Scan(ctx context.Context, path string) (*Report, error) {
	// Build command arguments
	// trivy fs --format json --scanners vuln <path>
	args := []string{"fs", "--format", "json", "--scanners", "vuln", path}

	// Parse command (may be "trivy" or "docker run aquasec/trivy")
	cmdParts := strings.Fields(c.command)
	if len(cmdParts) == 0 {
		return nil, fmt.Errorf("empty trivy command")
	}

	cmdName := cmdParts[0]
	cmdArgs := append(cmdParts[1:], args...)

	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Include stderr in error message for debugging
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return nil, fmt.Errorf("trivy scan failed: %w: %s", err, stderrStr)
		}
		return nil, fmt.Errorf("trivy scan failed: %w", err)
	}

	// Parse JSON output
	var report Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		return nil, fmt.Errorf("parsing trivy output: %w", err)
	}

	return &report, nil
}
