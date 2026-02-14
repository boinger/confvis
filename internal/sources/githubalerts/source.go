package githubalerts

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/httpclient"
	"github.com/boinger/confvis/internal/sources/repoparse"
	"github.com/boinger/confvis/internal/sources/scoring"
)

// SourceConfig defines configuration for a GitHub alerts source.
type SourceConfig struct {
	Name         string // Source name (e.g., "dependabot", "codeql")
	TokenEnvVar  string // Environment variable for source-specific token
	EndpointPath string // API endpoint path (e.g., "dependabot/alerts")
	WebURLPath   string // Web URL path (e.g., "security/dependabot")
}

// CountAlertsFunc extracts severity counts from raw JSON alert data.
type CountAlertsFunc func(data []byte) (scoring.SeverityCounts, error)

// ExtraParamsFunc adds source-specific query parameters.
type ExtraParamsFunc func(opts sources.Options) url.Values

// AlertsSource implements sources.Source for GitHub security alerts APIs.
type AlertsSource struct {
	config      SourceConfig
	countAlerts CountAlertsFunc
	extraParams ExtraParamsFunc
}

// NewSource creates a new GitHub alerts source with the given configuration.
// countAlerts is called to extract severity counts from the raw JSON response.
// extraParams is optional and can add source-specific query parameters.
func NewSource(cfg SourceConfig, countAlerts CountAlertsFunc, extraParams ExtraParamsFunc) *AlertsSource {
	return &AlertsSource{
		config:      cfg,
		countAlerts: countAlerts,
		extraParams: extraParams,
	}
}

// Name returns the source identifier.
func (s *AlertsSource) Name() string {
	return s.config.Name
}

var defaultTimeout = 30 * time.Second

// Fetch retrieves alerts from GitHub and converts them to a confidence report.
func (s *AlertsSource) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
	resolver := &sources.ConfigResolver{
		SourceName:     s.config.Name,
		TokenEnvVar:    s.config.TokenEnvVar,
		URLEnvVar:      "GITHUB_API_URL",
		TokenRequired:  false, // We handle token resolution manually for fallback
		URLRequired:    false, // Has default
		DefaultTimeout: defaultTimeout,
	}

	cfg, err := resolver.Resolve(opts)
	if err != nil {
		return nil, err
	}

	token, err := ResolveTokenWithFallback(cfg, s.config.Name, s.config.TokenEnvVar)
	if err != nil {
		return nil, err
	}

	owner, repo, err := repoparse.Parse(opts.Project)
	if err != nil {
		return nil, err
	}

	alertsConfig := Config{
		EndpointPath: s.config.EndpointPath,
		WebURLPath:   s.config.WebURLPath,
	}
	client := NewClient(cfg.URL, token, cfg.Timeout, alertsConfig)

	allAlerts, err := s.fetchAllAlerts(ctx, client, owner, repo, opts)
	if err != nil {
		return nil, err
	}

	counts, err := s.countAlerts(allAlerts)
	if err != nil {
		return nil, err
	}

	title := ResolveTitle(opts.Title, owner, repo)
	factors := scoring.BuildVulnFactors(counts, Penalties(), Weights(), client.AlertsURL(owner, repo))

	return scoring.BuildReport(title, s.config.Name, opts.Threshold, factors), nil
}

// fetchAllAlerts retrieves all open alerts with pagination.
func (s *AlertsSource) fetchAllAlerts(ctx context.Context, client *Client, owner, repo string, opts sources.Options) ([]byte, error) {
	const perPage = 100
	var allAlerts []json.RawMessage

	for page := 1; ; page++ {
		params := url.Values{
			"state":    {"open"},
			"per_page": {strconv.Itoa(perPage)},
			"page":     {strconv.Itoa(page)},
		}

		// Add source-specific parameters
		if s.extraParams != nil {
			for k, v := range s.extraParams(opts) {
				params[k] = v
			}
		}

		endpoint := client.BuildEndpoint(owner, repo, params)

		pageAlerts, err := httpclient.Get[[]json.RawMessage](client.HTTP, ctx, endpoint)
		if err != nil {
			return nil, err
		}

		allAlerts = append(allAlerts, pageAlerts...)

		if len(pageAlerts) < perPage {
			break
		}
	}

	return json.Marshal(allAlerts)
}
