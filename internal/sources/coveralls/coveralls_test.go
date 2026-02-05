package coveralls

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/boinger/confvis/internal/sources"
)

// mockFetcher implements Fetcher for testing.
type mockFetcher struct {
	report   *ReportResponse
	fetchErr error
}

func (m *mockFetcher) FetchReport(_ context.Context, _, _ string) (*ReportResponse, error) {
	return m.report, m.fetchErr
}

func (m *mockFetcher) ReportURL(service, ownerRepo string) string {
	return "https://coveralls.io/" + service + "/" + ownerRepo
}

func TestSource_FetchWithClient(t *testing.T) {
	tests := []struct {
		name      string
		coverage  float64
		wantScore int
	}{
		{
			name:      "100% coverage",
			coverage:  100.0,
			wantScore: 100,
		},
		{
			name:      "0% coverage",
			coverage:  0.0,
			wantScore: 0,
		},
		{
			name:      "85.5% coverage rounds to 86",
			coverage:  85.5,
			wantScore: 86,
		},
		{
			name:      "85.4% coverage rounds to 85",
			coverage:  85.4,
			wantScore: 85,
		},
		{
			name:      "50.5% coverage rounds to 51",
			coverage:  50.5,
			wantScore: 51,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Source{}
			mock := &mockFetcher{
				report: &ReportResponse{CoveredPercent: tt.coverage},
			}

			report, err := s.FetchWithClient(context.Background(), mock, defaultOpts(), "github")
			if err != nil {
				t.Fatalf("FetchWithClient() error = %v", err)
			}

			if report.ScoreValue() != tt.wantScore {
				t.Errorf("Score = %d, want %d", report.Score, tt.wantScore)
			}

			if report.Source != sourceName {
				t.Errorf("Source = %q, want %q", report.Source, sourceName)
			}

			if len(report.Factors) != 1 {
				t.Errorf("len(Factors) = %d, want 1", len(report.Factors))
			}
		})
	}
}

func TestSource_FetchWithClient_Title(t *testing.T) {
	s := &Source{}
	mock := &mockFetcher{
		report: &ReportResponse{CoveredPercent: 80},
	}

	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
		Title:     "Custom Title",
	}

	report, err := s.FetchWithClient(context.Background(), mock, opts, "github")
	if err != nil {
		t.Fatalf("FetchWithClient() error = %v", err)
	}

	if report.Title != "Custom Title" {
		t.Errorf("Title = %q, want %q", report.Title, "Custom Title")
	}
}

func TestSource_FetchWithClient_DefaultTitle(t *testing.T) {
	s := &Source{}
	mock := &mockFetcher{
		report: &ReportResponse{CoveredPercent: 80},
	}

	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	report, err := s.FetchWithClient(context.Background(), mock, opts, "github")
	if err != nil {
		t.Fatalf("FetchWithClient() error = %v", err)
	}

	if report.Title != "owner/repo" {
		t.Errorf("Title = %q, want %q", report.Title, "owner/repo")
	}
}

func TestSource_FetchWithClient_URL(t *testing.T) {
	s := &Source{}
	mock := &mockFetcher{
		report: &ReportResponse{CoveredPercent: 80},
	}

	opts := sources.Options{
		Project:   "myorg/myrepo",
		Threshold: 75,
	}

	report, err := s.FetchWithClient(context.Background(), mock, opts, "github")
	if err != nil {
		t.Fatalf("FetchWithClient() error = %v", err)
	}

	if len(report.Factors) != 1 {
		t.Fatalf("len(Factors) = %d, want 1", len(report.Factors))
	}

	wantURL := "https://coveralls.io/github/myorg/myrepo"
	if report.Factors[0].URL != wantURL {
		t.Errorf("Factor URL = %q, want %q", report.Factors[0].URL, wantURL)
	}
}

func TestSource_Name(t *testing.T) {
	s := &Source{}
	if s.Name() != sourceName {
		t.Errorf("Name() = %q, want %q", s.Name(), sourceName)
	}
}

func defaultOpts() sources.Options {
	return sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}
}

func TestClient_FetchReport(t *testing.T) {
	report := ReportResponse{
		RepoName:       "owner/repo",
		CoveredPercent: 85.5,
		CoverageChange: 2.5,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/github/owner/repo.json" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(report)
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, "", server.Client())
	result, err := client.FetchReport(context.Background(), "github", "owner/repo")
	if err != nil {
		t.Fatalf("FetchReport() error = %v", err)
	}

	if result.CoveredPercent != 85.5 {
		t.Errorf("CoveredPercent = %f, want 85.5", result.CoveredPercent)
	}
}

func TestClient_FetchReport_WithToken(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ReportResponse{CoveredPercent: 80})
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, "test-token", server.Client())
	_, err := client.FetchReport(context.Background(), "github", "owner/repo")
	if err != nil {
		t.Fatalf("FetchReport() error = %v", err)
	}

	if receivedAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "Bearer test-token")
	}
}

func TestClient_FetchReport_InvalidProject(t *testing.T) {
	client := NewClient("", 0)
	_, err := client.FetchReport(context.Background(), "github", "invalid-format")
	if err == nil {
		t.Error("expected error for invalid project format")
	}
}

func TestClient_ReportURL(t *testing.T) {
	tests := []struct {
		name     string
		service  string
		ownerRepo string
		want     string
	}{
		{"github", "github", "owner/repo", "https://coveralls.io/github/owner/repo"},
		{"gitlab", "gitlab", "owner/repo", "https://coveralls.io/gitlab/owner/repo"},
		{"bitbucket", "bitbucket", "owner/repo", "https://coveralls.io/bitbucket/owner/repo"},
		{"invalid", "github", "invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("", 0)
			got := client.ReportURL(tt.service, tt.ownerRepo)
			if got != tt.want {
				t.Errorf("ReportURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSource_Fetch_Success(t *testing.T) {
	report := ReportResponse{CoveredPercent: 80}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(report)
	}))
	defer server.Close()

	// Override the default base URL for testing
	// Since we can't easily inject the URL, test using FetchWithClient instead
	s := &Source{}
	client := NewClientWithHTTP(server.URL, "", server.Client())

	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	result, err := s.FetchWithClient(context.Background(), client, opts, "github")
	if err != nil {
		t.Fatalf("FetchWithClient() error = %v", err)
	}

	if result.ScoreValue() != 80 {
		t.Errorf("Score = %d, want 80", result.Score)
	}
}
