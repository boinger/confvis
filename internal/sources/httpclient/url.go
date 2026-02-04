package httpclient

import "strings"

// NormalizeBaseURL ensures URL has no trailing slash and applies a default if empty.
// This is used by clients that need consistent URL handling for API endpoints.
func NormalizeBaseURL(baseURL, defaultURL string) string {
	if baseURL == "" {
		baseURL = defaultURL
	}
	return strings.TrimRight(baseURL, "/")
}
