package sonarqube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is an HTTP client for the SonarQube API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new SonarQube API client.
func NewClient(baseURL, token string, timeout time.Duration) *Client {
	// Ensure baseURL doesn't have trailing slash
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
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
	if err := c.doRequest(ctx, endpoint, &result); err != nil {
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
	if err := c.doRequest(ctx, endpoint, &result); err != nil {
		return nil, fmt.Errorf("fetching quality gate: %w", err)
	}

	return &result, nil
}

// doRequest performs an HTTP GET request with authentication.
func (c *Client) doRequest(ctx context.Context, endpoint string, result interface{}) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	// SonarQube uses token as username with empty password
	if c.token != "" {
		req.SetBasicAuth(c.token, "")
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing response body: %w", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
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
