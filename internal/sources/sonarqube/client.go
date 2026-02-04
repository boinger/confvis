package sonarqube

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/boinger/confvis/internal/sources/httpclient"
)

// Client is an HTTP client for the SonarQube API.
type Client struct {
	baseURL string
	http    *httpclient.Client
}

// NewClient creates a new SonarQube API client.
func NewClient(baseURL, token string, timeout time.Duration) *Client {
	// Ensure baseURL doesn't have trailing slash
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		baseURL: baseURL,
		http: httpclient.New(httpclient.Config{
			BaseURL:  baseURL,
			Token:    token,
			AuthType: httpclient.AuthBasic,
			Accept:   "application/json",
			Timeout:  timeout,
		}),
	}
}

// NewClientWithHTTP creates a new client with a custom HTTP client.
// This is primarily intended for testing.
func NewClientWithHTTP(baseURL, token string, httpClient *http.Client) *Client {
	// Ensure baseURL doesn't have trailing slash
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		baseURL: baseURL,
		http: httpclient.NewWithHTTPClient(httpclient.Config{
			BaseURL:  baseURL,
			Token:    token,
			AuthType: httpclient.AuthBasic,
			Accept:   "application/json",
		}, httpClient),
	}
}

// FetchMeasures retrieves metrics for a project component.
func (c *Client) FetchMeasures(ctx context.Context, project, branch string) (*MeasuresResponse, error) {
	params := url.Values{
		"component":  {project},
		"metricKeys": {strings.Join(AllMetrics, ",")},
	}
	if branch != "" {
		params.Set("branch", branch)
	}

	endpoint := fmt.Sprintf("%s/api/measures/component?%s", c.baseURL, params.Encode())

	var result MeasuresResponse
	if err := c.http.Get(ctx, endpoint, &result); err != nil {
		return nil, fmt.Errorf("fetching measures: %w", err)
	}

	return &result, nil
}

// FetchQualityGate retrieves the quality gate status for a project.
func (c *Client) FetchQualityGate(ctx context.Context, project, branch string) (*QualityGateResponse, error) {
	params := url.Values{
		"projectKey": {project},
	}
	if branch != "" {
		params.Set("branch", branch)
	}

	endpoint := fmt.Sprintf("%s/api/qualitygates/project_status?%s", c.baseURL, params.Encode())

	var result QualityGateResponse
	if err := c.http.Get(ctx, endpoint, &result); err != nil {
		return nil, fmt.Errorf("fetching quality gate: %w", err)
	}

	return &result, nil
}

// ProjectURL returns the web URL for a project in SonarQube.
func (c *Client) ProjectURL(project, branch string) string {
	u := fmt.Sprintf("%s/dashboard?id=%s", c.baseURL, url.QueryEscape(project))
	if branch != "" {
		u += "&branch=" + url.QueryEscape(branch)
	}
	return u
}

// MeasureURL returns the web URL for a specific measure.
func (c *Client) MeasureURL(project, metric, branch string) string {
	u := fmt.Sprintf("%s/component_measures?id=%s&metric=%s",
		c.baseURL, url.QueryEscape(project), url.QueryEscape(metric))
	if branch != "" {
		u += "&branch=" + url.QueryEscape(branch)
	}
	return u
}
