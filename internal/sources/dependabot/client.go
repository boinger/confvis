package dependabot

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/boinger/confvis/internal/sources/httpclient"
)

// Client is an HTTP client for the GitHub Dependabot Alerts API.
type Client struct {
	baseURL string
	http    *httpclient.Client
}

// NewClient creates a new Dependabot API client.
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

// FetchAlerts retrieves all open Dependabot alerts for a repository.
// It paginates through results since the GitHub API caps per_page at 100.
func (c *Client) FetchAlerts(ctx context.Context, owner, repo string) (AlertsResponse, error) {
	const perPage = 100
	var allAlerts AlertsResponse

	for page := 1; ; page++ {
		params := url.Values{
			"state":    {"open"},
			"per_page": {strconv.Itoa(perPage)},
			"page":     {strconv.Itoa(page)},
		}

		endpoint := fmt.Sprintf("%s/repos/%s/%s/dependabot/alerts?%s",
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

// AlertsURL returns the web URL for Dependabot alerts in a repository.
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
	return fmt.Sprintf("https://%s/%s/%s/security/dependabot", host, owner, repo)
}

// countAlertsBySeverity counts alerts grouped by severity.
func countAlertsBySeverity(alerts AlertsResponse) AlertCounts {
	var counts AlertCounts
	for _, alert := range alerts {
		switch strings.ToLower(alert.SecurityAdvisory.Severity) {
		case "critical":
			counts.Critical++
		case "high":
			counts.High++
		case "medium":
			counts.Medium++
		case "low":
			counts.Low++
		}
	}
	return counts
}
