package codecov

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/boinger/confvis/internal/sources"
)

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
	client := &Client{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

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

	client := &Client{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

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

	client := &Client{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

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

	client := &Client{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

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

	client := &Client{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

	ctx := context.Background()
	report, err := client.FetchReport(ctx, "github", "owner/repo")
	if err != nil {
		t.Fatalf("FetchReport() error = %v", err)
	}

	// Verify score truncation (75.5 -> 75)
	score := int(report.Totals.Coverage)
	if score != 75 {
		t.Errorf("score = %d, want 75", score)
	}
}
