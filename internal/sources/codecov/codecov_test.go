package codecov

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/coverage"
)

// getCoverageSource is a test helper that safely asserts the source type.
func getCoverageSource(t *testing.T) *coverage.CoverageSource {
	t.Helper()
	s := sources.Get("codecov")
	if s == nil {
		t.Fatal("codecov source not registered")
	}
	cs, ok := s.(*coverage.CoverageSource)
	if !ok {
		t.Fatalf("codecov source is not *coverage.CoverageSource, got %T", s)
	}
	return cs
}

func TestExtractCoverage(t *testing.T) {
	tests := []struct {
		name     string
		coverage float64
		want     float64
	}{
		{"100% coverage", 100.0, 100.0},
		{"0% coverage", 0.0, 0.0},
		{"83.5% coverage", 83.5, 83.5},
		{"50.123% coverage", 50.123, 50.123},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ReportResponse{Totals: Totals{Coverage: tt.coverage}}
			data, err := json.Marshal(resp)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			got, err := extractCoverage(data)
			if err != nil {
				t.Fatalf("extractCoverage() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("extractCoverage() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestExtractCoverage_InvalidJSON(t *testing.T) {
	_, err := extractCoverage([]byte("invalid json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSource_Registration(t *testing.T) {
	s := sources.Get("codecov")
	if s == nil {
		t.Error("codecov source not registered")
	}
	if s.Name() != "codecov" {
		t.Errorf("Name() = %q, want %q", s.Name(), "codecov")
	}
}

func TestSource_Fetch_MissingToken(t *testing.T) {
	t.Setenv("CODECOV_TOKEN", "")

	s := sources.Get("codecov")
	opts := sources.Options{
		Project: "owner/repo",
	}

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error for missing token")
	}
}

func TestSource_Fetch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v2/github/owner/repos/repo/report/"
		if r.URL.Path != expectedPath {
			t.Errorf("path = %q, want %q", r.URL.Path, expectedPath)
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test-token")
		}

		resp := ReportResponse{Totals: Totals{Coverage: 83.5}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	s := getCoverageSource(t)
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	result, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "test-token", server.Client())
	if err != nil {
		t.Fatalf("FetchWithTestClient() error = %v", err)
	}

	// 83.5 rounds to 84
	if result.ScoreValue() != 84 {
		t.Errorf("Score = %d, want 84", result.ScoreValue())
	}

	if result.Source != "codecov" {
		t.Errorf("Source = %q, want %q", result.Source, "codecov")
	}
}

func TestSource_Fetch_InvalidProject(t *testing.T) {
	s := getCoverageSource(t)
	opts := sources.Options{
		Project: "invalid-no-slash",
	}

	_, err := s.FetchWithTestClient(context.Background(), opts, "http://example.com", "test-token", http.DefaultClient)
	if err == nil {
		t.Error("expected error for invalid project format")
	}
}

func TestSource_Fetch_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"detail": "Not found"}`, http.StatusNotFound)
	}))
	defer server.Close()

	s := getCoverageSource(t)
	opts := sources.Options{
		Project: "owner/repo",
	}

	_, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "test-token", server.Client())
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestSource_Fetch_ZeroCoverage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ReportResponse{Totals: Totals{Coverage: 0}})
	}))
	defer server.Close()

	s := getCoverageSource(t)
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	result, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "test-token", server.Client())
	if err != nil {
		t.Fatalf("FetchWithTestClient() error = %v", err)
	}

	if result.ScoreValue() != 0 {
		t.Errorf("Score = %d, want 0", result.ScoreValue())
	}
}

func TestSource_Fetch_FullCoverage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ReportResponse{Totals: Totals{Coverage: 100}})
	}))
	defer server.Close()

	s := getCoverageSource(t)
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	result, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "test-token", server.Client())
	if err != nil {
		t.Fatalf("FetchWithTestClient() error = %v", err)
	}

	if result.ScoreValue() != 100 {
		t.Errorf("Score = %d, want 100", result.ScoreValue())
	}
}

func TestSource_Fetch_WithTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ReportResponse{Totals: Totals{Coverage: 80}})
	}))
	defer server.Close()

	s := getCoverageSource(t)
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
		Title:     "Custom Title",
	}

	result, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "test-token", server.Client())
	if err != nil {
		t.Fatalf("FetchWithTestClient() error = %v", err)
	}

	if result.Title != "Custom Title" {
		t.Errorf("Title = %q, want %q", result.Title, "Custom Title")
	}
}

func TestSource_Fetch_DefaultTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ReportResponse{Totals: Totals{Coverage: 80}})
	}))
	defer server.Close()

	s := getCoverageSource(t)
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	result, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "test-token", server.Client())
	if err != nil {
		t.Fatalf("FetchWithTestClient() error = %v", err)
	}

	if result.Title != "owner/repo" {
		t.Errorf("Title = %q, want %q", result.Title, "owner/repo")
	}
}

func TestSource_Fetch_FactorURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ReportResponse{Totals: Totals{Coverage: 80}})
	}))
	defer server.Close()

	s := getCoverageSource(t)
	opts := sources.Options{
		Project:   "myorg/myrepo",
		Threshold: 75,
	}

	result, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "test-token", server.Client())
	if err != nil {
		t.Fatalf("FetchWithTestClient() error = %v", err)
	}

	if len(result.Factors) != 1 {
		t.Fatalf("len(Factors) = %d, want 1", len(result.Factors))
	}

	expectedURL := "https://app.codecov.io/github/myorg/myrepo"
	if result.Factors[0].URL != expectedURL {
		t.Errorf("Factor URL = %q, want %q", result.Factors[0].URL, expectedURL)
	}
}

func TestSource_Fetch_RoundingBehavior(t *testing.T) {
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(ReportResponse{Totals: Totals{Coverage: tt.coverage}})
			}))
			defer server.Close()

			s := getCoverageSource(t)
			opts := sources.Options{
				Project:   "owner/repo",
				Threshold: 75,
			}

			result, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "test-token", server.Client())
			if err != nil {
				t.Fatalf("FetchWithTestClient() error = %v", err)
			}

			if result.ScoreValue() != tt.want {
				t.Errorf("Score = %d, want %d for coverage %f", result.ScoreValue(), tt.want, tt.coverage)
			}
		})
	}
}

func TestSource_Fetch_GitLabService(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ReportResponse{Totals: Totals{Coverage: 90}})
	}))
	defer server.Close()

	s := getCoverageSource(t)
	opts := sources.Options{
		Project:   "group/project",
		Threshold: 75,
		Extra:     map[string]string{"service": "gitlab"},
	}

	_, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "test-token", server.Client())
	if err != nil {
		t.Fatalf("FetchWithTestClient() error = %v", err)
	}

	expectedPath := "/api/v2/gitlab/group/repos/project/report/"
	if receivedPath != expectedPath {
		t.Errorf("path = %q, want %q", receivedPath, expectedPath)
	}
}

func TestSource_Fetch_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	s := getCoverageSource(t)
	opts := sources.Options{
		Project: "owner/repo",
	}

	_, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "test-token", server.Client())
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}
