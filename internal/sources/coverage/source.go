package coverage

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/httpclient"
	"github.com/boinger/confvis/internal/sources/repoparse"
)

// SourceConfig defines configuration for a coverage source.
type SourceConfig struct {
	Name          string
	TokenEnvVar   string
	TokenRequired bool
	BaseURL       string
	BuildAPIPath  func(service, owner, repo string) string
	BuildWebURL   func(service, owner, repo string) string
}

// ExtractCoverageFunc extracts coverage percentage from raw JSON response data.
type ExtractCoverageFunc func(data []byte) (float64, error)

// CoverageSource implements sources.Source for coverage providers.
type CoverageSource struct {
	config          SourceConfig
	extractCoverage ExtractCoverageFunc
}

// NewSource creates a new coverage source with the given configuration.
func NewSource(cfg SourceConfig, extractor ExtractCoverageFunc) *CoverageSource {
	return &CoverageSource{
		config:          cfg,
		extractCoverage: extractor,
	}
}

// Name returns the source identifier.
func (s *CoverageSource) Name() string {
	return s.config.Name
}

var defaultTimeout = 30 * time.Second

// Fetch retrieves coverage metrics and converts them to a confidence report.
func (s *CoverageSource) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
	resolver := &sources.ConfigResolver{
		SourceName:     s.config.Name,
		TokenEnvVar:    s.config.TokenEnvVar,
		TokenRequired:  s.config.TokenRequired,
		DefaultTimeout: defaultTimeout,
	}

	cfg, err := resolver.Resolve(opts)
	if err != nil {
		return nil, err
	}

	service := ResolveService(opts)

	owner, repo, err := repoparse.Parse(opts.Project)
	if err != nil {
		return nil, err
	}

	client := s.createClient(cfg.Token, cfg.Timeout)
	endpoint := s.config.BaseURL + s.config.BuildAPIPath(service, owner, repo)

	rawResponse, err := httpclient.Get[json.RawMessage](client, ctx, endpoint)
	if err != nil {
		return nil, err
	}

	coverage, err := s.extractCoverage(rawResponse)
	if err != nil {
		return nil, err
	}

	title := sources.ResolveTitle(opts.Title, opts.Project)
	reportURL := s.config.BuildWebURL(service, owner, repo)

	return BuildReport(title, s.config.Name, opts.Threshold, coverage, reportURL), nil
}

// createClient creates an HTTP client for the coverage API.
func (s *CoverageSource) createClient(token string, timeout time.Duration) *httpclient.Client {
	authType := httpclient.AuthNone
	if token != "" {
		authType = httpclient.AuthBearer
	}

	return httpclient.New(httpclient.Config{
		BaseURL:  s.config.BaseURL,
		Token:    token,
		AuthType: authType,
		Accept:   "application/json",
		Timeout:  timeout,
	})
}

// FetchWithTestClient is a helper for testing that uses a mock HTTP server.
func (s *CoverageSource) FetchWithTestClient(ctx context.Context, opts sources.Options, baseURL, token string, httpClient *http.Client) (*confidence.Report, error) {
	service := ResolveService(opts)

	owner, repo, err := repoparse.Parse(opts.Project)
	if err != nil {
		return nil, err
	}

	authType := httpclient.AuthNone
	if token != "" {
		authType = httpclient.AuthBearer
	}

	client := httpclient.NewWithHTTPClient(httpclient.Config{
		BaseURL:  baseURL,
		Token:    token,
		AuthType: authType,
		Accept:   "application/json",
	}, httpClient)

	endpoint := baseURL + s.config.BuildAPIPath(service, owner, repo)

	rawResponse, err := httpclient.Get[json.RawMessage](client, ctx, endpoint)
	if err != nil {
		return nil, err
	}

	coverage, err := s.extractCoverage(rawResponse)
	if err != nil {
		return nil, err
	}

	title := sources.ResolveTitle(opts.Title, opts.Project)
	reportURL := s.config.BuildWebURL(service, owner, repo)

	return BuildReport(title, s.config.Name, opts.Threshold, coverage, reportURL), nil
}
