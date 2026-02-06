package gosec

import (
	"context"
	"fmt"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/scoring"
)

const sourceName = "gosec"

// Fetcher defines the interface for fetching Gosec data.
type Fetcher interface {
	Scan(ctx context.Context, path string) (*Report, error)
}

// Environment variable names for configuration.
const (
	EnvCommand = "GOSEC_CMD"
)

// Source implements the sources.Source interface for Gosec.
type Source struct{}

func init() {
	sources.Register(&Source{})
}

// Name returns the source identifier.
func (s *Source) Name() string {
	return sourceName
}

// Fetch runs Gosec on the specified path and converts results to a confidence report.
func (s *Source) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
	command := sources.ResolveCommand(opts, "gosec-cmd", EnvCommand)

	// Path is provided via Project field (matches CLI pattern of -p flag)
	path := opts.Project
	if path == "" {
		path = "./..."
	}

	client := NewClient(command)

	return s.FetchWithClient(ctx, client, opts, path)
}

// FetchWithClient runs Gosec using the provided Fetcher.
// This allows injecting mock clients for testing.
func (s *Source) FetchWithClient(ctx context.Context, fetcher Fetcher, opts sources.Options, path string) (*confidence.Report, error) {
	// Run scan
	report, err := fetcher.Scan(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("scanning with gosec: %w", err)
	}

	// Count findings by severity
	counts := countFromIssues(report.Issues)

	// Determine title - gosec uses "./..." pattern, so derive from parent dir
	effectivePath := path
	if path == "./..." {
		effectivePath = "."
	}
	title := sources.DeriveTitleFromPath(effectivePath, opts.Title)

	// Build factors with severity-based scoring
	factors := scoring.BuildSeverityFactors([]scoring.SeverityConfig{
		{Name: "High Severity", Count: counts.High, Penalty: penaltyHigh, Weight: weightHigh},
		{Name: "Medium Severity", Count: counts.Medium, Penalty: penaltyMedium, Weight: weightMedium},
		{Name: "Low Severity", Count: counts.Low, Penalty: penaltyLow, Weight: weightLow},
	}, "")

	return scoring.BuildReport(title, sourceName, opts.Threshold, factors), nil
}
