package codeql

import (
	"context"
	"time"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/githubalerts"
	"github.com/boinger/confvis/internal/sources/repoparse"
	"github.com/boinger/confvis/internal/sources/scoring"
)

const sourceName = "codeql"

// Fetcher defines the interface for fetching CodeQL data.
type Fetcher interface {
	FetchAlerts(ctx context.Context, owner, repo, toolName string) (AlertsResponse, error)
	AlertsURL(owner, repo string) string
}

// Environment variable names for configuration.
const (
	EnvToken  = "CODEQL_TOKEN" // #nosec G101 -- not a credential, just env var name
	EnvAPIURL = "GITHUB_API_URL"
)

var configResolver = &sources.ConfigResolver{
	SourceName:     sourceName,
	TokenEnvVar:    EnvToken,
	URLEnvVar:      EnvAPIURL,
	TokenRequired:  false, // We handle token resolution manually for fallback
	URLRequired:    false, // Has default
	DefaultTimeout: 30 * time.Second,
}

// Source implements the sources.Source interface for CodeQL.
type Source struct{}

func init() {
	sources.Register(&Source{})
}

// Name returns the source identifier.
func (s *Source) Name() string {
	return sourceName
}

// Fetch retrieves code scanning alerts from CodeQL and converts them to a confidence report.
func (s *Source) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
	cfg, err := configResolver.Resolve(opts)
	if err != nil {
		return nil, err
	}

	token, err := githubalerts.ResolveTokenWithFallback(cfg, sourceName, EnvToken)
	if err != nil {
		return nil, err
	}

	owner, repo, err := repoparse.Parse(opts.Project)
	if err != nil {
		return nil, err
	}

	toolName := sources.GetExtra(opts, "tool", "")
	client := NewClient(cfg.URL, token, cfg.Timeout)

	return s.FetchWithClient(ctx, client, opts, owner, repo, toolName)
}

// FetchWithClient retrieves code scanning alerts using the provided Fetcher.
func (s *Source) FetchWithClient(ctx context.Context, fetcher Fetcher, opts sources.Options, owner, repo, toolName string) (*confidence.Report, error) {
	alerts, err := fetcher.FetchAlerts(ctx, owner, repo, toolName)
	if err != nil {
		return nil, err
	}

	counts := countAlertsBySeverity(alerts)
	title := githubalerts.ResolveTitle(opts.Title, owner, repo)

	factors := scoring.BuildVulnFactors(
		scoring.SeverityCounts{Critical: counts.Critical, High: counts.High, Medium: counts.Medium, Low: counts.Low},
		githubalerts.Penalties(),
		githubalerts.Weights(),
		fetcher.AlertsURL(owner, repo),
	)

	return scoring.BuildReport(title, sourceName, opts.Threshold, factors), nil
}
