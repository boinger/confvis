package coveralls

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/boinger/confvis/internal/sources/httpclient"
	"github.com/boinger/confvis/internal/sources/repoparse"
)

const defaultBaseURL = "https://coveralls.io"

// Client is an HTTP client for the Coveralls API.
type Client struct {
	baseURL string
	http    *httpclient.Client
}

// NewClient creates a new Coveralls API client.
// Token is optional for public repos.
func NewClient(token string, timeout time.Duration) *Client {
	authType := httpclient.AuthNone
	if token != "" {
		authType = httpclient.AuthBearer
	}

	return &Client{
		baseURL: defaultBaseURL,
		http: httpclient.New(httpclient.Config{
			BaseURL:  defaultBaseURL,
			Token:    token,
			AuthType: authType,
			Accept:   "application/json",
			Timeout:  timeout,
		}),
	}
}

// NewClientWithHTTP creates a new client with a custom base URL and HTTP client.
// This is primarily intended for testing.
func NewClientWithHTTP(baseURL, token string, httpClient *http.Client) *Client {
	authType := httpclient.AuthNone
	if token != "" {
		authType = httpclient.AuthBearer
	}

	return &Client{
		baseURL: baseURL,
		http: httpclient.NewWithHTTPClient(httpclient.Config{
			BaseURL:  baseURL,
			Token:    token,
			AuthType: authType,
			Accept:   "application/json",
		}, httpClient),
	}
}

// FetchReport retrieves the coverage report for a repository.
// service is the git provider (github, gitlab, bitbucket).
// ownerRepo is in the format "owner/repo".
func (c *Client) FetchReport(ctx context.Context, service, ownerRepo string) (*ReportResponse, error) {
	owner, repo, err := repoparse.Parse(ownerRepo)
	if err != nil {
		return nil, err
	}

	// Coveralls API: GET /github/{owner}/{repo}.json
	endpoint := fmt.Sprintf("%s/%s/%s/%s.json", c.baseURL, service, owner, repo)

	var result ReportResponse
	if err := c.http.Get(ctx, endpoint, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ReportURL returns the web URL for a repository's coverage report.
func (c *Client) ReportURL(service, ownerRepo string) string {
	owner, repo := repoparse.ParseDefault(ownerRepo)
	if owner == "" || repo == "" {
		return ""
	}
	return fmt.Sprintf("https://coveralls.io/%s/%s/%s", service, owner, repo)
}
