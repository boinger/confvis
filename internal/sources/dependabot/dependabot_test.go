package dependabot

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
	alerts   AlertsResponse
	alertErr error
}

func (m *mockFetcher) FetchAlerts(_ context.Context, _, _ string) (AlertsResponse, error) {
	return m.alerts, m.alertErr
}

func (m *mockFetcher) AlertsURL(owner, repo string) string {
	return "https://github.com/" + owner + "/" + repo + "/security/dependabot"
}

func TestCountAlertsBySeverity(t *testing.T) {
	alerts := AlertsResponse{
		{Number: 1, SecurityAdvisory: SecurityAdvisory{Severity: "critical"}},
		{Number: 2, SecurityAdvisory: SecurityAdvisory{Severity: "critical"}},
		{Number: 3, SecurityAdvisory: SecurityAdvisory{Severity: "high"}},
		{Number: 4, SecurityAdvisory: SecurityAdvisory{Severity: "medium"}},
		{Number: 5, SecurityAdvisory: SecurityAdvisory{Severity: "medium"}},
		{Number: 6, SecurityAdvisory: SecurityAdvisory{Severity: "medium"}},
		{Number: 7, SecurityAdvisory: SecurityAdvisory{Severity: "low"}},
	}

	counts := CountAlertsBySeverity(alerts)

	if counts.Critical != 2 {
		t.Errorf("Critical = %d, want 2", counts.Critical)
	}
	if counts.High != 1 {
		t.Errorf("High = %d, want 1", counts.High)
	}
	if counts.Medium != 3 {
		t.Errorf("Medium = %d, want 3", counts.Medium)
	}
	if counts.Low != 1 {
		t.Errorf("Low = %d, want 1", counts.Low)
	}
}

func TestSource_FetchWithClient(t *testing.T) {
	tests := []struct {
		name      string
		alerts    AlertsResponse
		wantScore int
	}{
		{
			name:      "no vulnerabilities",
			alerts:    AlertsResponse{},
			wantScore: 100,
		},
		{
			name: "one critical",
			alerts: AlertsResponse{
				{Number: 1, SecurityAdvisory: SecurityAdvisory{Severity: "critical"}},
			},
			wantScore: 90, // (75*40 + 100*30 + 100*20 + 100*10) / 100 = 90
		},
		{
			name: "mixed severities",
			alerts: AlertsResponse{
				{Number: 1, SecurityAdvisory: SecurityAdvisory{Severity: "critical"}},
				{Number: 2, SecurityAdvisory: SecurityAdvisory{Severity: "high"}},
				{Number: 3, SecurityAdvisory: SecurityAdvisory{Severity: "medium"}},
				{Number: 4, SecurityAdvisory: SecurityAdvisory{Severity: "low"}},
			},
			wantScore: 84, // (75*40 + 85*30 + 95*20 + 98*10) / 100 = 84.3 -> 84
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Source{}
			mock := &mockFetcher{alerts: tt.alerts}

			report, err := s.FetchWithClient(context.Background(), mock, defaultOpts(), "owner", "repo")
			if err != nil {
				t.Fatalf("FetchWithClient() error = %v", err)
			}

			if report.Score != tt.wantScore {
				t.Errorf("Score = %d, want %d", report.Score, tt.wantScore)
			}

			if report.Source != sourceName {
				t.Errorf("Source = %q, want %q", report.Source, sourceName)
			}

			if len(report.Factors) != 4 {
				t.Errorf("len(Factors) = %d, want 4", len(report.Factors))
			}
		})
	}
}

func TestClient_FetchAlerts(t *testing.T) {
	alerts := AlertsResponse{
		{Number: 1, SecurityAdvisory: SecurityAdvisory{Severity: "high", GHSAID: "GHSA-1234"}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/dependabot/alerts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("state") != "open" {
			t.Errorf("expected state=open query param")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(alerts)
	}))
	defer server.Close()

	client := NewClientWithHTTP(server.URL, "test-token", server.Client())
	result, err := client.FetchAlerts(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("FetchAlerts() error = %v", err)
	}

	if len(result) != 1 {
		t.Errorf("len(result) = %d, want 1", len(result))
	}
}

func TestClient_AlertsURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		owner   string
		repo    string
		want    string
	}{
		{"default", "", "owner", "repo", "https://github.com/owner/repo/security/dependabot"},
		{"explicit github", "https://api.github.com", "my-org", "my-repo", "https://github.com/my-org/my-repo/security/dependabot"},
		{"ghes", "https://api.github.example.com", "corp", "app", "https://github.example.com/corp/app/security/dependabot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.baseURL, "token", 0)
			got := client.AlertsURL(tt.owner, tt.repo)
			if got != tt.want {
				t.Errorf("AlertsURL() = %q, want %q", got, tt.want)
			}
		})
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

func TestSource_Fetch_Success(t *testing.T) {
	// Create a mock server
	alerts := AlertsResponse{
		{Number: 1, SecurityAdvisory: SecurityAdvisory{Severity: "high", GHSAID: "GHSA-1234"}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(alerts)
	}))
	defer server.Close()

	// Set environment variables for the test
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	s := &Source{}
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if report == nil {
		t.Fatal("expected non-nil report")
	}

	if report.Source != sourceName {
		t.Errorf("Source = %q, want %q", report.Source, sourceName)
	}

	// With 1 high severity alert: score should be 93
	// (100*40 + 85*30 + 100*20 + 100*10) / 100 = 95.5 -> 95
	if report.Score < 90 || report.Score > 100 {
		t.Errorf("Score = %d, expected between 90-100 for single high severity", report.Score)
	}
}

func TestSource_Fetch_MissingToken(t *testing.T) {
	// Ensure no token environment variables are set
	t.Setenv("DEPENDABOT_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	s := &Source{}
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error when token is missing")
	}
}

func TestSource_Fetch_InvalidProject(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")

	s := &Source{}
	opts := sources.Options{
		Project:   "invalid-format", // missing slash
		Threshold: 75,
	}

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error for invalid project format")
	}
}

func TestSource_Fetch_WithTitle(t *testing.T) {
	alerts := AlertsResponse{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(alerts)
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	s := &Source{}
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
		Title:     "Custom Title",
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if report.Title != "Custom Title" {
		t.Errorf("Title = %q, want %q", report.Title, "Custom Title")
	}
}
