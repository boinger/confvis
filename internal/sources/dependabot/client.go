package dependabot

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/boinger/confvis/internal/sources/httpclient"
)

const defaultBaseURL = "https://api.github.com"

// Client is an HTTP client for the GitHub Dependabot Alerts API.
type Client struct {
	baseURL string
	http    *httpclient.Client
}

// NewClient creates a new Dependabot API client.
func NewClient(baseURL, token string, timeout time.Duration) *Client {
	baseURL = httpclient.NormalizeBaseURL(baseURL, defaultBaseURL)

	return &Client{
		baseURL: baseURL,
		http: httpclient.New(httpclient.Config{
			BaseURL:  baseURL,
			Token:    token,
			AuthType: httpclient.AuthBearer,
			Accept:   "application/vnd.github+json",
			Timeout:  timeout,
		}),
	}
}

// NewClientWithHTTP creates a new client with a custom HTTP client.
// This is primarily intended for testing.
func NewClientWithHTTP(baseURL, token string, httpClient *http.Client) *Client {
	baseURL = httpclient.NormalizeBaseURL(baseURL, defaultBaseURL)

	return &Client{
		baseURL: baseURL,
		http: httpclient.NewWithHTTPClient(httpclient.Config{
			BaseURL:  baseURL,
			Token:    token,
			AuthType: httpclient.AuthBearer,
			Accept:   "application/vnd.github+json",
		}, httpClient),
	}
}

// FetchAlerts retrieves all open Dependabot alerts for a repository.
func (c *Client) FetchAlerts(ctx context.Context, owner, repo string) (AlertsResponse, error) {
	params := url.Values{
		"state":    {"open"},
		"per_page": {"100"},
	}

	endpoint := fmt.Sprintf("%s/repos/%s/%s/dependabot/alerts?%s",
		c.baseURL, url.PathEscape(owner), url.PathEscape(repo), params.Encode())

	var result AlertsResponse
	if err := c.http.Get(ctx, endpoint, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// AlertsURL returns the web URL for Dependabot alerts in a repository.
func (c *Client) AlertsURL(owner, repo string) string {
	// GitHub web URL is always github.com regardless of API URL (for GHES it would differ)
	host := "github.com"
	if c.baseURL != defaultBaseURL {
		// Extract host from API URL for GitHub Enterprise
		if u, err := url.Parse(c.baseURL); err == nil {
			host = u.Host
			// Remove 'api.' prefix if present for GHES
			host = strings.TrimPrefix(host, "api.")
		}
	}
	return fmt.Sprintf("https://%s/%s/%s/security/dependabot", host, owner, repo)
}

// CountAlertsBySeverity counts alerts grouped by severity.
func CountAlertsBySeverity(alerts AlertsResponse) AlertCounts {
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
