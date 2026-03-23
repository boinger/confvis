// Package githubalerts provides shared infrastructure for GitHub security alerts sources.
package githubalerts

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/httpclient"
)

// Severity penalties for GitHub security alerts (Dependabot, CodeQL).
// Softer than the default strict-scanner penalties (33/20/10/5) because GitHub
// advisory-based alerts have a higher false-positive rate and often flag
// transitive dependencies the project doesn't directly exercise. The ~25-60%
// reduction reflects that an advisory "critical" is less certain than a scanner
// like Grype confirming a vulnerable call path.
const (
	PenaltyCritical = 25
	PenaltyHigh     = 15
	PenaltyMedium   = 5
	PenaltyLow      = 2
)

// Factor weights for GitHub security alerts.
// Weights match the defaults — the relative importance of severity tiers is the
// same; only the per-issue penalties are softer.
const (
	WeightCritical = 40
	WeightHigh     = 30
	WeightMedium   = 20
	WeightLow      = 10
)

// Penalties returns the standard penalty array for GitHub alerts.
func Penalties() [4]int {
	return [4]int{PenaltyCritical, PenaltyHigh, PenaltyMedium, PenaltyLow}
}

// Weights returns the standard weight array for GitHub alerts.
func Weights() [4]int {
	return [4]int{WeightCritical, WeightHigh, WeightMedium, WeightLow}
}

// Config defines the configuration for a GitHub alerts source.
type Config struct {
	EndpointPath string // e.g., "dependabot/alerts" or "code-scanning/alerts"
	WebURLPath   string // e.g., "security/dependabot" or "security/code-scanning"
}

// Client is an HTTP client for GitHub security alerts APIs.
type Client struct {
	BaseURL string
	HTTP    *httpclient.Client
	Config  Config
}

// NewClient creates a new GitHub alerts API client.
func NewClient(baseURL, token string, timeout time.Duration, config Config) *Client {
	cfg := httpclient.GitHubConfig(baseURL, token, timeout)
	return &Client{
		BaseURL: cfg.BaseURL,
		HTTP:    httpclient.New(cfg),
		Config:  config,
	}
}

// NewClientWithHTTP creates a new client with a custom HTTP client.
// This is primarily intended for testing.
func NewClientWithHTTP(baseURL, token string, httpClient *http.Client, config Config) *Client {
	cfg := httpclient.GitHubConfig(baseURL, token, 0)
	return &Client{
		BaseURL: cfg.BaseURL,
		HTTP:    httpclient.NewWithHTTPClient(cfg, httpClient),
		Config:  config,
	}
}

// BuildEndpoint constructs the API endpoint URL with query parameters.
func (c *Client) BuildEndpoint(owner, repo string, params url.Values) string {
	return fmt.Sprintf("%s/repos/%s/%s/%s?%s",
		c.BaseURL, url.PathEscape(owner), url.PathEscape(repo), c.Config.EndpointPath, params.Encode())
}

// AlertsURL returns the web URL for alerts in a repository.
func (c *Client) AlertsURL(owner, repo string) string {
	host := "github.com"
	if c.BaseURL != httpclient.GitHubDefaultURL {
		if u, err := url.Parse(c.BaseURL); err == nil {
			host = u.Host
			host = strings.TrimPrefix(host, "api.")
		}
	}
	return fmt.Sprintf("https://%s/%s/%s/%s", host, owner, repo, c.Config.WebURLPath)
}

// ResolveTokenWithFallback resolves a token from the config, falling back to GITHUB_TOKEN.
func ResolveTokenWithFallback(cfg *sources.ResolvedConfig, sourceName, primaryEnv string) (string, error) {
	token := cfg.Token
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		return "", fmt.Errorf("%s token required: use --token flag or set %s (or GITHUB_TOKEN)", sourceName, primaryEnv)
	}
	return token, nil
}

// ResolveTitle returns the explicit title or formats owner/repo.
func ResolveTitle(explicit, owner, repo string) string {
	if explicit != "" {
		return explicit
	}
	return fmt.Sprintf("%s/%s", owner, repo)
}
