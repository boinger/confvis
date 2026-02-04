package httpclient

import (
	"time"
)

// GitHubDefaultURL is the default GitHub API endpoint.
const GitHubDefaultURL = "https://api.github.com"

// GitHubConfig returns httpclient.Config pre-configured for GitHub API.
func GitHubConfig(baseURL, token string, timeout time.Duration) Config {
	baseURL = NormalizeBaseURL(baseURL, GitHubDefaultURL)
	return Config{
		BaseURL:  baseURL,
		Token:    token,
		AuthType: AuthBearer,
		Accept:   "application/vnd.github+json",
		Timeout:  timeout,
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
