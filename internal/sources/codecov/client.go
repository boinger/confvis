package codecov

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.codecov.io"

// Client is an HTTP client for the Codecov API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new Codecov API client.
func NewClient(token string, timeout time.Duration) *Client {
	return &Client{
		baseURL: defaultBaseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// FetchReport retrieves the coverage report for a repository.
// service is the git provider (github, gitlab, bitbucket).
// ownerRepo is in the format "owner/repo".
func (c *Client) FetchReport(ctx context.Context, service, ownerRepo string) (*ReportResponse, error) {
	parts := strings.SplitN(ownerRepo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("project must be in 'owner/repo' format, got %q", ownerRepo)
	}
	owner, repo := parts[0], parts[1]

	endpoint := fmt.Sprintf("%s/api/v2/%s/%s/repos/%s/report/", c.baseURL, service, owner, repo)

	var result ReportResponse
	if err := c.doRequest(ctx, endpoint, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// doRequest performs an HTTP GET request with Bearer authentication.
func (c *Client) doRequest(ctx context.Context, endpoint string, result interface{}) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
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

// ReportURL returns the web URL for a repository's coverage report.
func (c *Client) ReportURL(service, ownerRepo string) string {
	parts := strings.SplitN(ownerRepo, "/", 2)
	if len(parts) != 2 {
		return ""
	}
	owner, repo := parts[0], parts[1]
	return fmt.Sprintf("https://app.codecov.io/%s/%s/%s", service, owner, repo)
}
