// Package httpclient provides a common HTTP client for API sources.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AuthType defines how authentication is performed.
type AuthType int

const (
	// AuthNone means no authentication header is set.
	AuthNone AuthType = iota
	// AuthBearer uses "Authorization: Bearer <token>".
	AuthBearer
	// AuthToken uses "Authorization: token <token>" (Snyk style).
	AuthToken
	// AuthBasic uses HTTP Basic auth with token as username and empty password.
	AuthBasic
)

// Config holds configuration for the HTTP client.
type Config struct {
	BaseURL      string
	Token        string
	AuthType     AuthType
	Accept       string
	ExtraHeaders map[string]string
	Timeout      time.Duration
}

// Client is a configurable HTTP client for API requests.
type Client struct {
	baseURL      string
	token        string
	authType     AuthType
	accept       string
	extraHeaders map[string]string
	httpClient   *http.Client
}

// New creates a new HTTP client with the given configuration.
func New(cfg Config) *Client {
	return NewWithHTTPClient(cfg, &http.Client{Timeout: cfg.Timeout})
}

// NewWithHTTPClient creates a new HTTP client with a custom http.Client.
// This is primarily intended for testing.
func NewWithHTTPClient(cfg Config, httpClient *http.Client) *Client {
	accept := cfg.Accept
	if accept == "" {
		accept = "application/json"
	}

	return &Client{
		baseURL:      cfg.BaseURL,
		token:        cfg.Token,
		authType:     cfg.AuthType,
		accept:       accept,
		extraHeaders: cfg.ExtraHeaders,
		httpClient:   httpClient,
	}
}

// Get performs an HTTP GET request and decodes the JSON response.
func (c *Client) Get(ctx context.Context, endpoint string, result interface{}) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	// Set auth header based on type
	if c.token != "" {
		switch c.authType {
		case AuthBearer:
			req.Header.Set("Authorization", "Bearer "+c.token)
		case AuthToken:
			req.Header.Set("Authorization", "token "+c.token)
		case AuthBasic:
			req.SetBasicAuth(c.token, "")
		case AuthNone:
			// No auth header
		}
	}

	req.Header.Set("Accept", c.accept)

	// Set any extra headers
	for k, v := range c.extraHeaders {
		req.Header.Set(k, v)
	}

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

// BaseURL returns the configured base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}
