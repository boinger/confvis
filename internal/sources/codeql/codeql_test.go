package codeql

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/scoring"
)

func TestCountAlerts(t *testing.T) {
	alerts := AlertsResponse{
		{Number: 1, Rule: Rule{SecuritySeverityLevel: "critical"}},
		{Number: 2, Rule: Rule{SecuritySeverityLevel: "critical"}},
		{Number: 3, Rule: Rule{SecuritySeverityLevel: "high"}},
		{Number: 4, Rule: Rule{SecuritySeverityLevel: "medium"}},
		{Number: 5, Rule: Rule{SecuritySeverityLevel: "medium"}},
		{Number: 6, Rule: Rule{SecuritySeverityLevel: "medium"}},
		{Number: 7, Rule: Rule{SecuritySeverityLevel: "low"}},
	}

	data, err := json.Marshal(alerts)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	counts, err := countAlerts(data)
	if err != nil {
		t.Fatalf("countAlerts() error = %v", err)
	}

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

func TestCountAlerts_FallbackToRuleSeverity(t *testing.T) {
	// When security_severity_level is not set, fall back to rule severity
	alerts := AlertsResponse{
		{Number: 1, Rule: Rule{Severity: "error"}},   // -> high
		{Number: 2, Rule: Rule{Severity: "warning"}}, // -> medium
		{Number: 3, Rule: Rule{Severity: "note"}},    // -> low
		{Number: 4, Rule: Rule{Severity: "none"}},    // -> low
	}

	data, err := json.Marshal(alerts)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	counts, err := countAlerts(data)
	if err != nil {
		t.Fatalf("countAlerts() error = %v", err)
	}

	if counts.Critical != 0 {
		t.Errorf("Critical = %d, want 0", counts.Critical)
	}
	if counts.High != 1 {
		t.Errorf("High = %d, want 1", counts.High)
	}
	if counts.Medium != 1 {
		t.Errorf("Medium = %d, want 1", counts.Medium)
	}
	if counts.Low != 2 {
		t.Errorf("Low = %d, want 2", counts.Low)
	}
}

func TestCountAlerts_PreferSecuritySeverityLevel(t *testing.T) {
	// security_severity_level takes precedence over rule severity
	alerts := AlertsResponse{
		{Number: 1, Rule: Rule{SecuritySeverityLevel: "critical", Severity: "note"}},
	}

	data, err := json.Marshal(alerts)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	counts, err := countAlerts(data)
	if err != nil {
		t.Fatalf("countAlerts() error = %v", err)
	}

	if counts.Critical != 1 {
		t.Errorf("Critical = %d, want 1", counts.Critical)
	}
	if counts.Low != 0 {
		t.Errorf("Low = %d, want 0 (should use security_severity_level)", counts.Low)
	}
}

func TestCountAlerts_InvalidJSON(t *testing.T) {
	_, err := countAlerts([]byte("invalid json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestCountAlerts_EmptyAlerts(t *testing.T) {
	data, err := json.Marshal(AlertsResponse{})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	counts, err := countAlerts(data)
	if err != nil {
		t.Fatalf("countAlerts() error = %v", err)
	}
	if counts != (scoring.SeverityCounts{}) {
		t.Errorf("expected zero counts, got %+v", counts)
	}
}

func TestExtraParams_WithToolName(t *testing.T) {
	opts := sources.Options{
		Extra: map[string]string{"tool": "CodeQL"},
	}
	params := extraParams(opts)
	if params == nil {
		t.Fatal("expected non-nil params")
	}
	if params.Get("tool_name") != "CodeQL" {
		t.Errorf("tool_name = %q, want %q", params.Get("tool_name"), "CodeQL")
	}
}

func TestExtraParams_NoToolName(t *testing.T) {
	opts := sources.Options{}
	params := extraParams(opts)
	if params != nil {
		t.Errorf("expected nil params, got %v", params)
	}
}

func TestExtraParams_EmptyToolName(t *testing.T) {
	opts := sources.Options{
		Extra: map[string]string{"tool": ""},
	}
	params := extraParams(opts)
	if params != nil {
		t.Errorf("expected nil params for empty tool, got %v", params)
	}
}

func TestSource_Registration(t *testing.T) {
	s := sources.Get("codeql")
	if s == nil {
		t.Error("codeql source not registered")
	}
	if s.Name() != "codeql" {
		t.Errorf("Name() = %q, want %q", s.Name(), "codeql")
	}
}

func TestSource_Fetch_Success(t *testing.T) {
	alerts := AlertsResponse{
		{Number: 1, Rule: Rule{SecuritySeverityLevel: "high", ID: "go/sql-injection"}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/code-scanning/alerts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("state") != "open" {
			t.Errorf("expected state=open query param")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(alerts)
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	s := sources.Get("codeql")
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

	if report.Source != "codeql" {
		t.Errorf("Source = %q, want %q", report.Source, "codeql")
	}

	// With 1 high severity alert: score should be 95
	// (100*40 + 85*30 + 100*20 + 100*10) / 100 = 95.5 -> 95
	if report.ScoreValue() < 90 || report.ScoreValue() > 100 {
		t.Errorf("Score = %d, expected between 90-100 for single high severity", report.Score)
	}
}

func TestSource_Fetch_NoAlerts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AlertsResponse{})
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	s := sources.Get("codeql")
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if report.ScoreValue() != 100 {
		t.Errorf("Score = %d, want 100 for no alerts", report.ScoreValue())
	}
}

func TestSource_Fetch_MixedSeverities(t *testing.T) {
	alerts := AlertsResponse{
		{Number: 1, Rule: Rule{SecuritySeverityLevel: "critical"}},
		{Number: 2, Rule: Rule{SecuritySeverityLevel: "high"}},
		{Number: 3, Rule: Rule{SecuritySeverityLevel: "medium"}},
		{Number: 4, Rule: Rule{SecuritySeverityLevel: "low"}},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(alerts)
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	s := sources.Get("codeql")
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// (75*40 + 85*30 + 95*20 + 98*10) / 100 = 84.3 -> 84
	if report.ScoreValue() != 84 {
		t.Errorf("Score = %d, want 84", report.ScoreValue())
	}

	if len(report.Factors) != 4 {
		t.Errorf("len(Factors) = %d, want 4", len(report.Factors))
	}
}

func TestSource_Fetch_WithToolFilter(t *testing.T) {
	var receivedToolName string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToolName = r.URL.Query().Get("tool_name")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AlertsResponse{})
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	s := sources.Get("codeql")
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
		Extra:     map[string]string{"tool": "CodeQL"},
	}

	_, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if receivedToolName != "CodeQL" {
		t.Errorf("tool_name = %q, want %q", receivedToolName, "CodeQL")
	}
}

func TestSource_Fetch_Pagination(t *testing.T) {
	// Build 100 alerts for page 1, 50 for page 2
	page1Alerts := make(AlertsResponse, 100)
	for i := range page1Alerts {
		page1Alerts[i] = Alert{Number: i + 1, Rule: Rule{SecuritySeverityLevel: "medium"}}
	}
	page2Alerts := make(AlertsResponse, 50)
	for i := range page2Alerts {
		page2Alerts[i] = Alert{Number: 101 + i, Rule: Rule{SecuritySeverityLevel: "low"}}
	}

	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case "1", "":
			_ = json.NewEncoder(w).Encode(page1Alerts)
		case "2":
			_ = json.NewEncoder(w).Encode(page2Alerts)
		default:
			_ = json.NewEncoder(w).Encode(AlertsResponse{})
		}
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	s := sources.Get("codeql")
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	_, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if requestCount != 2 {
		t.Errorf("requestCount = %d, want 2", requestCount)
	}
}

func TestSource_Fetch_MissingToken(t *testing.T) {
	t.Setenv("CODEQL_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	s := sources.Get("codeql")
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

	s := sources.Get("codeql")
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AlertsResponse{})
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	s := sources.Get("codeql")
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

func TestSource_Fetch_DefaultTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AlertsResponse{})
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	s := sources.Get("codeql")
	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if report.Title != "owner/repo" {
		t.Errorf("Title = %q, want %q", report.Title, "owner/repo")
	}
}

func TestSource_Fetch_FactorURLs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AlertsResponse{})
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	s := sources.Get("codeql")
	opts := sources.Options{
		Project:   "myorg/myrepo",
		Threshold: 75,
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if len(report.Factors) == 0 {
		t.Fatal("expected at least one factor")
	}

	// All factors should have URLs containing the security path
	// (URL host depends on test server, but path is consistent)
	expectedPath := "/myorg/myrepo/security/code-scanning"
	for _, f := range report.Factors {
		if f.URL == "" {
			t.Errorf("Factor URL is empty")
		}
		if !strings.Contains(f.URL, expectedPath) {
			t.Errorf("Factor URL = %q, want to contain %q", f.URL, expectedPath)
		}
	}
}

// Silence unused import warning - url is used in extraParams
var _ = url.Values{}
