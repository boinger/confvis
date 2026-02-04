package codecov

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/boinger/confvis/internal/sources/httpclient"
)

const defaultBaseURL = "https://api.codecov.io"

// Client is an HTTP client for the Codecov API.
type Client struct {
	baseURL string
	http    *httpclient.Client
}

// NewClient creates a new Codecov API client.
func NewClient(token string, timeout time.Duration) *Client {
	return &Client{
		baseURL: defaultBaseURL,
		http: httpclient.New(httpclient.Config{
			BaseURL:  defaultBaseURL,
			Token:    token,
			AuthType: httpclient.AuthBearer,
			Accept:   "application/json",
			Timeout:  timeout,
		}),
	}
}

// NewClientWithHTTP creates a new client with a custom base URL and HTTP client.
// This is primarily intended for testing.
func NewClientWithHTTP(baseURL, token string, httpClient *http.Client) *Client {
	return &Client{
		baseURL: baseURL,
		http: httpclient.NewWithHTTPClient(httpclient.Config{
			BaseURL:  baseURL,
			Token:    token,
			AuthType: httpclient.AuthBearer,
			Accept:   "application/json",
		}, httpClient),
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
	if err := c.http.Get(ctx, endpoint, &result); err != nil {
		return nil, err
	}

	return &result, nil
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
