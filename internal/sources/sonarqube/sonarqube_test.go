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

func TestCountToScore(t *testing.T) {
	tests := []struct {
		count int
		want  int
	}{
		{0, 100},  // Perfect
		{1, 80},   // 1-5 issues
		{5, 80},   // Upper bound of first tier
		{6, 60},   // 6-10 issues
		{10, 60},  // Upper bound of second tier
		{11, 40},  // 11-25 issues
		{25, 40},  // Upper bound of third tier
		{26, 20},  // 26-50 issues
		{50, 20},  // Upper bound of fourth tier
		{51, 0},   // 51+ issues
		{100, 0},  // Many issues
		{1000, 0}, // Very many issues
	}

	for _, tt := range tests {
		got := CountToScore(tt.count)
		if got != tt.want {
			t.Errorf("CountToScore(%d) = %d, want %d", tt.count, got, tt.want)
		}
	}
}

func TestDuplicationToScore(t *testing.T) {
	tests := []struct {
		pct  float64
		want int
	}{
		{0.0, 100},  // No duplication
		{10.0, 90},  // 10% duplication
		{25.5, 75},  // 25.5% duplication → int(25.5) = 25 → 100 - 25 = 75
		{50.0, 50},  // 50% duplication
		{75.0, 25},  // 75% duplication
		{100.0, 0},  // Full duplication
		{110.0, 0},  // Over 100% (edge case, clamped to 0)
	}

	for _, tt := range tests {
		got := DuplicationToScore(tt.pct)
		if got != tt.want {
			t.Errorf("DuplicationToScore(%v) = %d, want %d", tt.pct, got, tt.want)
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
						{Metric: "reliability_rating", Value: "2.0"},          // B = 75
						{Metric: "security_rating", Value: "1.0"},             // A = 100
						{Metric: "sqale_rating", Value: "1.0"},                // A = 100
						{Metric: "vulnerabilities", Value: "0"},               // 0 = 100
						{Metric: "bugs", Value: "3"},                          // 1-5 = 80
						{Metric: "code_smells", Value: "12"},                  // 11-25 = 40
						{Metric: "duplicated_lines_density", Value: "5.2"},    // 100 - 5 = 94
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
	if len(report.Factors) != 8 {
		t.Errorf("len(Factors) = %d, want %d", len(report.Factors), 8)
	}

	// Verify factors with their expected scores and weights
	expectedFactors := map[string]struct {
		score  int
		weight int
	}{
		"Test Coverage":   {83, 20},  // 83.5 truncated, weight 20
		"Reliability":     {75, 20},  // B rating, weight 20
		"Security":        {100, 20}, // A rating, weight 20
		"Maintainability": {100, 20}, // A rating, weight 20
		"Vulnerabilities": {100, 10}, // 0 issues = 100, weight 10
		"Bugs":            {80, 10},  // 3 issues (1-5) = 80, weight 10
		"Code Smells":     {40, 5},   // 12 issues (11-25) = 40, weight 5
		"Duplication":     {95, 5},   // 100 - int(5.2) = 100 - 5 = 95, weight 5
	}

	for _, f := range report.Factors {
		expected, ok := expectedFactors[f.Name]
		if !ok {
			t.Errorf("unexpected factor: %q", f.Name)
			continue
		}
		if f.Score != expected.score {
			t.Errorf("factor %q score = %d, want %d", f.Name, f.Score, expected.score)
		}
		if f.Weight != expected.weight {
			t.Errorf("factor %q weight = %d, want %d", f.Name, f.Weight, expected.weight)
		}
		if f.URL == "" {
			t.Errorf("factor %q URL is empty", f.Name)
		}
	}

	// Verify weighted score calculation
	// (83*20 + 75*20 + 100*20 + 100*20 + 100*10 + 80*10 + 40*5 + 95*5) / 110
	// = (1660 + 1500 + 2000 + 2000 + 1000 + 800 + 200 + 475) / 110
	// = 9635 / 110 = 87.59... → 88 (rounded)
	totalWeight := 20 + 20 + 20 + 20 + 10 + 10 + 5 + 5 // 110
	weightedSum := 83*20 + 75*20 + 100*20 + 100*20 + 100*10 + 80*10 + 40*5 + 95*5
	expectedScore := (weightedSum + totalWeight/2) / totalWeight // with rounding
	if report.ScoreValue() != expectedScore {
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

	client := NewClientWithHTTP(server.URL, "test-token", server.Client())

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

	client := NewClientWithHTTP(server.URL, "test-token", server.Client())

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

	client := NewClientWithHTTP(server.URL, "test-token", server.Client())

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

	client := NewClientWithHTTP(server.URL, "test-token", server.Client())

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

func TestConvertMetricValue(t *testing.T) {
	tests := []struct {
		name    string
		val     string
		kind    metricKind
		want    int
		wantErr bool
	}{
		{"percentage 85.5", "85.5", metricKindPercentage, 85, false},
		{"percentage 0", "0", metricKindPercentage, 0, false},
		{"percentage 100", "100", metricKindPercentage, 100, false},
		{"percentage invalid", "abc", metricKindPercentage, 0, true},
		{"rating A", "1.0", metricKindRating, RatingToScore(1.0), false},
		{"rating E", "5.0", metricKindRating, RatingToScore(5.0), false},
		{"rating invalid", "xyz", metricKindRating, 0, true},
		{"count 0", "0", metricKindCount, CountToScore(0), false},
		{"count 5", "5", metricKindCount, CountToScore(5), false},
		{"count invalid", "abc", metricKindCount, 0, true},
		{"duplication 3.5", "3.5", metricKindDuplication, DuplicationToScore(3.5), false},
		{"duplication invalid", "abc", metricKindDuplication, 0, true},
		{"unknown kind", "10", metricKind(99), 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertMetricValue(tt.val, tt.kind)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("convertMetricValue() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("convertMetricValue(%q, %d) = %d, want %d", tt.val, tt.kind, got, tt.want)
			}
		})
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
