package grype

import (
	"context"

	"github.com/boinger/confvis/internal/sources/cmdrun"
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
	// Build command arguments: grype <target> -o json
	args := []string{target, "-o", "json"}

	result, err := cmdrun.Run(ctx, c.command, args, "grype")
	if err != nil {
		return nil, cmdrun.FormatError(err, result.Stderr, "grype", "scan")
	}

	// Parse JSON output
	return cmdrun.ParseJSONOutput[Report](result.Stdout, "grype")
}
