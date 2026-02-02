package snyk

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

const defaultBaseURL = "https://api.snyk.io"

// API version to use for Snyk REST API.
const apiVersion = "2024-10-15"

// Client is an HTTP client for the Snyk API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new Snyk API client.
func NewClient(baseURL, token string, timeout time.Duration) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// FetchProject retrieves a project with its issue counts.
func (c *Client) FetchProject(ctx context.Context, orgID, projectID string) (*ProjectResponse, error) {
	params := url.Values{
		"version": {apiVersion},
		"meta":    {"latest_issue_counts"},
	}

	endpoint := fmt.Sprintf("%s/rest/orgs/%s/projects/%s?%s",
		c.baseURL, url.PathEscape(orgID), url.PathEscape(projectID), params.Encode())

	var result ProjectResponse
	if err := c.doRequest(ctx, endpoint, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// doRequest performs an HTTP GET request with Snyk token authentication.
func (c *Client) doRequest(ctx context.Context, endpoint string, result interface{}) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.api+json")

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

// ProjectURL returns the web URL for a project in Snyk.
func (c *Client) ProjectURL(orgID, projectID string) string {
	// Snyk app URL is always app.snyk.io regardless of API URL
	return fmt.Sprintf("https://app.snyk.io/org/%s/project/%s", orgID, projectID)
}
