package dependabot

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/boinger/confvis/internal/sources/githubalerts"
	"github.com/boinger/confvis/internal/sources/scoring"
)

var alertsConfig = githubalerts.Config{
	EndpointPath: "dependabot/alerts",
	WebURLPath:   "security/dependabot",
}

// Client is an HTTP client for the GitHub Dependabot Alerts API.
type Client struct {
	*githubalerts.Client
}

// NewClient creates a new Dependabot API client.
func NewClient(baseURL, token string, timeout time.Duration) *Client {
	return &Client{
		Client: githubalerts.NewClient(baseURL, token, timeout, alertsConfig),
	}
}

// NewClientWithHTTP creates a new client with a custom HTTP client.
// This is primarily intended for testing.
func NewClientWithHTTP(baseURL, token string, httpClient *http.Client) *Client {
	return &Client{
		Client: githubalerts.NewClientWithHTTP(baseURL, token, httpClient, alertsConfig),
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

		endpoint := c.BuildEndpoint(owner, repo, params)

		var pageAlerts AlertsResponse
		if err := c.HTTP.Get(ctx, endpoint, &pageAlerts); err != nil {
			return nil, err
		}

		allAlerts = append(allAlerts, pageAlerts...)

		if len(pageAlerts) < perPage {
			break
		}
	}

	return allAlerts, nil
}

// countAlertsBySeverity counts alerts grouped by severity.
func countAlertsBySeverity(alerts AlertsResponse) AlertCounts {
	var scoringCounts scoring.SeverityCounts
	for _, alert := range alerts {
		scoring.CountSeverity(&scoringCounts, alert.SecurityAdvisory.Severity, "dependabot", true)
	}
	return AlertCounts{
		Critical: scoringCounts.Critical,
		High:     scoringCounts.High,
		Medium:   scoringCounts.Medium,
		Low:      scoringCounts.Low,
	}
}
