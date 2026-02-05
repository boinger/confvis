package trufflehog

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/boinger/confvis/internal/sources/cmdrun"
)

// DefaultCommand is the default trufflehog command.
const DefaultCommand = "trufflehog"

// Client wraps trufflehog CLI execution.
type Client struct {
	command string
}

// NewClient creates a new TruffleHog CLI client.
// If command is empty, uses "trufflehog" as the default.
func NewClient(command string) *Client {
	if command == "" {
		command = DefaultCommand
	}
	return &Client{command: command}
}

// Scan runs trufflehog on the specified path and returns the parsed findings.
func (c *Client) Scan(ctx context.Context, path string) ([]Finding, error) {
	// Build command arguments: trufflehog filesystem --json <path>
	args := []string{
		"filesystem",
		"--json",
		path,
	}

	result, err := cmdrun.Run(ctx, c.command, args, "trufflehog")
	// TruffleHog returns exit code 183 when secrets are found
	if err != nil {
		if fatalErr := checkTrufflehogError(err, result.Stderr); fatalErr != nil {
			return nil, fatalErr
		}
	}

	// Empty output means no findings
	stdout := bytes.TrimSpace(result.Stdout)
	if len(stdout) == 0 {
		return []Finding{}, nil
	}

	// Parse JSON lines output (one JSON object per line)
	return parseJSONLines(stdout)
}

// ScanGit runs trufflehog on a git repository.
func (c *Client) ScanGit(ctx context.Context, repoURL string) ([]Finding, error) {
	// Build command arguments: trufflehog git --json <repo-url>
	args := []string{
		"git",
		"--json",
		repoURL,
	}

	result, err := cmdrun.Run(ctx, c.command, args, "trufflehog")
	if err != nil {
		if fatalErr := checkTrufflehogError(err, result.Stderr); fatalErr != nil {
			return nil, fatalErr
		}
	}

	stdout := bytes.TrimSpace(result.Stdout)
	if len(stdout) == 0 {
		return []Finding{}, nil
	}

	return parseJSONLines(stdout)
}

// checkTrufflehogError returns nil if the error is acceptable,
// otherwise returns a formatted error.
func checkTrufflehogError(err error, stderr []byte) error {
	exitError, ok := err.(*exec.ExitError)
	if !ok {
		return fmt.Errorf("trufflehog scan failed: %w", err)
	}

	// Exit code 183 means secrets were found - this is valid output, not an error
	if exitError.ExitCode() == 183 {
		return nil
	}

	return cmdrun.FormatError(err, stderr, "trufflehog", "scan")
}

// parseJSONLines parses TruffleHog's JSON lines output.
func parseJSONLines(data []byte) ([]Finding, error) {
	var findings []Finding
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var finding Finding
		if err := json.Unmarshal(line, &finding); err != nil {
			return nil, fmt.Errorf("parsing trufflehog finding: %w", err)
		}
		findings = append(findings, finding)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading trufflehog output: %w", err)
	}

	return findings, nil
}

// countFindingsByVerification counts findings grouped by verification status.
func countFindingsByVerification(findings []Finding) FindingCounts {
	var counts FindingCounts
	for _, finding := range findings {
		if finding.Verified {
			counts.Verified++
		} else {
			counts.Unverified++
		}
	}
	return counts
}
