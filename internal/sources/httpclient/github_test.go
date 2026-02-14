package httpclient

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestGitHubConfig(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		token       string
		timeout     time.Duration
		wantBaseURL string
		wantAccept  string
	}{
		{
			name:        "default URL",
			baseURL:     "",
			token:       "test-token",
			timeout:     30 * time.Second,
			wantBaseURL: GitHubDefaultURL,
			wantAccept:  "application/vnd.github+json",
		},
		{
			name:        "custom URL",
			baseURL:     "https://api.github.example.com/",
			token:       "test-token",
			timeout:     60 * time.Second,
			wantBaseURL: "https://api.github.example.com",
			wantAccept:  "application/vnd.github+json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := GitHubConfig(tt.baseURL, tt.token, tt.timeout)

			if cfg.BaseURL != tt.wantBaseURL {
				t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, tt.wantBaseURL)
			}
			if cfg.Token != tt.token {
				t.Errorf("Token = %q, want %q", cfg.Token, tt.token)
			}
			if cfg.AuthType != AuthBearer {
				t.Errorf("AuthType = %v, want AuthBearer", cfg.AuthType)
			}
			if cfg.Accept != tt.wantAccept {
				t.Errorf("Accept = %q, want %q", cfg.Accept, tt.wantAccept)
			}
			if cfg.Timeout != tt.timeout {
				t.Errorf("Timeout = %v, want %v", cfg.Timeout, tt.timeout)
			}
			if cfg.ExtraHeaders != nil {
				t.Errorf("ExtraHeaders = %v, want nil", cfg.ExtraHeaders)
			}
			if cfg.OnResponse == nil {
				t.Error("OnResponse should be set to GitHubRateLimitHook")
			}
		})
	}
}

func TestGitHubConfigWithVersion(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		token       string
		timeout     time.Duration
		version     string
		wantBaseURL string
	}{
		{
			name:        "default URL with version",
			baseURL:     "",
			token:       "test-token",
			timeout:     30 * time.Second,
			version:     "2022-11-28",
			wantBaseURL: GitHubDefaultURL,
		},
		{
			name:        "custom URL with version",
			baseURL:     "https://api.github.example.com",
			token:       "test-token",
			timeout:     60 * time.Second,
			version:     "2024-01-01",
			wantBaseURL: "https://api.github.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := GitHubConfigWithVersion(tt.baseURL, tt.token, tt.timeout, tt.version)

			if cfg.BaseURL != tt.wantBaseURL {
				t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, tt.wantBaseURL)
			}
			if cfg.Token != tt.token {
				t.Errorf("Token = %q, want %q", cfg.Token, tt.token)
			}
			if cfg.AuthType != AuthBearer {
				t.Errorf("AuthType = %v, want AuthBearer", cfg.AuthType)
			}
			if cfg.Accept != "application/vnd.github+json" {
				t.Errorf("Accept = %q, want %q", cfg.Accept, "application/vnd.github+json")
			}

			if cfg.ExtraHeaders == nil {
				t.Fatal("ExtraHeaders = nil, want map with version")
			}
			if gotVersion := cfg.ExtraHeaders["X-GitHub-Api-Version"]; gotVersion != tt.version {
				t.Errorf("X-GitHub-Api-Version = %q, want %q", gotVersion, tt.version)
			}
		})
	}
}

// captureStderr redirects stderr and captures output during fn execution.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = old }()

	fn()
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestGitHubRateLimitHook_NoHeaders(t *testing.T) {
	output := captureStderr(t, func() {
		GitHubRateLimitHook(http.Header{})
	})
	if output != "" {
		t.Errorf("expected no output for missing headers, got %q", output)
	}
}

func TestGitHubRateLimitHook_HighRemaining(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-RateLimit-Remaining", "4500")
	headers.Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10))

	output := captureStderr(t, func() {
		GitHubRateLimitHook(headers)
	})
	if output != "" {
		t.Errorf("expected no warning for high remaining count, got %q", output)
	}
}

func TestGitHubRateLimitHook_LowRemaining(t *testing.T) {
	resetTime := time.Now().Add(30 * time.Minute)
	headers := http.Header{}
	headers.Set("X-RateLimit-Remaining", "42")
	headers.Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

	output := captureStderr(t, func() {
		GitHubRateLimitHook(headers)
	})

	if !strings.Contains(output, "rate limit low") {
		t.Errorf("expected rate limit warning, got %q", output)
	}
	if !strings.Contains(output, "42 requests remaining") {
		t.Errorf("expected remaining count in warning, got %q", output)
	}
	if !strings.Contains(output, "resets at") {
		t.Errorf("expected reset time in warning, got %q", output)
	}
}

func TestGitHubRateLimitHook_LowRemainingNoReset(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-RateLimit-Remaining", "5")

	output := captureStderr(t, func() {
		GitHubRateLimitHook(headers)
	})

	if !strings.Contains(output, "5 requests remaining") {
		t.Errorf("expected remaining count in warning, got %q", output)
	}
	// Should not crash when reset header is missing
	if strings.Contains(output, "resets at") {
		t.Errorf("should not include reset time when header is missing, got %q", output)
	}
}

func TestGitHubRateLimitHook_ZeroRemaining(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-RateLimit-Remaining", "0")

	output := captureStderr(t, func() {
		GitHubRateLimitHook(headers)
	})

	if !strings.Contains(output, "0 requests remaining") {
		t.Errorf("expected warning for zero remaining, got %q", output)
	}
}

func TestGitHubRateLimitHook_InvalidRemaining(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-RateLimit-Remaining", "not-a-number")

	output := captureStderr(t, func() {
		GitHubRateLimitHook(headers)
	})
	if output != "" {
		t.Errorf("expected no output for invalid remaining value, got %q", output)
	}
}

func TestGitHubRateLimitHook_ExactThreshold(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", rateLimitWarningThreshold))

	output := captureStderr(t, func() {
		GitHubRateLimitHook(headers)
	})
	// Exactly at threshold should NOT warn (we use <, not <=)
	if output != "" {
		t.Errorf("expected no warning at exact threshold, got %q", output)
	}
}
