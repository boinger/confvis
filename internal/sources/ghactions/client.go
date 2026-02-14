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

// Compile-time interface compliance check.
var _ Fetcher = (*Client)(nil)

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
// It paginates through results since the GitHub API caps per_page at 100.
func (c *Client) FetchRuns(ctx context.Context, ownerRepo string, opts FetchRunsOptions) (*WorkflowRunsResponse, error) {
	owner, repo, err := repoparse.Parse(ownerRepo)
	if err != nil {
		return nil, err
	}

	perPage := opts.Count
	if perPage > 100 {
		perPage = 100
	}

	var allRuns []WorkflowRun
	var totalCount int

	for page := 1; len(allRuns) < opts.Count; page++ {
		params := url.Values{
			"status":   {"completed"},
			"per_page": {strconv.Itoa(perPage)},
			"page":     {strconv.Itoa(page)},
		}
		if opts.Event != "" {
			params.Set("event", opts.Event)
		}

		var endpoint string
		if opts.Workflow != "" {
			endpoint = fmt.Sprintf("%s/repos/%s/%s/actions/workflows/%s/runs?%s",
				c.baseURL, owner, repo, url.PathEscape(opts.Workflow), params.Encode())
		} else {
			endpoint = fmt.Sprintf("%s/repos/%s/%s/actions/runs?%s",
				c.baseURL, owner, repo, params.Encode())
		}

		result, err := httpclient.Get[WorkflowRunsResponse](c.http, ctx, endpoint)
		if err != nil {
			return nil, err
		}

		totalCount = result.TotalCount
		allRuns = append(allRuns, result.WorkflowRuns...)

		if len(result.WorkflowRuns) < perPage {
			break
		}
	}

	// Trim to requested count
	if len(allRuns) > opts.Count {
		allRuns = allRuns[:opts.Count]
	}

	return &WorkflowRunsResponse{
		TotalCount:   totalCount,
		WorkflowRuns: allRuns,
	}, nil
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
