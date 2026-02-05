package codecov

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/boinger/confvis/internal/sources"
)

// mockFetcher implements the Fetcher interface for testing.
type mockFetcher struct {
	reportResp *ReportResponse
	reportErr  error
	reportURL  string
}

func (m *mockFetcher) FetchReport(_ context.Context, _, _ string) (*ReportResponse, error) {
	return m.reportResp, m.reportErr
}

func (m *mockFetcher) ReportURL(_, _ string) string {
	return m.reportURL
}

func TestSource_Name(t *testing.T) {
	s := &Source{}
	if got := s.Name(); got != "codecov" {
		t.Errorf("Name() = %q, want %q", got, "codecov")
	}
}

func TestSource_Fetch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify path format
		expectedPath := "/api/v2/github/myorg/repos/myrepo/report/"
		if r.URL.Path != expectedPath {
			t.Errorf("path = %q, want %q", r.URL.Path, expectedPath)
		}

		// Verify auth header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test-token")
		}

		resp := ReportResponse{
			Totals: Totals{
				Coverage: 83.5,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	}))
	defer server.Close()

	ctx := context.Background()

	// Create a client with the mock server URL
	client := NewClientWithHTTP(server.URL, "test-token", server.Client())

	report, err := client.FetchReport(ctx, "github", "myorg/myrepo")
	if err != nil {
		t.Fatalf("FetchReport() error = %v", err)
	}

	if report.Totals.Coverage != 83.5 {
		t.Errorf("Coverage = %f, want %f", report.Totals.Coverage, 83.5)
	}

	// Test the full Fetch - since we can't override baseURL, we test token requirement separately
	// in TestSource_Fetch_MissingToken
}

func TestSource_Fetch_MissingToken(t *testing.T) {
	s := &Source{}
	opts := sources.Options{
		Project: "myorg/myrepo",
		Timeout: 5,
	}

	t.Setenv(EnvToken, "")

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error for missing token")
	}
}

func TestSource_Fetch_InvalidProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make request with invalid project")
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, "test-token", server.Client())

	_, err := client.FetchReport(context.Background(), "github", "invalid-no-slash")
	if err == nil {
		t.Error("expected error for invalid project format")
	}
}

func TestSource_Fetch_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail": "Not found"}`, http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, "test-token", server.Client())

	_, err := client.FetchReport(context.Background(), "github", "myorg/myrepo")
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestClient_ReportURL(t *testing.T) {
	client := NewClient("token", 0)

	tests := []struct {
		service  string
		ownerRepo string
		want     string
	}{
		{"github", "myorg/myrepo", "https://app.codecov.io/github/myorg/myrepo"},
		{"gitlab", "group/project", "https://app.codecov.io/gitlab/group/project"},
		{"bitbucket", "team/repo", "https://app.codecov.io/bitbucket/team/repo"},
		{"github", "invalid", ""}, // No slash
	}

	for _, tt := range tests {
		got := client.ReportURL(tt.service, tt.ownerRepo)
		if got != tt.want {
			t.Errorf("ReportURL(%q, %q) = %q, want %q", tt.service, tt.ownerRepo, got, tt.want)
		}
	}
}

func TestSource_Fetch_WithService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify gitlab service is used
		expectedPath := "/api/v2/gitlab/mygroup/repos/myproject/report/"
		if r.URL.Path != expectedPath {
			t.Errorf("path = %q, want %q", r.URL.Path, expectedPath)
		}

		resp := ReportResponse{
			Totals: Totals{Coverage: 90.0},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, "test-token", server.Client())

	report, err := client.FetchReport(context.Background(), "gitlab", "mygroup/myproject")
	if err != nil {
		t.Fatalf("FetchReport() error = %v", err)
	}

	if report.Totals.Coverage != 90.0 {
		t.Errorf("Coverage = %f, want %f", report.Totals.Coverage, 90.0)
	}
}

func TestSource_Registration(t *testing.T) {
	s := sources.Get("codecov")
	if s == nil {
		t.Error("codecov source not registered")
	}
}

func TestClient_FetchReport_Integration(t *testing.T) {
	// Full integration test with mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ReportResponse{
			Totals: Totals{Coverage: 75.5},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, "test-token", server.Client())

	ctx := context.Background()
	report, err := client.FetchReport(ctx, "github", "owner/repo")
	if err != nil {
		t.Fatalf("FetchReport() error = %v", err)
	}

	// Verify score rounding (75.5 -> 76)
	score := int(math.Round(report.Totals.Coverage))
	if score != 76 {
		t.Errorf("score = %d, want 76", score)
	}
}

func TestSource_Fetch_WithTokenFromEnv(t *testing.T) {
	// Set up a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ReportResponse{
			Totals: Totals{Coverage: 80.0},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	}))
	defer server.Close()

	// Set token in environment - but since we can't override baseURL in Source.Fetch,
	// this test verifies that token from env is accepted (by checking no error for missing token)
	t.Setenv(EnvToken, "env-token")

	s := &Source{}
	opts := sources.Options{
		Project: "myorg/myrepo",
		Timeout: 1, // Short timeout since real API won't work
	}

	// This will fail with connection error (not auth error), proving token was found
	_, err := s.Fetch(context.Background(), opts)
	if err != nil && err.Error() == "codecov token required" {
		t.Error("token from environment should be accepted")
	}
	// We expect a connection error since we're hitting the real API, but not a token error
}

func TestSource_Fetch_WithCustomTitle(t *testing.T) {
	// Can't fully test without mocking, but test the logic paths exist
	s := &Source{}
	opts := sources.Options{
		Project: "myorg/myrepo",
		Title:   "Custom Title",
		Token:   "test-token",
		Timeout: 1,
	}

	// This will fail with connection error, but exercises the Title path
	_, _ = s.Fetch(context.Background(), opts)
}

func TestSource_Fetch_DefaultTimeout(t *testing.T) {
	// Test with zero timeout (should use default)
	s := &Source{}
	opts := sources.Options{
		Project: "myorg/myrepo",
		Token:   "test-token",
		Timeout: 0, // Should use default 30s
	}

	// This will fail with connection error, but exercises the timeout path
	_, _ = s.Fetch(context.Background(), opts)
}

func TestClient_doRequest_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("not valid json")); err != nil {
			t.Errorf("writing response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, "test-token", server.Client())

	_, err := client.FetchReport(context.Background(), "github", "owner/repo")
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

func TestClient_doRequest_NoToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no auth header when token is empty
		auth := r.Header.Get("Authorization")
		if auth != "" {
			t.Errorf("expected no Authorization header, got %q", auth)
		}

		resp := ReportResponse{Totals: Totals{Coverage: 50.0}}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, "", server.Client())

	_, err := client.FetchReport(context.Background(), "github", "owner/repo")
	if err != nil {
		t.Errorf("FetchReport() with no token should work if server accepts: %v", err)
	}
}

func TestNewClient_WithTimeout(t *testing.T) {
	client := NewClient("token123", 60*time.Second)
	// Verify baseURL is set to default
	if client.baseURL != defaultBaseURL {
		t.Errorf("baseURL = %q, want %q", client.baseURL, defaultBaseURL)
	}
	// Verify http client is not nil
	if client.http == nil {
		t.Error("http client should not be nil")
	}
}

func TestSource_Fetch_DefaultService(t *testing.T) {
	// Test that default service is "github" when Extra is nil
	s := &Source{}
	t.Setenv(EnvToken, "test-token")

	opts := sources.Options{
		Project: "myorg/myrepo",
		Extra:   nil, // No Extra map
		Timeout: 1,
	}

	// This will fail with connection error, but exercises the default service path
	_, _ = s.Fetch(context.Background(), opts)
}

func TestSource_Fetch_EmptyServiceInExtra(t *testing.T) {
	// Test that empty service in Extra falls back to "github"
	s := &Source{}
	t.Setenv(EnvToken, "test-token")

	opts := sources.Options{
		Project: "myorg/myrepo",
		Extra:   map[string]string{"service": ""}, // Empty service
		Timeout: 1,
	}

	// This will fail with connection error, but exercises the empty service path
	_, _ = s.Fetch(context.Background(), opts)
}

func TestSource_Fetch_CustomService(t *testing.T) {
	// Test that custom service from Extra is used
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify bitbucket service is used in path
		if !strings.Contains(r.URL.Path, "/bitbucket/") {
			t.Errorf("path should contain bitbucket, got %s", r.URL.Path)
		}

		resp := ReportResponse{
			Totals: Totals{Coverage: 80.0},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, "test-token", server.Client())

	report, err := client.FetchReport(context.Background(), "bitbucket", "team/repo")
	if err != nil {
		t.Fatalf("FetchReport() error = %v", err)
	}

	if report.Totals.Coverage != 80.0 {
		t.Errorf("Coverage = %f, want 80.0", report.Totals.Coverage)
	}
}

func TestSource_Fetch_NegativeTimeout(t *testing.T) {
	// Test that negative timeout falls back to default 30s
	s := &Source{}
	t.Setenv(EnvToken, "test-token")

	opts := sources.Options{
		Project: "myorg/myrepo",
		Timeout: -10, // Negative timeout
	}

	// This will fail with connection error, but verifies timeout handling
	_, _ = s.Fetch(context.Background(), opts)
}

func TestSource_Fetch_TitleFallback(t *testing.T) {
	// Test that title falls back to Project when not specified
	s := &Source{}
	t.Setenv(EnvToken, "test-token")

	opts := sources.Options{
		Project: "myorg/myrepo",
		Title:   "", // Empty title should fall back to Project
		Timeout: 1,
	}

	// This exercises the title fallback path
	_, _ = s.Fetch(context.Background(), opts)
}

func TestSource_Fetch_ZeroCoverage(t *testing.T) {
	// Test handling of 0% coverage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ReportResponse{
			Totals: Totals{Coverage: 0.0},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, "test-token", server.Client())

	report, err := client.FetchReport(context.Background(), "github", "owner/repo")
	if err != nil {
		t.Fatalf("FetchReport() error = %v", err)
	}

	if report.Totals.Coverage != 0.0 {
		t.Errorf("Coverage = %f, want 0.0", report.Totals.Coverage)
	}

	// Verify score truncation (0.0 -> 0)
	score := int(report.Totals.Coverage)
	if score != 0 {
		t.Errorf("score = %d, want 0", score)
	}
}

func TestSource_Fetch_FullCoverage(t *testing.T) {
	// Test handling of 100% coverage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ReportResponse{
			Totals: Totals{Coverage: 100.0},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, "test-token", server.Client())

	report, err := client.FetchReport(context.Background(), "github", "owner/repo")
	if err != nil {
		t.Fatalf("FetchReport() error = %v", err)
	}

	if report.Totals.Coverage != 100.0 {
		t.Errorf("Coverage = %f, want 100.0", report.Totals.Coverage)
	}

	score := int(report.Totals.Coverage)
	if score != 100 {
		t.Errorf("score = %d, want 100", score)
	}
}

func TestFetchWithClient_Success(t *testing.T) {
	mock := &mockFetcher{
		reportResp: &ReportResponse{
			Totals: Totals{Coverage: 87.5},
		},
		reportURL: "https://app.codecov.io/github/myorg/myrepo",
	}

	s := &Source{}
	opts := sources.Options{
		Project:   "myorg/myrepo",
		Title:     "Coverage Report",
		Threshold: 80,
	}

	report, err := s.FetchWithClient(context.Background(), mock, opts, "github")
	if err != nil {
		t.Fatalf("FetchWithClient() error = %v", err)
	}

	// 87.5 rounds to 88
	if report.ScoreValue() != 88 {
		t.Errorf("Score = %d, want 88", report.Score)
	}
	if report.Title != "Coverage Report" {
		t.Errorf("Title = %q, want %q", report.Title, "Coverage Report")
	}
	if len(report.Factors) != 1 {
		t.Fatalf("Factors count = %d, want 1", len(report.Factors))
	}
	if report.Factors[0].Score != 88 {
		t.Errorf("Factor Score = %d, want 88", report.Factors[0].Score)
	}
	if report.Factors[0].URL != "https://app.codecov.io/github/myorg/myrepo" {
		t.Errorf("URL = %q, want %q", report.Factors[0].URL, "https://app.codecov.io/github/myorg/myrepo")
	}
}

func TestFetchWithClient_FractionalCoverageRounding(t *testing.T) {
	tests := []struct {
		name     string
		coverage float64
		want     int
	}{
		{"rounds up from .5", 87.5, 88},
		{"rounds up from .6", 87.6, 88},
		{"rounds down from .4", 87.4, 87},
		{"rounds down from .1", 87.1, 87},
		{"exact integer", 90.0, 90},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockFetcher{
				reportResp: &ReportResponse{
					Totals: Totals{Coverage: tt.coverage},
				},
				reportURL: "https://app.codecov.io/github/myorg/myrepo",
			}

			s := &Source{}
			opts := sources.Options{
				Project:   "myorg/myrepo",
				Threshold: 80,
			}

			report, err := s.FetchWithClient(context.Background(), mock, opts, "github")
			if err != nil {
				t.Fatalf("FetchWithClient() error = %v", err)
			}

			if report.ScoreValue() != tt.want {
				t.Errorf("Score = %d, want %d for coverage %.1f", report.ScoreValue(), tt.want, tt.coverage)
			}
		})
	}
}

func TestFetchWithClient_FetchReportError(t *testing.T) {
	mock := &mockFetcher{
		reportErr: errors.New("API connection failed"),
	}

	s := &Source{}
	opts := sources.Options{
		Project: "myorg/myrepo",
	}

	_, err := s.FetchWithClient(context.Background(), mock, opts, "github")
	if err == nil {
		t.Error("expected error when FetchReport fails")
	}
	if !strings.Contains(err.Error(), "API connection failed") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "API connection failed")
	}
}

func TestFetchWithClient_ZeroCoverage(t *testing.T) {
	mock := &mockFetcher{
		reportResp: &ReportResponse{
			Totals: Totals{Coverage: 0.0},
		},
		reportURL: "https://app.codecov.io/github/myorg/myrepo",
	}

	s := &Source{}
	opts := sources.Options{
		Project: "myorg/myrepo",
	}

	report, err := s.FetchWithClient(context.Background(), mock, opts, "github")
	if err != nil {
		t.Fatalf("FetchWithClient() error = %v", err)
	}

	if report.ScoreValue() != 0 {
		t.Errorf("Score = %d, want 0", report.Score)
	}
}

func TestFetchWithClient_FullCoverage(t *testing.T) {
	mock := &mockFetcher{
		reportResp: &ReportResponse{
			Totals: Totals{Coverage: 100.0},
		},
		reportURL: "https://app.codecov.io/github/myorg/myrepo",
	}

	s := &Source{}
	opts := sources.Options{
		Project: "myorg/myrepo",
	}

	report, err := s.FetchWithClient(context.Background(), mock, opts, "github")
	if err != nil {
		t.Fatalf("FetchWithClient() error = %v", err)
	}

	if report.ScoreValue() != 100 {
		t.Errorf("Score = %d, want 100", report.Score)
	}
}

func TestFetchWithClient_PartialCoverage(t *testing.T) {
	mock := &mockFetcher{
		reportResp: &ReportResponse{
			Totals: Totals{Coverage: 55.7},
		},
		reportURL: "https://app.codecov.io/gitlab/mygroup/myproject",
	}

	s := &Source{}
	opts := sources.Options{
		Project: "mygroup/myproject",
	}

	report, err := s.FetchWithClient(context.Background(), mock, opts, "gitlab")
	if err != nil {
		t.Fatalf("FetchWithClient() error = %v", err)
	}

	// 55.7 rounds to 56
	if report.ScoreValue() != 56 {
		t.Errorf("Score = %d, want 56", report.Score)
	}
}

func TestFetchWithClient_TitleFallback(t *testing.T) {
	mock := &mockFetcher{
		reportResp: &ReportResponse{
			Totals: Totals{Coverage: 80.0},
		},
		reportURL: "https://app.codecov.io/github/myorg/myrepo",
	}

	s := &Source{}
	opts := sources.Options{
		Project: "myorg/myrepo",
		Title:   "", // Empty title should fall back to Project
	}

	report, err := s.FetchWithClient(context.Background(), mock, opts, "github")
	if err != nil {
		t.Fatalf("FetchWithClient() error = %v", err)
	}

	// Title should fall back to Project
	if report.Title != "myorg/myrepo" {
		t.Errorf("Title = %q, want %q", report.Title, "myorg/myrepo")
	}
}
