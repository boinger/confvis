package gitleaks

import (
	"context"
	"fmt"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/scoring"
)

const sourceName = "gitleaks"

// Fetcher defines the interface for fetching GitLeaks data.
type Fetcher interface {
	Scan(ctx context.Context, path string) (Report, error)
}

// Environment variable names for configuration.
const (
	EnvCommand = "GITLEAKS_CMD"
)

// Source implements the sources.Source interface for GitLeaks.
type Source struct{}

func init() {
	sources.Register(&Source{})
}

// Name returns the source identifier.
func (s *Source) Name() string {
	return sourceName
}

// Fetch runs GitLeaks on the specified path and converts results to a confidence report.
func (s *Source) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
	command := sources.ResolveCommand(opts, "gitleaks-cmd", EnvCommand)

	// Path is provided via Project field (matches CLI pattern of -p flag)
	path := opts.Project
	if path == "" {
		path = "."
	}

	client := NewClient(command)

	return s.FetchWithClient(ctx, client, opts, path)
}

// FetchWithClient runs GitLeaks using the provided Fetcher.
// This allows injecting mock clients for testing.
func (s *Source) FetchWithClient(ctx context.Context, fetcher Fetcher, opts sources.Options, path string) (*confidence.Report, error) {
	// Run scan
	report, err := fetcher.Scan(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("scanning with gitleaks: %w", err)
	}

	// Count findings
	counts := FindingCounts{
		Secrets: len(report),
	}

	// Determine title
	title := sources.DeriveTitleFromPath(path, opts.Title)

	// Build factor with secrets count
	// Each secret deducts points; no secrets = 100
	score := scoring.SeverityScore(counts.Secrets, penaltyPerSecret)

	factors := []confidence.Factor{
		{
			Name:        "Leaked Secrets",
			Score:       score,
			Weight:      weightSecrets,
			Description: fmt.Sprintf("%d secrets detected", counts.Secrets),
		},
	}

	return scoring.BuildReport(title, sourceName, opts.Threshold, factors), nil
}
