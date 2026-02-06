package codeql

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/boinger/confvis/internal/sources/githubalerts"
	"github.com/boinger/confvis/internal/sources/scoring"
)

var alertsConfig = githubalerts.Config{
	EndpointPath: "code-scanning/alerts",
	WebURLPath:   "security/code-scanning",
}

// Client is an HTTP client for the GitHub Code Scanning API.
type Client struct {
	*githubalerts.Client
}

// NewClient creates a new CodeQL API client.
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
