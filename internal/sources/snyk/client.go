package snyk

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/boinger/confvis/internal/sources/httpclient"
)

const defaultBaseURL = "https://api.snyk.io"

// API version to use for Snyk REST API.
const apiVersion = "2024-10-15"

// Client is an HTTP client for the Snyk API.
type Client struct {
	baseURL string
	http    *httpclient.Client
}

// NewClient creates a new Snyk API client.
func NewClient(baseURL, token string, timeout time.Duration) *Client {
	baseURL = httpclient.NormalizeBaseURL(baseURL, defaultBaseURL)

	return &Client{
		baseURL: baseURL,
		http: httpclient.New(httpclient.Config{
			BaseURL:  baseURL,
			Token:    token,
			AuthType: httpclient.AuthToken,
			Accept:   "application/vnd.api+json",
			Timeout:  timeout,
		}),
	}
}

// NewClientWithHTTP creates a new client with a custom HTTP client.
// This is primarily intended for testing.
func NewClientWithHTTP(baseURL, token string, httpClient *http.Client) *Client {
	baseURL = httpclient.NormalizeBaseURL(baseURL, defaultBaseURL)

	return &Client{
		baseURL: baseURL,
		http: httpclient.NewWithHTTPClient(httpclient.Config{
			BaseURL:  baseURL,
			Token:    token,
			AuthType: httpclient.AuthToken,
			Accept:   "application/vnd.api+json",
		}, httpClient),
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
	if err := c.http.Get(ctx, endpoint, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ProjectURL returns the web URL for a project in Snyk.
func (c *Client) ProjectURL(orgID, projectID string) string {
	// Snyk app URL is always app.snyk.io regardless of API URL
	return fmt.Sprintf("https://app.snyk.io/org/%s/project/%s", orgID, projectID)
}
