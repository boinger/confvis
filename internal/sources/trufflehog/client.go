package trufflehog

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/boinger/confvis/internal/sources/cmdrun"
)

// Compile-time interface compliance check.
var _ Fetcher = (*Client)(nil)

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
		if fatalErr := cmdrun.CheckAcceptableExitCode(err, []int{183}, result.Stderr, "trufflehog", "scan"); fatalErr != nil {
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
		if fatalErr := cmdrun.CheckAcceptableExitCode(err, []int{183}, result.Stderr, "trufflehog", "scan"); fatalErr != nil {
			return nil, fatalErr
		}
	}

	stdout := bytes.TrimSpace(result.Stdout)
	if len(stdout) == 0 {
		return []Finding{}, nil
	}

	return parseJSONLines(stdout)
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
