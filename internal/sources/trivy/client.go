package trivy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/boinger/confvis/internal/sources/cmdrun"
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
	// Build command arguments: trivy fs --format json --scanners vuln <path>
	args := []string{"fs", "--format", "json", "--scanners", "vuln", path}

	result, err := cmdrun.Run(ctx, c.command, args, "trivy")
	if err != nil {
		return nil, cmdrun.FormatError(err, result.Stderr, "trivy", "scan")
	}

	// Parse JSON output
	var report Report
	if err := json.Unmarshal(result.Stdout, &report); err != nil {
		return nil, fmt.Errorf("parsing trivy output: %w", err)
	}

	return &report, nil
}
