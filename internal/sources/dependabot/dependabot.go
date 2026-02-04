package dependabot

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/scoring"
)

const sourceName = "dependabot"

// Fetcher defines the interface for fetching Dependabot data.
type Fetcher interface {
	FetchAlerts(ctx context.Context, owner, repo string) (AlertsResponse, error)
	AlertsURL(owner, repo string) string
}

// Environment variable names for configuration.
const (
	EnvToken         = "DEPENDABOT_TOKEN"
	EnvTokenFallback = "GITHUB_TOKEN"
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
	PenaltyCritical = 25
	PenaltyHigh     = 15
	PenaltyMedium   = 5
	PenaltyLow      = 2
)

// Factor weights.
const (
	WeightCritical = 40
	WeightHigh     = 30
	WeightMedium   = 20
	WeightLow      = 10
)

// Source implements the sources.Source interface for Dependabot.
type Source struct{}

func init() {
	sources.Register(&Source{})
}

// Name returns the source identifier.
func (s *Source) Name() string {
	return sourceName
}

// Fetch retrieves vulnerability metrics from Dependabot and converts them to a confidence report.
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
		return nil, fmt.Errorf("dependabot token required: use --token flag or set %s (or %s)", EnvToken, EnvTokenFallback)
	}

	// Parse owner/repo from project (format: owner/repo)
	owner, repo, err := parseRepository(opts.Project)
	if err != nil {
		return nil, err
	}

	client := NewClient(cfg.URL, token, cfg.Timeout)

	return s.FetchWithClient(ctx, client, opts, owner, repo)
}

// FetchWithClient retrieves vulnerability metrics using the provided Fetcher.
// This allows injecting mock clients for testing.
func (s *Source) FetchWithClient(ctx context.Context, fetcher Fetcher, opts sources.Options, owner, repo string) (*confidence.Report, error) {
	// Fetch alerts
	alerts, err := fetcher.FetchAlerts(ctx, owner, repo)
	if err != nil {
		return nil, err
	}

	// Count alerts by severity
	counts := CountAlertsBySeverity(alerts)

	// Determine title
	title := opts.Title
	if title == "" {
		title = fmt.Sprintf("%s/%s", owner, repo)
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
	alertsURL := fetcher.AlertsURL(owner, repo)
	factors := scoring.BuildSeverityFactors(configs, alertsURL)

	return scoring.BuildReport(title, sourceName, opts.Threshold, factors), nil
}

// parseRepository splits "owner/repo" into separate parts.
func parseRepository(project string) (owner, repo string, err error) {
	if project == "" {
		return "", "", fmt.Errorf("repository required: use --project owner/repo")
	}

	parts := strings.SplitN(project, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repository format %q: expected owner/repo", project)
	}

	return parts[0], parts[1], nil
}
