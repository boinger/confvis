package githubalerts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/scoring"
)

func TestNewSource(t *testing.T) {
	cfg := SourceConfig{
		Name:         "test-alerts",
		TokenEnvVar:  "TEST_TOKEN",
		EndpointPath: "test/alerts",
		WebURLPath:   "security/test",
	}
	counter := func(data []byte) (scoring.SeverityCounts, error) {
		return scoring.SeverityCounts{}, nil
	}

	s := NewSource(cfg, counter, nil)

	if s.Name() != "test-alerts" {
		t.Errorf("Name() = %q, want %q", s.Name(), "test-alerts")
	}
}

func TestAlertsSource_Fetch(t *testing.T) {
	alerts := []map[string]interface{}{
		{"number": 1, "severity": "high"},
		{"number": 2, "severity": "medium"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != "open" {
			t.Errorf("expected state=open, got %s", r.URL.Query().Get("state"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(alerts)
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	cfg := SourceConfig{
		Name:         "test-alerts",
		TokenEnvVar:  "TEST_ALERTS_TOKEN",
		EndpointPath: "test/alerts",
		WebURLPath:   "security/test",
	}
	counter := func(data []byte) (scoring.SeverityCounts, error) {
		var alerts []map[string]interface{}
		if err := json.Unmarshal(data, &alerts); err != nil {
			return scoring.SeverityCounts{}, err
		}
		var counts scoring.SeverityCounts
		for _, a := range alerts {
			switch a["severity"] {
			case "critical":
				counts.Critical++
			case "high":
				counts.High++
			case "medium":
				counts.Medium++
			case "low":
				counts.Low++
			}
		}
		return counts, nil
	}

	s := NewSource(cfg, counter, nil)

	opts := sources.Options{
		Project:   "owner/repo",
		Threshold: 75,
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if report.Source != "test-alerts" {
		t.Errorf("Source = %q, want %q", report.Source, "test-alerts")
	}

	if len(report.Factors) != 4 {
		t.Errorf("len(Factors) = %d, want 4", len(report.Factors))
	}
}

func TestAlertsSource_Fetch_MissingToken(t *testing.T) {
	t.Setenv("TEST_ALERTS_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")

	cfg := SourceConfig{
		Name:         "test-alerts",
		TokenEnvVar:  "TEST_ALERTS_TOKEN",
		EndpointPath: "test/alerts",
		WebURLPath:   "security/test",
	}
	counter := func(data []byte) (scoring.SeverityCounts, error) {
		return scoring.SeverityCounts{}, nil
	}

	s := NewSource(cfg, counter, nil)

	opts := sources.Options{
		Project: "owner/repo",
	}

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error when token is missing")
	}
}

func TestAlertsSource_Fetch_InvalidProject(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")

	cfg := SourceConfig{
		Name:         "test-alerts",
		TokenEnvVar:  "TEST_ALERTS_TOKEN",
		EndpointPath: "test/alerts",
		WebURLPath:   "security/test",
	}
	counter := func(data []byte) (scoring.SeverityCounts, error) {
		return scoring.SeverityCounts{}, nil
	}

	s := NewSource(cfg, counter, nil)

	opts := sources.Options{
		Project: "invalid-no-slash",
	}

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error for invalid project format")
	}
}

func TestAlertsSource_Fetch_Pagination(t *testing.T) {
	page1Alerts := make([]map[string]interface{}, 100)
	for i := range page1Alerts {
		page1Alerts[i] = map[string]interface{}{"number": i + 1, "severity": "medium"}
	}
	page2Alerts := make([]map[string]interface{}, 50)
	for i := range page2Alerts {
		page2Alerts[i] = map[string]interface{}{"number": 101 + i, "severity": "low"}
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
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
		}
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	cfg := SourceConfig{
		Name:         "test-alerts",
		TokenEnvVar:  "TEST_ALERTS_TOKEN",
		EndpointPath: "test/alerts",
		WebURLPath:   "security/test",
	}
	counter := func(data []byte) (scoring.SeverityCounts, error) {
		var alerts []map[string]interface{}
		if err := json.Unmarshal(data, &alerts); err != nil {
			return scoring.SeverityCounts{}, err
		}
		var counts scoring.SeverityCounts
		counts.Medium = len(alerts) // Simplified for test
		return counts, nil
	}

	s := NewSource(cfg, counter, nil)

	opts := sources.Options{
		Project: "owner/repo",
	}

	_, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if requestCount != 2 {
		t.Errorf("requestCount = %d, want 2", requestCount)
	}
}

func TestAlertsSource_Fetch_WithExtraParams(t *testing.T) {
	var receivedToolName string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToolName = r.URL.Query().Get("tool_name")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	cfg := SourceConfig{
		Name:         "test-alerts",
		TokenEnvVar:  "TEST_ALERTS_TOKEN",
		EndpointPath: "test/alerts",
		WebURLPath:   "security/test",
	}
	counter := func(data []byte) (scoring.SeverityCounts, error) {
		return scoring.SeverityCounts{}, nil
	}
	extraParams := func(opts sources.Options) url.Values {
		toolName := sources.GetExtra(opts, "tool", "")
		if toolName == "" {
			return nil
		}
		return url.Values{"tool_name": {toolName}}
	}

	s := NewSource(cfg, counter, extraParams)

	opts := sources.Options{
		Project: "owner/repo",
		Extra:   map[string]string{"tool": "TestTool"},
	}

	_, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if receivedToolName != "TestTool" {
		t.Errorf("tool_name = %q, want %q", receivedToolName, "TestTool")
	}
}

func TestAlertsSource_Fetch_CounterError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	cfg := SourceConfig{
		Name:         "test-alerts",
		TokenEnvVar:  "TEST_ALERTS_TOKEN",
		EndpointPath: "test/alerts",
		WebURLPath:   "security/test",
	}
	counter := func(_ []byte) (scoring.SeverityCounts, error) {
		var dummy interface{}
		return scoring.SeverityCounts{}, json.Unmarshal([]byte("invalid json"), &dummy)
	}

	s := NewSource(cfg, counter, nil)

	opts := sources.Options{
		Project: "owner/repo",
	}

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error from counter")
	}
}

func TestAlertsSource_Fetch_WithTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	cfg := SourceConfig{
		Name:         "test-alerts",
		TokenEnvVar:  "TEST_ALERTS_TOKEN",
		EndpointPath: "test/alerts",
		WebURLPath:   "security/test",
	}
	counter := func(data []byte) (scoring.SeverityCounts, error) {
		return scoring.SeverityCounts{}, nil
	}

	s := NewSource(cfg, counter, nil)

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

func TestAlertsSource_Fetch_DefaultTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer server.Close()

	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	cfg := SourceConfig{
		Name:         "test-alerts",
		TokenEnvVar:  "TEST_ALERTS_TOKEN",
		EndpointPath: "test/alerts",
		WebURLPath:   "security/test",
	}
	counter := func(data []byte) (scoring.SeverityCounts, error) {
		return scoring.SeverityCounts{}, nil
	}

	s := NewSource(cfg, counter, nil)

	opts := sources.Options{
		Project: "myorg/myrepo",
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if report.Title != "myorg/myrepo" {
		t.Errorf("Title = %q, want %q", report.Title, "myorg/myrepo")
	}
}
