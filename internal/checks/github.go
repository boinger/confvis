// Package checks provides integrations for creating check runs on CI platforms.
package checks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/httputil"
	"github.com/boinger/confvis/internal/sources/httpclient"
)

const (
	defaultGitHubAPIURL = "https://api.github.com"
	defaultTimeout      = 30 * time.Second

	// HTTP headers.
	headerAuthorization    = "Authorization"
	headerAccept           = "Accept"
	headerContentType      = "Content-Type"
	headerGitHubAPIVersion = "X-GitHub-Api-Version"

	// Header values.
	bearerPrefix     = "Bearer "
	acceptGitHubJSON = "application/vnd.github+json"
	contentTypeJSON  = "application/json"
	gitHubAPIVersion = "2022-11-28"

	// Pagination.
	commentsPerPage  = 100
	maxCommentPages  = 20

	// Retry configuration for GET requests.
	maxRetries        = 3
	initialBackoff    = 1 * time.Second
	maxJitterFraction = 0.5

	// Endpoint format strings.
	issueCommentsEndpoint = "%s/repos/%s/%s/issues/%d/comments"

	// Error format strings.
	errMarshalingRequest = "marshaling request: %w"
	errCreatingRequest   = "creating request: %w"
	errMakingRequest     = "making request: %w"
	errReadingResponse   = "reading response: %w"
	errDecodingResponse  = "decoding response: %w"
	errAPIStatus         = "API returned status %d: %s"
)

// Sentinel errors for validation.
var (
	errOwnerRepoRequired = errors.New("owner and repo are required")
	errPRRequired        = errors.New("PR number is required")
	errNoPRInEvent       = errors.New("no PR number found in event")
)

// retryableStatusCodes are HTTP status codes that indicate a transient failure.
var retryableStatusCodes = map[int]bool{
	http.StatusTooManyRequests:    true, // 429
	http.StatusBadGateway:         true, // 502
	http.StatusServiceUnavailable: true, // 503
	http.StatusGatewayTimeout:     true, // 504
}

// GitHubClient is an HTTP client for the GitHub Checks API.
type GitHubClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// GitHubClientConfig holds configuration for creating a GitHub client.
type GitHubClientConfig struct {
	BaseURL string
	Token   string
	Timeout time.Duration
}

// NewGitHubClient creates a new GitHub Checks API client.
func NewGitHubClient(cfg GitHubClientConfig) *GitHubClient {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	return &GitHubClient{
		baseURL:    httpclient.NormalizeBaseURL(cfg.BaseURL, defaultGitHubAPIURL),
		token:      cfg.Token,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// NewGitHubClientWithHTTP creates a new client with a custom HTTP client.
// This is primarily intended for testing.
func NewGitHubClientWithHTTP(cfg GitHubClientConfig, httpClient *http.Client) *GitHubClient {
	return &GitHubClient{
		baseURL:    httpclient.NormalizeBaseURL(cfg.BaseURL, defaultGitHubAPIURL),
		token:      cfg.Token,
		httpClient: httpClient,
	}
}

// CreateCheckOptions configures the check run creation.
type CreateCheckOptions struct {
	Owner   string // Repository owner
	Repo    string // Repository name
	SHA     string // Commit SHA
	Name    string // Check name
	BaseURL string // Optional: Override base URL for compare baseline
}

// CheckRunOutput represents the output section of a check run.
type CheckRunOutput struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Text    string `json:"text,omitempty"`
}

// CheckRunRequest is the request body for creating a check run.
type CheckRunRequest struct {
	Name       string          `json:"name"`
	HeadSHA    string          `json:"head_sha"`
	Status     string          `json:"status"`
	Conclusion string          `json:"conclusion,omitempty"`
	Output     *CheckRunOutput `json:"output,omitempty"`
}

// CheckRunResponse is the response from creating a check run.
type CheckRunResponse struct {
	ID      int64  `json:"id"`
	HTMLURL string `json:"html_url"`
	Status  string `json:"status"`
}

// CreateCheck creates a check run for the given confidence report.
func (c *GitHubClient) CreateCheck(ctx context.Context, report *confidence.Report, opts CreateCheckOptions) (*CheckRunResponse, error) {
	if opts.Owner == "" || opts.Repo == "" {
		return nil, errOwnerRepoRequired
	}
	if opts.SHA == "" {
		return nil, fmt.Errorf("SHA is required")
	}
	if opts.Name == "" {
		opts.Name = "Confidence Score"
	}

	// Determine conclusion based on threshold
	conclusion := "success"
	if !report.Passed() {
		conclusion = "failure"
	}

	// Build output
	output := buildCheckOutput(report)

	req := CheckRunRequest{
		Name:       opts.Name,
		HeadSHA:    opts.SHA,
		Status:     "completed",
		Conclusion: conclusion,
		Output:     output,
	}

	endpoint := fmt.Sprintf("%s/repos/%s/%s/check-runs", c.baseURL, opts.Owner, opts.Repo)
	return c.postCheckRun(ctx, endpoint, req)
}

// doRequest executes an HTTP request with standard GitHub API headers.
// It returns the response body bytes on success, or an error if the request
// fails or the status code doesn't match expectedStatus.
//
// GET requests are automatically retried on transient failures (429, 502, 503,
// 504, and network errors) with exponential backoff. Non-GET requests (POST,
// PATCH, DELETE) are NOT retried because they are not idempotent — retrying a
// POST that the server already processed would create duplicate check runs or
// comments.
func (c *GitHubClient) doRequest(ctx context.Context, method, endpoint string, body io.Reader, expectedStatus int) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf(errCreatingRequest, err)
	}

	httpReq.Header.Set(headerAuthorization, bearerPrefix+c.token)
	httpReq.Header.Set(headerAccept, acceptGitHubJSON)
	if body != nil {
		httpReq.Header.Set(headerContentType, contentTypeJSON)
	}
	httpReq.Header.Set(headerGitHubAPIVersion, gitHubAPIVersion)

	// Non-GET requests: single attempt (not idempotent).
	if method != http.MethodGet {
		return c.doHTTP(httpReq, expectedStatus)
	}

	// GET requests: retry on transient failures. The request has a nil body,
	// so it is safe to reuse across attempts without re-reading.
	var lastErr error
	for attempt := range maxRetries + 1 {
		if attempt > 0 {
			delay := c.retryDelay(attempt, lastErr)
			if err := httputil.SleepWithContext(ctx, delay); err != nil {
				return nil, lastErr
			}
		}

		respBody, err := c.doHTTP(httpReq, expectedStatus)
		if err == nil {
			return respBody, nil
		}
		lastErr = err

		if !isRetryableError(err) {
			return nil, err
		}
	}

	return nil, lastErr
}

// doHTTP performs a single HTTP request attempt and returns the response body.
func (c *GitHubClient) doHTTP(httpReq *http.Request, expectedStatus int) ([]byte, error) {
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// Network errors (no response) are retryable.
		return nil, &retryableHTTPError{err: fmt.Errorf(errMakingRequest, err)}
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(errReadingResponse, err)
	}

	if resp.StatusCode != expectedStatus {
		apiErr := fmt.Errorf(errAPIStatus, resp.StatusCode, string(respBody))
		if retryableStatusCodes[resp.StatusCode] {
			return nil, &retryableHTTPError{
				err:        apiErr,
				retryAfter: resp.Header.Get("Retry-After"),
			}
		}
		return nil, apiErr
	}

	return respBody, nil
}

// retryableHTTPError wraps an error to indicate it can be retried.
type retryableHTTPError struct {
	err        error
	retryAfter string // Retry-After header value, if present
}

func (e *retryableHTTPError) Error() string { return e.err.Error() }
func (e *retryableHTTPError) Unwrap() error { return e.err }

// isRetryableError reports whether an error should trigger a retry.
func isRetryableError(err error) bool {
	var re *retryableHTTPError
	return errors.As(err, &re)
}

// retryDelay calculates the delay before a retry attempt. If the previous
// error carried a Retry-After value, that takes precedence over exponential
// backoff.
func (c *GitHubClient) retryDelay(attempt int, lastErr error) time.Duration {
	var re *retryableHTTPError
	if errors.As(lastErr, &re) && re.retryAfter != "" {
		if d := httputil.ParseRetryAfter(re.retryAfter); d > 0 {
			return d
		}
	}

	// Exponential backoff with jitter.
	backoff := float64(initialBackoff) * math.Pow(2, float64(attempt-1))
	jitter := backoff * maxJitterFraction * rand.Float64() //#nosec G404 -- jitter for retry backoff, not security-sensitive
	return time.Duration(backoff + jitter)
}

// marshalAndDo marshals a request body and sends it via doRequest.
func (c *GitHubClient) marshalAndDo(ctx context.Context, method, endpoint string, reqBody any, expectedStatus int) ([]byte, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf(errMarshalingRequest, err)
	}
	return c.doRequest(ctx, method, endpoint, bytes.NewReader(jsonBody), expectedStatus)
}

func (c *GitHubClient) postCheckRun(ctx context.Context, endpoint string, req CheckRunRequest) (*CheckRunResponse, error) {
	respBody, err := c.marshalAndDo(ctx, http.MethodPost, endpoint, req, http.StatusCreated)
	if err != nil {
		return nil, err
	}

	var result CheckRunResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf(errDecodingResponse, err)
	}

	return &result, nil
}

func buildCheckOutput(report *confidence.Report) *CheckRunOutput {
	status := "Passed"
	if !report.Passed() {
		status = "Failed"
	}

	title := fmt.Sprintf("%s: %d%% (%s)", report.Title, report.ScoreValue(), status)

	// Build summary with score and threshold
	var summary strings.Builder
	fmt.Fprintf(&summary, "**Score:** %d%% | **Threshold:** %d%%\n\n", report.ScoreValue(), report.Threshold)

	if report.Description != "" {
		summary.WriteString(report.Description)
		summary.WriteString("\n\n")
	}

	// Add factors table if present
	if len(report.Factors) > 0 {
		summary.WriteString("### Factor Breakdown\n\n")
		summary.WriteString("| Factor | Score | Weight |\n")
		summary.WriteString("|--------|------:|-------:|\n")
		for _, f := range report.Factors {
			fmt.Fprintf(&summary, "| %s | %d%% | %d%% |\n", f.Name, f.Score, f.Weight)
		}
	}

	return &CheckRunOutput{
		Title:   title,
		Summary: summary.String(),
	}
}

// GitHubEnv holds environment variables from GitHub Actions.
type GitHubEnv struct {
	Token      string
	Repository string // "owner/repo" format
	SHA        string
	APIURL     string
}

// LoadGitHubEnv loads GitHub Actions environment variables.
// Returns an error only if GITHUB_TOKEN is missing.
func LoadGitHubEnv() (*GitHubEnv, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable is required")
	}

	return &GitHubEnv{
		Token:      token,
		Repository: os.Getenv("GITHUB_REPOSITORY"),
		SHA:        os.Getenv("GITHUB_SHA"),
		APIURL:     os.Getenv("GITHUB_API_URL"),
	}, nil
}

// CommentMarker is the hidden HTML comment used to identify confvis comments.
// Known limitation: if a user edits a confvis PR comment and removes this
// marker, confvis will create a duplicate comment on the next run rather than
// updating the existing one. This is by design — reliable identification
// without external state requires the marker to be present in the comment body.
const CommentMarker = "<!-- confvis-comment -->"

// CommentOptions configures comment posting.
type CommentOptions struct {
	Owner string // Repository owner
	Repo  string // Repository name
	PR    int    // Pull request number
}

// CommentResponse is the response from creating/updating a comment.
type CommentResponse struct {
	ID      int64  `json:"id"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

// issueCommentsResponse is used to parse the list comments API response.
type issueCommentsResponse []struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

// FindComment finds an existing confvis comment on a PR.
// Returns nil if no confvis comment is found.
func (c *GitHubClient) FindComment(ctx context.Context, opts CommentOptions) (*CommentResponse, error) {
	comments, err := c.FindAllConfvisComments(ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(comments) == 0 {
		return nil, nil //nolint:nilnil // nil response with no error means "not found"
	}
	return &comments[0], nil
}

// PostComment creates a new comment on a PR.
func (c *GitHubClient) PostComment(ctx context.Context, opts CommentOptions, body string) (*CommentResponse, error) {
	if opts.Owner == "" || opts.Repo == "" {
		return nil, errOwnerRepoRequired
	}
	if opts.PR <= 0 {
		return nil, errPRRequired
	}

	endpoint := fmt.Sprintf(issueCommentsEndpoint, c.baseURL, opts.Owner, opts.Repo, opts.PR)

	respBody, err := c.marshalAndDo(ctx, http.MethodPost, endpoint, struct {
		Body string `json:"body"`
	}{Body: body}, http.StatusCreated)
	if err != nil {
		return nil, err
	}

	var result CommentResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf(errDecodingResponse, err)
	}

	return &result, nil
}

// UpdateComment updates an existing comment.
func (c *GitHubClient) UpdateComment(ctx context.Context, opts CommentOptions, commentID int64, body string) (*CommentResponse, error) {
	if opts.Owner == "" || opts.Repo == "" {
		return nil, errOwnerRepoRequired
	}

	endpoint := fmt.Sprintf("%s/repos/%s/%s/issues/comments/%d", c.baseURL, opts.Owner, opts.Repo, commentID)

	respBody, err := c.marshalAndDo(ctx, http.MethodPatch, endpoint, struct {
		Body string `json:"body"`
	}{Body: body}, http.StatusOK)
	if err != nil {
		return nil, err
	}

	var result CommentResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf(errDecodingResponse, err)
	}

	return &result, nil
}

// DeleteComment deletes a comment.
func (c *GitHubClient) DeleteComment(ctx context.Context, opts CommentOptions, commentID int64) error {
	if opts.Owner == "" || opts.Repo == "" {
		return errOwnerRepoRequired
	}

	endpoint := fmt.Sprintf("%s/repos/%s/%s/issues/comments/%d", c.baseURL, opts.Owner, opts.Repo, commentID)

	_, err := c.doRequest(ctx, http.MethodDelete, endpoint, nil, http.StatusNoContent)
	return err
}

// FindAllConfvisComments finds all confvis comments on a PR.
// Returns empty slice if no confvis comments are found.
func (c *GitHubClient) FindAllConfvisComments(ctx context.Context, opts CommentOptions) ([]CommentResponse, error) {
	if opts.Owner == "" || opts.Repo == "" {
		return nil, errOwnerRepoRequired
	}
	if opts.PR <= 0 {
		return nil, errPRRequired
	}

	baseEndpoint := fmt.Sprintf(issueCommentsEndpoint, c.baseURL, opts.Owner, opts.Repo, opts.PR)

	var result []CommentResponse
	for page := 1; page <= maxCommentPages; page++ {
		endpoint := fmt.Sprintf("%s?per_page=%d&page=%d", baseEndpoint, commentsPerPage, page)

		respBody, err := c.doRequest(ctx, http.MethodGet, endpoint, nil, http.StatusOK)
		if err != nil {
			return nil, err
		}

		var comments issueCommentsResponse
		if err := json.Unmarshal(respBody, &comments); err != nil {
			return nil, fmt.Errorf(errDecodingResponse, err)
		}

		for _, comment := range comments {
			if strings.Contains(comment.Body, CommentMarker) {
				result = append(result, CommentResponse{
					ID:   comment.ID,
					Body: comment.Body,
				})
			}
		}

		if len(comments) < commentsPerPage {
			break
		}

		if page == maxCommentPages {
			_, _ = fmt.Fprintf(os.Stderr, "Warning: comment pagination capped at %d pages (%d comments); results may be incomplete\n",
				maxCommentPages, maxCommentPages*commentsPerPage)
		}
	}

	return result, nil
}

// LoadGitHubEnvWithPR loads GitHub Actions environment variables including PR number.
// Returns nil for env if not in a GitHub Actions environment.
func LoadGitHubEnvWithPR() (*GitHubEnv, int, error) {
	env, err := LoadGitHubEnv()
	if err != nil {
		return nil, 0, err
	}

	// Try to get PR number from GITHUB_EVENT_PATH
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return env, 0, nil
	}

	prNumber, parseErr := parsePRNumberFromEvent(eventPath)
	if parseErr != nil {
		if errors.Is(parseErr, errNoPRInEvent) {
			return env, 0, nil
		}
		_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to parse PR from event file %s: %v\n", eventPath, parseErr)
		return env, 0, nil
	}

	return env, prNumber, nil
}

// parsePRNumberFromEvent extracts the PR number from the GitHub event JSON file.
func parsePRNumberFromEvent(eventPath string) (int, error) {
	data, err := os.ReadFile(eventPath) //#nosec G304 -- path from GITHUB_EVENT_PATH env var set by GitHub Actions
	if err != nil {
		return 0, err
	}

	var event struct {
		PullRequest *struct {
			Number int `json:"number"`
		} `json:"pull_request"`
		Issue *struct {
			Number int `json:"number"`
		} `json:"issue"`
		Number int `json:"number"` // For issue_comment events
	}

	if err := json.Unmarshal(data, &event); err != nil {
		return 0, err
	}

	if event.PullRequest != nil && event.PullRequest.Number > 0 {
		return event.PullRequest.Number, nil
	}
	if event.Issue != nil && event.Issue.Number > 0 {
		return event.Issue.Number, nil
	}
	if event.Number > 0 {
		return event.Number, nil
	}

	return 0, errNoPRInEvent
}
