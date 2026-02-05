package ghactions

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/boinger/confvis/internal/sources/httpclient"
	"github.com/boinger/confvis/internal/sources/repoparse"
)

const githubAPIVersion = "2022-11-28"

// Client is an HTTP client for the GitHub Actions API.
type Client struct {
	baseURL string
	http    *httpclient.Client
}

// NewClient creates a new GitHub Actions API client.
func NewClient(baseURL, token string, timeout time.Duration) *Client {
	cfg := httpclient.GitHubConfigWithVersion(baseURL, token, timeout, githubAPIVersion)
	return &Client{
		baseURL: cfg.BaseURL,
		http:    httpclient.New(cfg),
	}
}

// NewClientWithHTTP creates a new client with a custom HTTP client.
// This is primarily intended for testing.
func NewClientWithHTTP(baseURL, token string, httpClient *http.Client) *Client {
	cfg := httpclient.GitHubConfigWithVersion(baseURL, token, 0, githubAPIVersion)
	return &Client{
		baseURL: cfg.BaseURL,
		http:    httpclient.NewWithHTTPClient(cfg, httpClient),
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
	owner, repo, err := repoparse.Parse(ownerRepo)
	if err != nil {
		return nil, err
	}

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
	if err := c.http.Get(ctx, endpoint, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ActionsURL returns the web URL for a repository's Actions page.
func (c *Client) ActionsURL(ownerRepo string) string {
	owner, repo := repoparse.ParseDefault(ownerRepo)
	if owner == "" || repo == "" {
		return ""
	}

	// Determine web URL from API URL
	webURL := "https://github.com"
	if c.baseURL != httpclient.GitHubDefaultURL {
		// Enterprise: api.github.example.com -> github.example.com
		webURL = strings.Replace(c.baseURL, "api.", "", 1)
		webURL = strings.TrimSuffix(webURL, "/api/v3")
	}

	return fmt.Sprintf("%s/%s/%s/actions", webURL, owner, repo)
}
