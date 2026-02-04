package trivy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
	factors := []confidence.Factor{
		{
			Name:        "Critical Vulnerabilities",
			Score:       scoring.SeverityScore(counts.Critical, PenaltyCritical),
			Weight:      WeightCritical,
			Description: fmt.Sprintf("%d critical", counts.Critical),
		},
		{
			Name:        "High Vulnerabilities",
			Score:       scoring.SeverityScore(counts.High, PenaltyHigh),
			Weight:      WeightHigh,
			Description: fmt.Sprintf("%d high", counts.High),
		},
		{
			Name:        "Medium Vulnerabilities",
			Score:       scoring.SeverityScore(counts.Medium, PenaltyMedium),
			Weight:      WeightMedium,
			Description: fmt.Sprintf("%d medium", counts.Medium),
		},
		{
			Name:        "Low Vulnerabilities",
			Score:       scoring.SeverityScore(counts.Low, PenaltyLow),
			Weight:      WeightLow,
			Description: fmt.Sprintf("%d low", counts.Low),
		},
	}

	// Build report
	result := &confidence.Report{
		Title:       title,
		Threshold:   opts.Threshold,
		Source:      sourceName,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Factors:     factors,
	}

	// Calculate weighted score
	result.Score = result.CalculateScore()

	return result, nil
}
