package coveralls

import (
	"context"
	"math"
	"time"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/scoring"
)

const sourceName = "coveralls"

// Fetcher defines the interface for fetching Coveralls data.
type Fetcher interface {
	FetchReport(ctx context.Context, service, ownerRepo string) (*ReportResponse, error)
	ReportURL(service, ownerRepo string) string
}

// Environment variable names for configuration.
const (
	EnvToken = "COVERALLS_TOKEN"
)

var configResolver = &sources.ConfigResolver{
	SourceName:     sourceName,
	TokenEnvVar:    EnvToken,
	TokenRequired:  false, // Public repos don't require a token
	DefaultTimeout: 30 * time.Second,
}

// Source implements the sources.Source interface for Coveralls.
type Source struct{}

func init() {
	sources.Register(&Source{})
}

// Name returns the source identifier.
func (s *Source) Name() string {
	return sourceName
}

// Fetch retrieves coverage metrics from Coveralls and converts them to a confidence report.
func (s *Source) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
	cfg, err := configResolver.Resolve(opts)
	if err != nil {
		return nil, err
	}

	// Get service from Extra options, default to github
	service := "github"
	if opts.Extra != nil {
		if svc, ok := opts.Extra["service"]; ok && svc != "" {
			service = svc
		}
	}

	client := NewClient(cfg.Token, cfg.Timeout)

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
			Score:  int(math.Round(report.CoveredPercent)),
			Weight: 100,
			URL:    fetcher.ReportURL(service, opts.Project),
		},
	}

	return scoring.BuildReport(title, sourceName, opts.Threshold, factors), nil
}
