// Package httpclient provides a common HTTP client for API sources.
package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
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

const (
	// DefaultMaxRetries is the default number of retry attempts for transient failures.
	DefaultMaxRetries = 3

	// DefaultInitialBackoff is the starting backoff duration before the first retry.
	DefaultInitialBackoff = 1 * time.Second

	// maxJitterFraction controls the random jitter added to backoff (0.0–0.5).
	maxJitterFraction = 0.5
)

// retryableStatusCodes are HTTP status codes that indicate a transient failure.
var retryableStatusCodes = map[int]bool{
	http.StatusTooManyRequests:    true, // 429
	http.StatusBadGateway:         true, // 502
	http.StatusServiceUnavailable: true, // 503
	http.StatusGatewayTimeout:     true, // 504
}

// ResponseHook is called after each successful HTTP response (status 200).
// It receives the response headers and can inspect rate-limit or other metadata.
type ResponseHook func(headers http.Header)

// Config holds configuration for the HTTP client.
type Config struct {
	BaseURL        string
	Token          string
	AuthType       AuthType
	Accept         string
	ExtraHeaders   map[string]string
	Timeout        time.Duration
	MaxRetries     int           // Max retry attempts (0 = use default; -1 = disable)
	InitialBackoff time.Duration // Starting backoff before first retry (0 = use default)
	OnResponse     ResponseHook  // Optional callback after successful responses
}

// Client is a configurable HTTP client for API requests.
type Client struct {
	baseURL        string
	token          string
	authType       AuthType
	accept         string
	extraHeaders   map[string]string
	httpClient     *http.Client
	maxRetries     int
	initialBackoff time.Duration
	onResponse     ResponseHook
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

	maxRetries := cfg.MaxRetries
	if maxRetries == 0 {
		maxRetries = DefaultMaxRetries
	} else if maxRetries < 0 {
		maxRetries = 0
	}

	initialBackoff := cfg.InitialBackoff
	if initialBackoff == 0 {
		initialBackoff = DefaultInitialBackoff
	}

	return &Client{
		baseURL:        cfg.BaseURL,
		token:          cfg.Token,
		authType:       cfg.AuthType,
		accept:         accept,
		extraHeaders:   cfg.ExtraHeaders,
		httpClient:     httpClient,
		maxRetries:     maxRetries,
		initialBackoff: initialBackoff,
		onResponse:     cfg.OnResponse,
	}
}

// Get performs an HTTP GET request and decodes the JSON response.
// Transient failures (429, 502, 503, 504, network errors) are retried
// with exponential backoff. The Retry-After header is respected when present.
func (c *Client) Get(ctx context.Context, endpoint string, result interface{}) error {
	var lastErr error

	for attempt := range c.maxRetries + 1 {
		if attempt > 0 {
			delay := c.retryDelay(attempt, lastErr)
			if err := sleepWithContext(ctx, delay); err != nil {
				return lastErr
			}
		}

		err := c.doGet(ctx, endpoint, result)
		if err == nil {
			return nil
		}
		lastErr = err

		if !isRetryable(err) {
			return err
		}
	}

	return lastErr
}

// doGet performs a single HTTP GET attempt.
func (c *Client) doGet(ctx context.Context, endpoint string, result interface{}) (err error) {
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
		return &retryableError{err: fmt.Errorf("making request: %w", err)}
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing response body: %w", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		apiErr := fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		if retryableStatusCodes[resp.StatusCode] {
			retryAfter := resp.Header.Get("Retry-After")
			return &retryableError{err: apiErr, retryAfter: retryAfter}
		}
		return apiErr
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	if c.onResponse != nil {
		c.onResponse(resp.Header)
	}

	return nil
}

// retryableError wraps an error to indicate it can be retried.
type retryableError struct {
	err        error
	retryAfter string // Retry-After header value, if present
}

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

// isRetryable reports whether an error should trigger a retry.
func isRetryable(err error) bool {
	var re *retryableError
	return errors.As(err, &re)
}

// retryDelay calculates the delay before a retry attempt. If the previous
// error carried a Retry-After value, that takes precedence over exponential
// backoff.
func (c *Client) retryDelay(attempt int, lastErr error) time.Duration {
	// Check if previous error had a Retry-After hint
	var re *retryableError
	if errors.As(lastErr, &re) && re.retryAfter != "" {
		if d := parseRetryAfter(re.retryAfter); d > 0 {
			return d
		}
	}

	// Exponential backoff with jitter
	backoff := float64(c.initialBackoff) * math.Pow(2, float64(attempt-1))
	jitter := backoff * maxJitterFraction * rand.Float64() //#nosec G404 -- jitter for retry backoff, not security-sensitive
	return time.Duration(backoff + jitter)
}

// parseRetryAfter parses a Retry-After header value (seconds or HTTP-date).
func parseRetryAfter(value string) time.Duration {
	// Try parsing as seconds first
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP-date
	if t, err := http.ParseTime(value); err == nil {
		if delay := time.Until(t); delay > 0 {
			return delay
		}
	}

	return 0
}

// sleepWithContext sleeps for the given duration, returning early if the
// context is canceled.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// BaseURL returns the configured base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}
