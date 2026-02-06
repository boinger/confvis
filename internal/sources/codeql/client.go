package codeql

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/boinger/confvis/internal/sources/httpclient"
	"github.com/boinger/confvis/internal/sources/scoring"
)

// Client is an HTTP client for the GitHub Code Scanning API.
type Client struct {
	baseURL string
	http    *httpclient.Client
}

// NewClient creates a new CodeQL API client.
func NewClient(baseURL, token string, timeout time.Duration) *Client {
	cfg := httpclient.GitHubConfig(baseURL, token, timeout)
	return &Client{
		baseURL: cfg.BaseURL,
		http:    httpclient.New(cfg),
	}
}

// NewClientWithHTTP creates a new client with a custom HTTP client.
// This is primarily intended for testing.
func NewClientWithHTTP(baseURL, token string, httpClient *http.Client) *Client {
	cfg := httpclient.GitHubConfig(baseURL, token, 0)
	return &Client{
		baseURL: cfg.BaseURL,
		http:    httpclient.NewWithHTTPClient(cfg, httpClient),
	}
}

// FetchAlerts retrieves all open code scanning alerts for a repository.
// It paginates through results since the GitHub API caps per_page at 100.
// If toolName is non-empty, only alerts from that tool are returned.
func (c *Client) FetchAlerts(ctx context.Context, owner, repo, toolName string) (AlertsResponse, error) {
	const perPage = 100
	var allAlerts AlertsResponse

	for page := 1; ; page++ {
		params := url.Values{
			"state":    {"open"},
			"per_page": {strconv.Itoa(perPage)},
			"page":     {strconv.Itoa(page)},
		}
		if toolName != "" {
			params.Set("tool_name", toolName)
		}

		endpoint := fmt.Sprintf("%s/repos/%s/%s/code-scanning/alerts?%s",
			c.baseURL, url.PathEscape(owner), url.PathEscape(repo), params.Encode())

		var pageAlerts AlertsResponse
		if err := c.http.Get(ctx, endpoint, &pageAlerts); err != nil {
			return nil, err
		}

		allAlerts = append(allAlerts, pageAlerts...)

		if len(pageAlerts) < perPage {
			break
		}
	}

	return allAlerts, nil
}

// AlertsURL returns the web URL for code scanning alerts in a repository.
func (c *Client) AlertsURL(owner, repo string) string {
	// GitHub web URL is always github.com regardless of API URL (for GHES it would differ)
	host := "github.com"
	if c.baseURL != httpclient.GitHubDefaultURL {
		// Extract host from API URL for GitHub Enterprise
		if u, err := url.Parse(c.baseURL); err == nil {
			host = u.Host
			// Remove 'api.' prefix if present for GHES
			host = strings.TrimPrefix(host, "api.")
		}
	}
	return fmt.Sprintf("https://%s/%s/%s/security/code-scanning", host, owner, repo)
}

// countAlertsBySeverity counts alerts grouped by security severity level.
// CodeQL alerts have a security_severity_level field (critical, high, medium, low).
// If that's not set, we fall back to the rule severity (error->high, warning->medium, note->low).
func countAlertsBySeverity(alerts AlertsResponse) AlertCounts {
	var scoringCounts scoring.SeverityCounts
	for _, alert := range alerts {
		// Prefer security_severity_level if available
		severity := strings.ToLower(alert.Rule.SecuritySeverityLevel)
		if severity == "" {
			// Fall back to rule severity
			switch strings.ToLower(alert.Rule.Severity) {
			case "error":
				severity = "high"
			case "warning":
				severity = "medium"
			default:
				severity = "low"
			}
		}
		scoring.CountSeverity(&scoringCounts, severity, "codeql", true)
	}
	return AlertCounts{
		Critical: scoringCounts.Critical,
		High:     scoringCounts.High,
		Medium:   scoringCounts.Medium,
		Low:      scoringCounts.Low,
	}
}
