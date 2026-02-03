package codecov

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
)

const sourceName = "codecov"

// Fetcher defines the interface for fetching Codecov data.
type Fetcher interface {
	FetchReport(ctx context.Context, service, ownerRepo string) (*ReportResponse, error)
	ReportURL(service, ownerRepo string) string
}

// Environment variable names for configuration.
const (
	EnvToken = "CODECOV_TOKEN"
)

// Source implements the sources.Source interface for Codecov.
type Source struct{}

func init() {
	sources.Register(&Source{})
}

// Name returns the source identifier.
func (s *Source) Name() string {
	return sourceName
}

// Fetch retrieves coverage metrics from Codecov and converts them to a confidence report.
func (s *Source) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
	// Resolve token from option or environment
	token := opts.Token
	if token == "" {
		token = os.Getenv(EnvToken)
	}
	if token == "" {
		return nil, fmt.Errorf("codecov token required: use --token flag or set %s", EnvToken)
	}

	// Get service from Extra options, default to github
	service := "github"
	if opts.Extra != nil {
		if svc, ok := opts.Extra["service"]; ok && svc != "" {
			service = svc
		}
	}

	// Create client with timeout
	timeout := time.Duration(opts.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := NewClient(token, timeout)

	return s.FetchWithClient(ctx, client, opts, service)
}

// FetchWithClient retrieves coverage metrics using the provided Fetcher.
// This allows injecting mock clients for testing.
func (s *Source) FetchWithClient(ctx context.Context, fetcher Fetcher, opts sources.Options, service string) (*confidence.Report, error) {
	// Fetch report
	report, err := fetcher.FetchReport(ctx, service, opts.Project)
	if err != nil {
		return nil, err
	}

	// Determine title
	title := opts.Title
	if title == "" {
		title = opts.Project
	}

	// Build factors
	factors := []confidence.Factor{
		{
			Name:   "Code Coverage",
			Score:  int(report.Totals.Coverage),
			Weight: 100,
			URL:    fetcher.ReportURL(service, opts.Project),
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

	// Calculate weighted score (just the coverage in this case)
	result.Score = result.CalculateScore()

	return result, nil
}
