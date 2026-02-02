package ghactions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.github.com"

// Client is an HTTP client for the GitHub Actions API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new GitHub Actions API client.
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

// FetchRunsOptions configures the workflow runs query.
type FetchRunsOptions struct {
	Workflow string // Filter by workflow file name or ID
	Event    string // Filter by trigger event (push, pull_request, etc.)
	Count    int    // Number of runs to fetch
}

// FetchRuns retrieves workflow runs for a repository.
// ownerRepo is in the format "owner/repo".
func (c *Client) FetchRuns(ctx context.Context, ownerRepo string, opts FetchRunsOptions) (*WorkflowRunsResponse, error) {
	parts := strings.SplitN(ownerRepo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("project must be in 'owner/repo' format, got %q", ownerRepo)
	}
	owner, repo := parts[0], parts[1]

	params := url.Values{
		"status":   {"completed"}, // Only completed runs
		"per_page": {strconv.Itoa(opts.Count)},
	}
	if opts.Event != "" {
		params.Set("event", opts.Event)
	}

	// Build endpoint - different path if filtering by workflow
	var endpoint string
	if opts.Workflow != "" {
		endpoint = fmt.Sprintf("%s/repos/%s/%s/actions/workflows/%s/runs?%s",
			c.baseURL, owner, repo, url.PathEscape(opts.Workflow), params.Encode())
	} else {
		endpoint = fmt.Sprintf("%s/repos/%s/%s/actions/runs?%s",
			c.baseURL, owner, repo, params.Encode())
	}

	var result WorkflowRunsResponse
	if err := c.doRequest(ctx, endpoint, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// doRequest performs an HTTP GET request with GitHub API headers.
func (c *Client) doRequest(ctx context.Context, endpoint string, result interface{}) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

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

// ActionsURL returns the web URL for a repository's Actions page.
func (c *Client) ActionsURL(ownerRepo string) string {
	parts := strings.SplitN(ownerRepo, "/", 2)
	if len(parts) != 2 {
		return ""
	}
	owner, repo := parts[0], parts[1]

	// Determine web URL from API URL
	webURL := "https://github.com"
	if c.baseURL != defaultBaseURL {
		// Enterprise: api.github.example.com -> github.example.com
		webURL = strings.Replace(c.baseURL, "api.", "", 1)
		webURL = strings.TrimSuffix(webURL, "/api/v3")
	}

	return fmt.Sprintf("%s/%s/%s/actions", webURL, owner, repo)
}
