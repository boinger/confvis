package codeql

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
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
	EnvToken         = "CODEQL_TOKEN"  // #nosec G101 -- not a credential, just env var name
	EnvTokenFallback = "GITHUB_TOKEN"  // #nosec G101 -- not a credential, just env var name
	EnvAPIURL        = "GITHUB_API_URL"
)

var configResolver = &sources.ConfigResolver{
	SourceName:     sourceName,
	TokenEnvVar:    EnvToken,
	URLEnvVar:      EnvAPIURL,
	TokenRequired:  false, // We handle token resolution manually for fallback
	URLRequired:    false, // Has default
	DefaultTimeout: 30 * time.Second,
}

// Severity penalties (points deducted per issue).
const (
	penaltyCritical = 25
	penaltyHigh     = 15
	penaltyMedium   = 5
	penaltyLow      = 2
)

// Factor weights.
const (
	weightCritical = 40
	weightHigh     = 30
	weightMedium   = 20
	weightLow      = 10
)

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

	// Resolve token with fallback to GITHUB_TOKEN
	token := cfg.Token
	if token == "" {
		token = os.Getenv(EnvTokenFallback)
	}
	if token == "" {
		return nil, fmt.Errorf("codeql token required: use --token flag or set %s (or %s)", EnvToken, EnvTokenFallback)
	}

	// Parse owner/repo from project (format: owner/repo)
	owner, repo, err := repoparse.Parse(opts.Project)
	if err != nil {
		return nil, err
	}

	// Get optional tool filter from Extra options
	toolName := opts.Extra["tool"]

	client := NewClient(cfg.URL, token, cfg.Timeout)

	return s.FetchWithClient(ctx, client, opts, owner, repo, toolName)
}

// FetchWithClient retrieves code scanning alerts using the provided Fetcher.
// This allows injecting mock clients for testing.
func (s *Source) FetchWithClient(ctx context.Context, fetcher Fetcher, opts sources.Options, owner, repo, toolName string) (*confidence.Report, error) {
	// Fetch alerts
	alerts, err := fetcher.FetchAlerts(ctx, owner, repo, toolName)
	if err != nil {
		return nil, err
	}

	// Count alerts by severity
	counts := countAlertsBySeverity(alerts)

	// Determine title
	title := opts.Title
	if title == "" {
		title = fmt.Sprintf("%s/%s", owner, repo)
	}

	// Build factors with severity-based scoring
	factors := scoring.BuildVulnFactors(
		scoring.SeverityCounts{Critical: counts.Critical, High: counts.High, Medium: counts.Medium, Low: counts.Low},
		[4]int{penaltyCritical, penaltyHigh, penaltyMedium, penaltyLow},
		[4]int{weightCritical, weightHigh, weightMedium, weightLow},
		fetcher.AlertsURL(owner, repo),
	)

	return scoring.BuildReport(title, sourceName, opts.Threshold, factors), nil
}
