package grype

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// DefaultCommand is the default grype command.
const DefaultCommand = "grype"

// Client wraps grype CLI execution.
type Client struct {
	command string
}

// NewClient creates a new Grype CLI client.
// If command is empty, uses "grype" as the default.
func NewClient(command string) *Client {
	if command == "" {
		command = DefaultCommand
	}
	return &Client{command: command}
}

// Scan runs grype on the specified target and returns the parsed report.
// Target can be a filesystem path, container image, or SBOM file.
func (c *Client) Scan(ctx context.Context, target string) (*Report, error) {
	// Build command arguments
	// grype <target> -o json
	args := []string{target, "-o", "json"}

	// Parse command (may be "grype" or "docker run anchore/grype")
	cmdParts := strings.Fields(c.command)
	if len(cmdParts) == 0 {
		return nil, fmt.Errorf("empty grype command")
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
			return nil, fmt.Errorf("grype scan failed: %w: %s", err, stderrStr)
		}
		return nil, fmt.Errorf("grype scan failed: %w", err)
	}

	// Parse JSON output
	var report Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		return nil, fmt.Errorf("parsing grype output: %w", err)
	}

	return &report, nil
}
