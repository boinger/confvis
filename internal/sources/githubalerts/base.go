// Package githubalerts provides shared infrastructure for GitHub security alerts sources.
package githubalerts

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/boinger/confvis/internal/sources/httpclient"
)

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
