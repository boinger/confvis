package trivy

import (
	"context"
	"fmt"
	"os"
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
	// Resolve command from Extra options or environment
	command := ""
	if opts.Extra != nil {
		command = opts.Extra["trivy-cmd"]
	}
	if command == "" {
		command = os.Getenv(EnvCommand)
	}

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
	counts := CountFromResults(report.Results)

	// Determine title
	title := opts.Title
	if title == "" {
		title = filepath.Base(absPath)
	}

	// Build factors with severity-based scoring
	severityCounts := scoring.SeverityCounts{
		Critical: counts.Critical,
		High:     counts.High,
		Medium:   counts.Medium,
		Low:      counts.Low,
	}
	penalties := [4]int{PenaltyCritical, PenaltyHigh, PenaltyMedium, PenaltyLow}
	weights := [4]int{WeightCritical, WeightHigh, WeightMedium, WeightLow}

	configs := scoring.VulnSeverityConfigs(severityCounts, penalties, weights)
	factors := scoring.BuildSeverityFactors(configs, "")

	return scoring.BuildReport(title, sourceName, opts.Threshold, factors), nil
}
