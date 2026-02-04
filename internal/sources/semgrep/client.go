package semgrep

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"

	"github.com/boinger/confvis/internal/sources/cmdrun"
)

// DefaultCommand is the default semgrep command.
const DefaultCommand = "semgrep"

// Client wraps semgrep CLI execution.
type Client struct {
	command string
}

// NewClient creates a new Semgrep CLI client.
// If command is empty, uses "semgrep" as the default.
func NewClient(command string) *Client {
	if command == "" {
		command = DefaultCommand
	}
	return &Client{command: command}
}

// Scan runs semgrep on the specified path and returns the parsed report.
func (c *Client) Scan(ctx context.Context, path string, config string) (*Report, error) {
	// Build command arguments: semgrep scan --json --config <config> <path>
	args := []string{"scan", "--json"}
	if config != "" {
		args = append(args, "--config", config)
	} else {
		args = append(args, "--config", "auto")
	}
	args = append(args, path)

	result, err := cmdrun.Run(ctx, c.command, args, "semgrep")
	// Semgrep returns exit code 1 when findings exist, which is not an error for us
	if err != nil {
		if fatalErr := checkSemgrepError(err, result.Stderr); fatalErr != nil {
			return nil, fatalErr
		}
	}

	// Parse JSON output
	var report Report
	if err := json.Unmarshal(result.Stdout, &report); err != nil {
		return nil, fmt.Errorf("parsing semgrep output: %w", err)
	}

	return &report, nil
}

// checkSemgrepError returns nil if the error is acceptable (exit code 1 = findings found),
// otherwise returns a formatted error.
func checkSemgrepError(err error, stderr []byte) error {
	exitError, ok := err.(*exec.ExitError)
	if !ok {
		return fmt.Errorf("semgrep scan failed: %w", err)
	}

	// Exit code 1 means findings were found - this is valid output, not an error
	if exitError.ExitCode() == 1 {
		return nil
	}

	return cmdrun.FormatError(err, stderr, "semgrep", "scan")
}

// ParseFromReader parses semgrep JSON output from a reader.
// This is used when piping semgrep output directly to confvis.
func ParseFromReader(r io.Reader) (*Report, error) {
	var report Report
	if err := json.NewDecoder(r).Decode(&report); err != nil {
		return nil, fmt.Errorf("parsing semgrep output: %w", err)
	}
	return &report, nil
}
