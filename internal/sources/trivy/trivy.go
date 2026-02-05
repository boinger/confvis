package trivy

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/scoring"
)

const sourceName = "trivy"

// Environment variable names for configuration.
const (
	EnvCommand = "TRIVY_CMD"
)

// Source implements the sources.Source interface for Trivy.
type Source struct{}

func init() {
	sources.Register(&Source{})
}

// Name returns the source identifier.
func (s *Source) Name() string {
	return sourceName
}

// Fetch runs Trivy on the specified path and converts results to a confidence report.
func (s *Source) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
	command := sources.ResolveCommand(opts, "trivy-cmd", EnvCommand)

	// Path is provided via Project field (matches CLI pattern of -p flag)
	path := opts.Project
	if path == "" {
		path = "."
	}

	// Resolve to absolute path for clearer output
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	// Create client
	client := NewClient(command)

	// Run scan
	report, err := client.Scan(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("scanning with trivy: %w", err)
	}

	// Aggregate counts
	counts := countFromResults(report.Results)

	// Determine title
	title := opts.Title
	if title == "" {
		title = filepath.Base(absPath)
	}

	// Build factors with severity-based scoring
	factors := scoring.BuildVulnFactors(
		counts,
		scoring.DefaultPenalties(),
		scoring.DefaultWeights(),
		"",
	)

	return scoring.BuildReport(title, sourceName, opts.Threshold, factors), nil
}
