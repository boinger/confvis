package httpclient

import (
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
