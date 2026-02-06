package coveralls

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
	s := sources.Get("coveralls")
	if s == nil {
		t.Fatal("coveralls source not registered")
	}
	cs, ok := s.(*coverage.CoverageSource)
	if !ok {
		t.Fatalf("coveralls source is not *coverage.CoverageSource, got %T", s)
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
		{"85.5% coverage", 85.5, 85.5},
		{"50.123% coverage", 50.123, 50.123},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ReportResponse{CoveredPercent: tt.coverage}
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
	s := sources.Get("coveralls")
	if s == nil {
		t.Error("coveralls source not registered")
	}
	if s.Name() != "coveralls" {
		t.Errorf("Name() = %q, want %q", s.Name(), "coveralls")
	}
}

func TestSource_Fetch_Success(t *testing.T) {
	report := ReportResponse{CoveredPercent: 85.5}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/github/owner/repo.json" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(report)
	}))
	defer server.Close()

	// Get the source and use the test helper
	s := getCoverageSource(t)
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	result, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "", server.Client())
	if err != nil {
		t.Fatalf("FetchWithTestClient() error = %v", err)
	}

	// 85.5 rounds to 86
	if result.ScoreValue() != 86 {
		t.Errorf("Score = %d, want 86", result.ScoreValue())
	}

	if result.Source != "coveralls" {
		t.Errorf("Source = %q, want %q", result.Source, "coveralls")
	}

	if len(result.Factors) != 1 {
		t.Errorf("len(Factors) = %d, want 1", len(result.Factors))
	}
}

func TestSource_Fetch_WithToken(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ReportResponse{CoveredPercent: 80})
	}))
	defer server.Close()

	s := getCoverageSource(t)
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	_, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "test-token", server.Client())
	if err != nil {
		t.Fatalf("FetchWithTestClient() error = %v", err)
	}

	if receivedAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "Bearer test-token")
	}
}

func TestSource_Fetch_InvalidProject(t *testing.T) {
	s := getCoverageSource(t)
	opts := sources.Options{
		Project:   "invalid-format", // missing slash
		Threshold: 75,
	}

	_, err := s.FetchWithTestClient(context.Background(), opts, "http://example.com", "", http.DefaultClient)
	if err == nil {
		t.Error("expected error for invalid project format")
	}
}

func TestSource_Fetch_ZeroCoverage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ReportResponse{CoveredPercent: 0})
	}))
	defer server.Close()

	s := getCoverageSource(t)
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	result, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "", server.Client())
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
		_ = json.NewEncoder(w).Encode(ReportResponse{CoveredPercent: 100})
	}))
	defer server.Close()

	s := getCoverageSource(t)
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	result, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "", server.Client())
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
		_ = json.NewEncoder(w).Encode(ReportResponse{CoveredPercent: 80})
	}))
	defer server.Close()

	s := getCoverageSource(t)
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
		Title:     "Custom Title",
	}

	result, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "", server.Client())
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
		_ = json.NewEncoder(w).Encode(ReportResponse{CoveredPercent: 80})
	}))
	defer server.Close()

	s := getCoverageSource(t)
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	result, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "", server.Client())
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
		_ = json.NewEncoder(w).Encode(ReportResponse{CoveredPercent: 80})
	}))
	defer server.Close()

	s := getCoverageSource(t)
	opts := sources.Options{
		Project:   "myorg/myrepo",
		Threshold: 75,
	}

	result, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "", server.Client())
	if err != nil {
		t.Fatalf("FetchWithTestClient() error = %v", err)
	}

	if len(result.Factors) != 1 {
		t.Fatalf("len(Factors) = %d, want 1", len(result.Factors))
	}

	expectedURL := "https://coveralls.io/github/myorg/myrepo"
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
		{"rounds up from .5", 85.5, 86},
		{"rounds up from .6", 85.6, 86},
		{"rounds down from .4", 85.4, 85},
		{"rounds down from .1", 85.1, 85},
		{"exact integer", 90.0, 90},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(ReportResponse{CoveredPercent: tt.coverage})
			}))
			defer server.Close()

			s := getCoverageSource(t)
			opts := sources.Options{
				Project:   "owner/repo",
				Threshold: 75,
			}

			result, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "", server.Client())
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
		_ = json.NewEncoder(w).Encode(ReportResponse{CoveredPercent: 80})
	}))
	defer server.Close()

	s := getCoverageSource(t)
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
		Extra:     map[string]string{"service": "gitlab"},
	}

	_, err := s.FetchWithTestClient(context.Background(), opts, server.URL, "", server.Client())
	if err != nil {
		t.Fatalf("FetchWithTestClient() error = %v", err)
	}

	expectedPath := "/gitlab/owner/repo.json"
	if receivedPath != expectedPath {
		t.Errorf("path = %q, want %q", receivedPath, expectedPath)
	}
}
