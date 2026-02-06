package coverage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/boinger/confvis/internal/sources"
)

func TestNewSource(t *testing.T) {
	cfg := SourceConfig{
		Name:        "test-source",
		TokenEnvVar: "TEST_TOKEN",
		BaseURL:     "https://example.com",
		BuildAPIPath: func(service, owner, repo string) string {
			return fmt.Sprintf("/%s/%s/%s", service, owner, repo)
		},
		BuildWebURL: func(service, owner, repo string) string {
			return fmt.Sprintf("https://example.com/%s/%s/%s", service, owner, repo)
		},
	}
	extractor := func(data []byte) (float64, error) {
		return 85.5, nil
	}

	s := NewSource(cfg, extractor)

	if s.Name() != "test-source" {
		t.Errorf("Name() = %q, want %q", s.Name(), "test-source")
	}
}

func TestCoverageSource_Fetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/github/owner/repo" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]float64{"coverage": 85.5})
	}))
	defer server.Close()

	cfg := SourceConfig{
		Name:          "test-coverage",
		TokenEnvVar:   "TEST_TOKEN",
		TokenRequired: false,
		BaseURL:       server.URL,
		BuildAPIPath: func(service, owner, repo string) string {
			return fmt.Sprintf("/%s/%s/%s", service, owner, repo)
		},
		BuildWebURL: func(service, owner, repo string) string {
			return fmt.Sprintf("https://example.com/%s/%s/%s", service, owner, repo)
		},
	}
	extractor := func(data []byte) (float64, error) {
		var resp map[string]float64
		if err := json.Unmarshal(data, &resp); err != nil {
			return 0, err
		}
		return resp["coverage"], nil
	}

	s := NewSource(cfg, extractor)

	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// 85.5 rounds to 86
	if report.ScoreValue() != 86 {
		t.Errorf("Score = %d, want 86", report.ScoreValue())
	}

	if report.Source != "test-coverage" {
		t.Errorf("Source = %q, want %q", report.Source, "test-coverage")
	}
}

func TestCoverageSource_Fetch_TokenRequired(t *testing.T) {
	cfg := SourceConfig{
		Name:          "test-coverage",
		TokenEnvVar:   "TEST_COVERAGE_TOKEN",
		TokenRequired: true,
		BaseURL:       "https://example.com",
		BuildAPIPath: func(service, owner, repo string) string {
			return fmt.Sprintf("/%s/%s/%s", service, owner, repo)
		},
		BuildWebURL: func(service, owner, repo string) string {
			return fmt.Sprintf("https://example.com/%s/%s/%s", service, owner, repo)
		},
	}
	extractor := func(data []byte) (float64, error) { return 0, nil }

	s := NewSource(cfg, extractor)

	t.Setenv("TEST_COVERAGE_TOKEN", "")

	opts := sources.Options{
		Project: "owner/repo",
	}

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error when token is required but missing")
	}
}

func TestCoverageSource_Fetch_InvalidProject(t *testing.T) {
	cfg := SourceConfig{
		Name:          "test-coverage",
		TokenEnvVar:   "TEST_TOKEN",
		TokenRequired: false,
		BaseURL:       "https://example.com",
		BuildAPIPath: func(service, owner, repo string) string {
			return fmt.Sprintf("/%s/%s/%s", service, owner, repo)
		},
		BuildWebURL: func(service, owner, repo string) string {
			return fmt.Sprintf("https://example.com/%s/%s/%s", service, owner, repo)
		},
	}
	extractor := func(data []byte) (float64, error) { return 0, nil }

	s := NewSource(cfg, extractor)

	opts := sources.Options{
		Project: "invalid-no-slash",
	}

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error for invalid project format")
	}
}

func TestCoverageSource_Fetch_WithToken(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]float64{"coverage": 80})
	}))
	defer server.Close()

	cfg := SourceConfig{
		Name:          "test-coverage",
		TokenEnvVar:   "TEST_TOKEN",
		TokenRequired: false,
		BaseURL:       server.URL,
		BuildAPIPath: func(service, owner, repo string) string {
			return fmt.Sprintf("/%s/%s/%s", service, owner, repo)
		},
		BuildWebURL: func(service, owner, repo string) string {
			return fmt.Sprintf("https://example.com/%s/%s/%s", service, owner, repo)
		},
	}
	extractor := func(data []byte) (float64, error) {
		var resp map[string]float64
		if err := json.Unmarshal(data, &resp); err != nil {
			return 0, err
		}
		return resp["coverage"], nil
	}

	s := NewSource(cfg, extractor)

	opts := sources.Options{
		Project: "owner/repo",
		Token:   "test-token",
	}

	_, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if receivedAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "Bearer test-token")
	}
}

func TestCoverageSource_Fetch_ExtractorError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]float64{"coverage": 80})
	}))
	defer server.Close()

	cfg := SourceConfig{
		Name:          "test-coverage",
		TokenEnvVar:   "TEST_TOKEN",
		TokenRequired: false,
		BaseURL:       server.URL,
		BuildAPIPath: func(service, owner, repo string) string {
			return fmt.Sprintf("/%s/%s/%s", service, owner, repo)
		},
		BuildWebURL: func(service, owner, repo string) string {
			return fmt.Sprintf("https://example.com/%s/%s/%s", service, owner, repo)
		},
	}
	extractor := func(data []byte) (float64, error) {
		return 0, fmt.Errorf("extractor error")
	}

	s := NewSource(cfg, extractor)

	opts := sources.Options{
		Project: "owner/repo",
	}

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error from extractor")
	}
}

func TestCoverageSource_Fetch_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := SourceConfig{
		Name:          "test-coverage",
		TokenEnvVar:   "TEST_TOKEN",
		TokenRequired: false,
		BaseURL:       server.URL,
		BuildAPIPath: func(service, owner, repo string) string {
			return fmt.Sprintf("/%s/%s/%s", service, owner, repo)
		},
		BuildWebURL: func(service, owner, repo string) string {
			return fmt.Sprintf("https://example.com/%s/%s/%s", service, owner, repo)
		},
	}
	extractor := func(data []byte) (float64, error) { return 0, nil }

	s := NewSource(cfg, extractor)

	opts := sources.Options{
		Project: "owner/repo",
	}

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestCoverageSource_Fetch_WithTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]float64{"coverage": 80})
	}))
	defer server.Close()

	cfg := SourceConfig{
		Name:          "test-coverage",
		TokenEnvVar:   "TEST_TOKEN",
		TokenRequired: false,
		BaseURL:       server.URL,
		BuildAPIPath: func(service, owner, repo string) string {
			return fmt.Sprintf("/%s/%s/%s", service, owner, repo)
		},
		BuildWebURL: func(service, owner, repo string) string {
			return fmt.Sprintf("https://example.com/%s/%s/%s", service, owner, repo)
		},
	}
	extractor := func(data []byte) (float64, error) {
		var resp map[string]float64
		if err := json.Unmarshal(data, &resp); err != nil {
			return 0, err
		}
		return resp["coverage"], nil
	}

	s := NewSource(cfg, extractor)

	opts := sources.Options{
		Project: "owner/repo",
		Title:   "Custom Title",
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if report.Title != "Custom Title" {
		t.Errorf("Title = %q, want %q", report.Title, "Custom Title")
	}
}

func TestCoverageSource_createClient_NoToken(t *testing.T) {
	cfg := SourceConfig{
		Name:    "test",
		BaseURL: "https://example.com",
	}
	s := NewSource(cfg, nil)

	client := s.createClient("", 30)
	if client == nil {
		t.Error("createClient returned nil")
	}
}

func TestCoverageSource_createClient_WithToken(t *testing.T) {
	cfg := SourceConfig{
		Name:    "test",
		BaseURL: "https://example.com",
	}
	s := NewSource(cfg, nil)

	client := s.createClient("some-token", 30)
	if client == nil {
		t.Error("createClient returned nil")
	}
}
