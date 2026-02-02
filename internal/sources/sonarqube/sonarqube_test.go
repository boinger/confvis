package sonarqube

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/boinger/confvis/internal/sources"
)

func TestRatingToScore(t *testing.T) {
	tests := []struct {
		rating float64
		want   int
	}{
		{1.0, 100}, // A
		{2.0, 75},  // B
		{3.0, 50},  // C
		{4.0, 25},  // D
		{5.0, 0},   // E
		{1.5, 75},  // Between A and B rounds to B
		{2.5, 50},  // Between B and C rounds to C
		{0.5, 100}, // Below A still A
		{6.0, 0},   // Above E still E
	}

	for _, tt := range tests {
		got := RatingToScore(tt.rating)
		if got != tt.want {
			t.Errorf("RatingToScore(%v) = %d, want %d", tt.rating, got, tt.want)
		}
	}
}

func TestSource_Name(t *testing.T) {
	s := &Source{}
	if got := s.Name(); got != "sonarqube" {
		t.Errorf("Name() = %q, want %q", got, "sonarqube")
	}
}

func TestSource_Fetch_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/measures/component":
			// Verify project parameter
			if r.URL.Query().Get("component") != "myproject" {
				t.Errorf("expected component=myproject, got %s", r.URL.Query().Get("component"))
			}

			resp := MeasuresResponse{
				Component: ComponentMeasures{
					Key:  "myproject",
					Name: "My Project",
					Measures: []Measure{
						{Metric: "coverage", Value: "83.5"},
						{Metric: "reliability_rating", Value: "2.0"},   // B = 75
						{Metric: "security_rating", Value: "1.0"},     // A = 100
						{Metric: "sqale_rating", Value: "1.0"},        // A = 100
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Errorf("encoding response: %v", err)
			}

		case "/api/qualitygates/project_status":
			resp := QualityGateResponse{
				ProjectStatus: ProjectStatus{
					Status: "OK",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Errorf("encoding response: %v", err)
			}

		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	s := &Source{}
	ctx := context.Background()
	opts := sources.Options{
		URL:       server.URL,
		Project:   "myproject",
		Threshold: 75,
		Timeout:   5,
	}

	report, err := s.Fetch(ctx, opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Verify report
	if report.Title != "My Project" {
		t.Errorf("Title = %q, want %q", report.Title, "My Project")
	}
	if report.Source != "sonarqube" {
		t.Errorf("Source = %q, want %q", report.Source, "sonarqube")
	}
	if report.Threshold != 75 {
		t.Errorf("Threshold = %d, want %d", report.Threshold, 75)
	}
	if len(report.Factors) != 4 {
		t.Errorf("len(Factors) = %d, want %d", len(report.Factors), 4)
	}

	// Verify factors
	expectedFactors := map[string]int{
		"Test Coverage":   83,  // 83.5 truncated
		"Reliability":     75,  // B rating
		"Security":        100, // A rating
		"Maintainability": 100, // A rating
	}

	for _, f := range report.Factors {
		expected, ok := expectedFactors[f.Name]
		if !ok {
			t.Errorf("unexpected factor: %q", f.Name)
			continue
		}
		if f.Score != expected {
			t.Errorf("factor %q score = %d, want %d", f.Name, f.Score, expected)
		}
		if f.Weight != 25 {
			t.Errorf("factor %q weight = %d, want %d", f.Name, f.Weight, 25)
		}
		if f.URL == "" {
			t.Errorf("factor %q URL is empty", f.Name)
		}
	}

	// Verify weighted score: (83*25 + 75*25 + 100*25 + 100*25) / 100 = 89.5 → 90 (rounded)
	expectedScore := (83*25 + 75*25 + 100*25 + 100*25 + 50) / 100 // with rounding
	if report.Score != expectedScore {
		t.Errorf("Score = %d, want %d", report.Score, expectedScore)
	}
}

func TestSource_Fetch_CustomTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := MeasuresResponse{
			Component: ComponentMeasures{
				Key:      "myproject",
				Name:     "My Project",
				Measures: []Measure{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	}))
	defer server.Close()

	s := &Source{}
	opts := sources.Options{
		URL:     server.URL,
		Project: "myproject",
		Title:   "Custom Title",
		Timeout: 5,
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if report.Title != "Custom Title" {
		t.Errorf("Title = %q, want %q", report.Title, "Custom Title")
	}
}

func TestSource_Fetch_WithBranch(t *testing.T) {
	var receivedBranch string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBranch = r.URL.Query().Get("branch")
		resp := MeasuresResponse{
			Component: ComponentMeasures{
				Key:      "myproject",
				Name:     "My Project",
				Measures: []Measure{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	}))
	defer server.Close()

	s := &Source{}
	opts := sources.Options{
		URL:     server.URL,
		Project: "myproject",
		Branch:  "feature/test",
		Timeout: 5,
	}

	_, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if receivedBranch != "feature/test" {
		t.Errorf("branch = %q, want %q", receivedBranch, "feature/test")
	}
}

func TestSource_Fetch_WithToken(t *testing.T) {
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		resp := MeasuresResponse{
			Component: ComponentMeasures{
				Key:      "myproject",
				Name:     "My Project",
				Measures: []Measure{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	}))
	defer server.Close()

	s := &Source{}
	opts := sources.Options{
		URL:     server.URL,
		Project: "myproject",
		Token:   "mytoken",
		Timeout: 5,
	}

	_, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// SonarQube uses Basic auth with token as username
	if receivedAuth == "" {
		t.Error("expected Authorization header to be set")
	}
}

func TestSource_Fetch_MissingURL(t *testing.T) {
	s := &Source{}
	opts := sources.Options{
		Project: "myproject",
		Timeout: 5,
	}

	// Clear env var if set
	t.Setenv(EnvURL, "")

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error for missing URL")
	}
}

func TestSource_Fetch_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Project not found"}]}`, http.StatusNotFound)
	}))
	defer server.Close()

	s := &Source{}
	opts := sources.Options{
		URL:     server.URL,
		Project: "nonexistent",
		Timeout: 5,
	}

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestSource_Fetch_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay to allow context cancellation
		time.Sleep(100 * time.Millisecond)
		resp := MeasuresResponse{
			Component: ComponentMeasures{
				Key:      "myproject",
				Name:     "My Project",
				Measures: []Measure{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			// Expected if context is canceled
			return
		}
	}))
	defer server.Close()

	s := &Source{}
	opts := sources.Options{
		URL:     server.URL,
		Project: "myproject",
		Timeout: 5,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := s.Fetch(ctx, opts)
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestClient_ProjectURL(t *testing.T) {
	c := NewClient("https://sonar.example.com", "", 5*time.Second)

	tests := []struct {
		project string
		branch  string
		want    string
	}{
		{
			project: "myproject",
			branch:  "",
			want:    "https://sonar.example.com/dashboard?id=myproject",
		},
		{
			project: "myproject",
			branch:  "main",
			want:    "https://sonar.example.com/dashboard?id=myproject&branch=main",
		},
		{
			project: "my/project",
			branch:  "feature/test",
			want:    "https://sonar.example.com/dashboard?id=my%2Fproject&branch=feature%2Ftest",
		},
	}

	for _, tt := range tests {
		got := c.ProjectURL(tt.project, tt.branch)
		if got != tt.want {
			t.Errorf("ProjectURL(%q, %q) = %q, want %q", tt.project, tt.branch, got, tt.want)
		}
	}
}

func TestClient_MeasureURL(t *testing.T) {
	c := NewClient("https://sonar.example.com", "", 5*time.Second)

	got := c.MeasureURL("myproject", "coverage", "main")
	want := "https://sonar.example.com/component_measures?id=myproject&metric=coverage&branch=main"
	if got != want {
		t.Errorf("MeasureURL() = %q, want %q", got, want)
	}
}

func TestSource_Registration(t *testing.T) {
	// Verify source is registered
	s := sources.Get("sonarqube")
	if s == nil {
		t.Error("sonarqube source not registered")
	}
}

func TestClient_FetchQualityGate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify path
		if r.URL.Path != "/api/qualitygates/project_status" {
			t.Errorf("path = %q, want /api/qualitygates/project_status", r.URL.Path)
		}

		// Verify project key
		if r.URL.Query().Get("projectKey") != "myproject" {
			t.Errorf("projectKey = %q, want myproject", r.URL.Query().Get("projectKey"))
		}

		resp := QualityGateResponse{
			ProjectStatus: ProjectStatus{
				Status: "OK",
			},
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

	qg, err := client.FetchQualityGate(context.Background(), "myproject", "")
	if err != nil {
		t.Fatalf("FetchQualityGate() error = %v", err)
	}

	if qg.ProjectStatus.Status != "OK" {
		t.Errorf("status = %q, want OK", qg.ProjectStatus.Status)
	}
}

func TestClient_FetchQualityGate_WithBranch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify branch parameter
		if r.URL.Query().Get("branch") != "main" {
			t.Errorf("branch = %q, want main", r.URL.Query().Get("branch"))
		}

		resp := QualityGateResponse{
			ProjectStatus: ProjectStatus{
				Status: "ERROR",
			},
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

	qg, err := client.FetchQualityGate(context.Background(), "myproject", "main")
	if err != nil {
		t.Fatalf("FetchQualityGate() error = %v", err)
	}

	if qg.ProjectStatus.Status != "ERROR" {
		t.Errorf("status = %q, want ERROR", qg.ProjectStatus.Status)
	}
}

func TestClient_FetchQualityGate_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":[{"msg":"Not found"}]}`, http.StatusNotFound)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

	_, err := client.FetchQualityGate(context.Background(), "nonexistent", "")
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestClient_doRequest_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("not valid json")); err != nil {
			t.Errorf("writing response: %v", err)
		}
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

	_, err := client.FetchMeasures(context.Background(), "myproject", "")
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	client := NewClient("https://sonar.example.com/", "token", 5*time.Second)
	if client.baseURL != "https://sonar.example.com" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://sonar.example.com")
	}
}

func TestSource_Fetch_DefaultTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := MeasuresResponse{
			Component: ComponentMeasures{
				Key:      "myproject",
				Name:     "My Project",
				Measures: []Measure{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	}))
	defer server.Close()

	s := &Source{}
	opts := sources.Options{
		URL:     server.URL,
		Project: "myproject",
		Timeout: 0, // Should use default 30s
	}

	_, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
}

func TestSource_Fetch_EmptyComponentName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := MeasuresResponse{
			Component: ComponentMeasures{
				Key:      "myproject",
				Name:     "", // Empty name, should fall back to project
				Measures: []Measure{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	}))
	defer server.Close()

	s := &Source{}
	opts := sources.Options{
		URL:     server.URL,
		Project: "myproject",
		Timeout: 5,
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Should fall back to project name
	if report.Title != "myproject" {
		t.Errorf("Title = %q, want %q", report.Title, "myproject")
	}
}

func TestSource_Fetch_TokenFromEnv(t *testing.T) {
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		resp := MeasuresResponse{
			Component: ComponentMeasures{
				Key:      "myproject",
				Name:     "My Project",
				Measures: []Measure{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	}))
	defer server.Close()

	t.Setenv(EnvToken, "env-token")

	s := &Source{}
	opts := sources.Options{
		URL:     server.URL,
		Project: "myproject",
		Timeout: 5,
	}

	_, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Should have used env token
	if receivedAuth == "" {
		t.Error("expected Authorization header from env token")
	}
}

func TestSource_Fetch_URLFromEnv(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := MeasuresResponse{
			Component: ComponentMeasures{
				Key:      "myproject",
				Name:     "My Project",
				Measures: []Measure{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encoding response: %v", err)
		}
	}))
	defer server.Close()

	t.Setenv(EnvURL, server.URL)

	s := &Source{}
	opts := sources.Options{
		Project: "myproject",
		Timeout: 5,
	}

	_, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
}
