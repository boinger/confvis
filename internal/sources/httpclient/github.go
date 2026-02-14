package httpclient

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

// GitHubDefaultURL is the default GitHub API endpoint.
const GitHubDefaultURL = "https://api.github.com"

// rateLimitWarningThreshold is the number of remaining requests below which
// a warning is printed to stderr.
const rateLimitWarningThreshold = 100

// GitHubConfig returns httpclient.Config pre-configured for GitHub API.
func GitHubConfig(baseURL, token string, timeout time.Duration) Config {
	baseURL = NormalizeBaseURL(baseURL, GitHubDefaultURL)
	return Config{
		BaseURL:    baseURL,
		Token:      token,
		AuthType:   AuthBearer,
		Accept:     "application/vnd.github+json",
		Timeout:    timeout,
		OnResponse: GitHubRateLimitHook,
	}
}

// GitHubConfigWithVersion returns a Config including the API version header.
// Use this for APIs that require the X-GitHub-Api-Version header.
func GitHubConfigWithVersion(baseURL, token string, timeout time.Duration, version string) Config {
	cfg := GitHubConfig(baseURL, token, timeout)
	cfg.ExtraHeaders = map[string]string{
		"X-GitHub-Api-Version": version,
	}
	return cfg
}

// GitHubRateLimitHook inspects GitHub API rate-limit response headers and
// warns to stderr when the remaining request count is low.
func GitHubRateLimitHook(headers http.Header) {
	remaining := headers.Get("X-RateLimit-Remaining")
	if remaining == "" {
		return
	}

	n, err := strconv.Atoi(remaining)
	if err != nil {
		return
	}

	if n < rateLimitWarningThreshold {
		reset := headers.Get("X-RateLimit-Reset")
		if resetEpoch, err := strconv.ParseInt(reset, 10, 64); err == nil {
			resetTime := time.Unix(resetEpoch, 0)
			fmt.Fprintf(os.Stderr, "Warning: GitHub API rate limit low (%d requests remaining, resets at %s)\n",
				n, resetTime.Format(time.RFC3339))
		} else {
			fmt.Fprintf(os.Stderr, "Warning: GitHub API rate limit low (%d requests remaining)\n", n)
		}
	}
}
