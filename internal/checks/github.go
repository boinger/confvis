// Package checks provides integrations for creating check runs on CI platforms.
package checks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/boinger/confvis/internal/confidence"
)

const (
	defaultGitHubAPIURL = "https://api.github.com"
	defaultTimeout      = 30 * time.Second
)

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
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultGitHubAPIURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	return &GitHubClient{
		baseURL:    baseURL,
		token:      cfg.Token,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// NewGitHubClientWithHTTP creates a new client with a custom HTTP client.
// This is primarily intended for testing.
func NewGitHubClientWithHTTP(cfg GitHubClientConfig, httpClient *http.Client) *GitHubClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultGitHubAPIURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &GitHubClient{
		baseURL:    baseURL,
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
		return nil, fmt.Errorf("owner and repo are required")
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

func (c *GitHubClient) postCheckRun(ctx context.Context, endpoint string, req CheckRunRequest) (*CheckRunResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.token)
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result CheckRunResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &result, nil
}

func buildCheckOutput(report *confidence.Report) *CheckRunOutput {
	status := "Passed"
	if !report.Passed() {
		status = "Failed"
	}

	title := fmt.Sprintf("%s: %d%% (%s)", report.Title, report.Score, status)

	// Build summary with score and threshold
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("**Score:** %d%% | **Threshold:** %d%%\n\n", report.Score, report.Threshold))

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
			summary.WriteString(fmt.Sprintf("| %s | %d%% | %d%% |\n", f.Name, f.Score, f.Weight))
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

// ParseRepository splits "owner/repo" into owner and repo.
func ParseRepository(ownerRepo string) (owner, repo string, err error) {
	parts := strings.SplitN(ownerRepo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repository must be in 'owner/repo' format, got %q", ownerRepo)
	}
	return parts[0], parts[1], nil
}
