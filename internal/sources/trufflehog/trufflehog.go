package trufflehog

import (
	"context"
	"fmt"
	"strings"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/scoring"
)

const sourceName = "trufflehog"

// Fetcher defines the interface for fetching TruffleHog data.
type Fetcher interface {
	Scan(ctx context.Context, path string) ([]Finding, error)
	ScanGit(ctx context.Context, repoURL string) ([]Finding, error)
}

// Environment variable names for configuration.
const (
	EnvCommand = "TRUFFLEHOG_CMD"
)

// Source implements the sources.Source interface for TruffleHog.
type Source struct{}

func init() {
	sources.Register(&Source{})
}

// Name returns the source identifier.
func (s *Source) Name() string {
	return sourceName
}

// Fetch runs TruffleHog on the specified path and converts results to a confidence report.
func (s *Source) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
	command := sources.ResolveCommand(opts, "trufflehog-cmd", EnvCommand)

	// Path/URL is provided via Project field
	target := opts.Project
	if target == "" {
		target = "."
	}

	// Check if scanning git or filesystem
	scanMode := sources.GetExtra(opts, "mode", "filesystem")

	client := NewClient(command)

	return s.FetchWithClient(ctx, client, opts, target, scanMode)
}

// FetchWithClient runs TruffleHog using the provided Fetcher.
// This allows injecting mock clients for testing.
func (s *Source) FetchWithClient(ctx context.Context, fetcher Fetcher, opts sources.Options, target, scanMode string) (*confidence.Report, error) {
	var findings []Finding
	var err error

	// Run appropriate scan based on mode
	if scanMode == "git" || isGitURL(target) {
		findings, err = fetcher.ScanGit(ctx, target)
	} else {
		findings, err = fetcher.Scan(ctx, target)
	}

	if err != nil {
		return nil, fmt.Errorf("scanning with trufflehog: %w", err)
	}

	// Count findings by verification status
	counts := countFindingsByVerification(findings)

	// Determine title - trufflehog targets can be paths or git URLs
	title := deriveTitleWithOpts(target, opts.Title)

	// Build factors with verification-based scoring
	factors := scoring.BuildSeverityFactors([]scoring.SeverityConfig{
		{Name: "Verified Secrets", Count: counts.Verified, Penalty: penaltyVerified, Weight: weightVerified},
		{Name: "Unverified Secrets", Count: counts.Unverified, Penalty: penaltyUnverified, Weight: weightUnverified},
	}, "")

	return scoring.BuildReport(title, sourceName, opts.Threshold, factors)
}

// deriveTitleWithOpts extracts a title from the target path or URL.
func deriveTitleWithOpts(target, explicitTitle string) string {
	if explicitTitle != "" {
		return explicitTitle
	}
	if isGitURL(target) {
		// Extract repo name from URL
		parts := strings.Split(strings.TrimSuffix(target, ".git"), "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
		return target
	}
	// For filesystem paths, use the shared helper
	return sources.DeriveTitleFromPath(target, "")
}

// isGitURL checks if the target looks like a git URL.
func isGitURL(target string) bool {
	return strings.HasPrefix(target, "http://") ||
		strings.HasPrefix(target, "https://") ||
		strings.HasPrefix(target, "git@") ||
		strings.HasSuffix(target, ".git")
}
