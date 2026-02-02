package ghactions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/boinger/confvis/internal/sources"
)

func TestConclusionScore(t *testing.T) {
	tests := []struct {
		conclusion string
		want       int
	}{
		{"success", 100},
		{"neutral", 75},
		{"skipped", 75},
		{"cancelled", 50},
		{"failure", 0},
		{"timed_out", 0},
		{"unknown", 0},
		{"", 0},
	}

	for _, tt := range tests {
		got := ConclusionScore(tt.conclusion)
		if got != tt.want {
			t.Errorf("ConclusionScore(%q) = %d, want %d", tt.conclusion, got, tt.want)
		}
	}
}

func TestSource_Name(t *testing.T) {
	s := &Source{}
	if got := s.Name(); got != "github-actions" {
		t.Errorf("Name() = %q, want %q", got, "github-actions")
	}
}

func TestSource_Fetch_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify path
		expectedPath := "/repos/myorg/myrepo/actions/runs"
		if r.URL.Path != expectedPath {
			t.Errorf("path = %q, want %q", r.URL.Path, expectedPath)
		}

		// Verify auth header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test-token")
		}

		// Verify GitHub API headers
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("Accept header missing or incorrect")
		}
		if r.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
			t.Errorf("X-GitHub-Api-Version header missing or incorrect")
		}

		// Verify query params
		if r.URL.Query().Get("status") != "completed" {
			t.Errorf("expected status=completed query param")
		}

		resp := WorkflowRunsResponse{
			TotalCount: 5,
			WorkflowRuns: []WorkflowRun{
				{ID: 1, Conclusion: "success"},
				{ID: 2, Conclusion: "success"},
				{ID: 3, Conclusion: "success"},
				{ID: 4, Conclusion: "failure"},
				{ID: 5, Conclusion: "success"},
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

	ctx := context.Background()
	runsResp, err := client.FetchRuns(ctx, "myorg/myrepo", FetchRunsOptions{Count: 20})
	if err != nil {
		t.Fatalf("FetchRuns() error = %v", err)
	}

	if len(runsResp.WorkflowRuns) != 5 {
		t.Errorf("got %d runs, want 5", len(runsResp.WorkflowRuns))
	}

	// Calculate expected success rate: 4/5 = 80%
	successCount := 0
	for _, run := range runsResp.WorkflowRuns {
		if run.Conclusion == "success" {
			successCount++
		}
	}
	successRate := (successCount * 100) / len(runsResp.WorkflowRuns)
	if successRate != 80 {
		t.Errorf("success rate = %d, want 80", successRate)
	}
}

func TestSource_Fetch_WithWorkflowFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify workflow-specific path
		expectedPath := "/repos/myorg/myrepo/actions/workflows/ci.yml/runs"
		if r.URL.Path != expectedPath {
			t.Errorf("path = %q, want %q", r.URL.Path, expectedPath)
		}

		resp := WorkflowRunsResponse{
			TotalCount:   1,
			WorkflowRuns: []WorkflowRun{{ID: 1, Conclusion: "success"}},
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

	_, err := client.FetchRuns(context.Background(), "myorg/myrepo", FetchRunsOptions{
		Workflow: "ci.yml",
		Count:    20,
	})
	if err != nil {
		t.Fatalf("FetchRuns() error = %v", err)
	}
}

func TestSource_Fetch_WithEventFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify event query param
		if r.URL.Query().Get("event") != "push" {
			t.Errorf("event = %q, want %q", r.URL.Query().Get("event"), "push")
		}

		resp := WorkflowRunsResponse{
			TotalCount:   1,
			WorkflowRuns: []WorkflowRun{{ID: 1, Conclusion: "success"}},
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

	_, err := client.FetchRuns(context.Background(), "myorg/myrepo", FetchRunsOptions{
		Event: "push",
		Count: 20,
	})
	if err != nil {
		t.Fatalf("FetchRuns() error = %v", err)
	}
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
	client := NewClient("", "test-token", 5*time.Second)

	_, err := client.FetchRuns(context.Background(), "invalid-no-slash", FetchRunsOptions{Count: 20})
	if err == nil {
		t.Error("expected error for invalid project format")
	}
}

func TestSource_Fetch_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message": "Not Found"}`, http.StatusNotFound)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL,
		token:      "test-token",
		httpClient: server.Client(),
	}

	_, err := client.FetchRuns(context.Background(), "myorg/myrepo", FetchRunsOptions{Count: 20})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestSource_Fetch_NoRuns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := WorkflowRunsResponse{
			TotalCount:   0,
			WorkflowRuns: []WorkflowRun{},
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

	runsResp, err := client.FetchRuns(context.Background(), "myorg/myrepo", FetchRunsOptions{Count: 20})
	if err != nil {
		t.Fatalf("FetchRuns() error = %v", err)
	}

	// 0 runs should result in 0 success rate (avoiding division by zero)
	if len(runsResp.WorkflowRuns) != 0 {
		t.Errorf("expected 0 runs, got %d", len(runsResp.WorkflowRuns))
	}
}

func TestClient_ActionsURL(t *testing.T) {
	tests := []struct {
		baseURL   string
		ownerRepo string
		want      string
	}{
		{"", "myorg/myrepo", "https://github.com/myorg/myrepo/actions"},
		{"https://api.github.com", "myorg/myrepo", "https://github.com/myorg/myrepo/actions"},
		{"https://api.github.example.com", "myorg/myrepo", "https://github.example.com/myorg/myrepo/actions"},
		{"https://github.example.com/api/v3", "myorg/myrepo", "https://github.example.com/myorg/myrepo/actions"},
		{"", "invalid", ""},
	}

	for _, tt := range tests {
		client := NewClient(tt.baseURL, "token", 5*time.Second)
		got := client.ActionsURL(tt.ownerRepo)
		if got != tt.want {
			t.Errorf("ActionsURL(%q) with baseURL=%q = %q, want %q", tt.ownerRepo, tt.baseURL, got, tt.want)
		}
	}
}

func TestSource_Registration(t *testing.T) {
	s := sources.Get("github-actions")
	if s == nil {
		t.Error("github-actions source not registered")
	}
}

func TestSource_Fetch_AllFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := WorkflowRunsResponse{
			TotalCount: 3,
			WorkflowRuns: []WorkflowRun{
				{ID: 1, Conclusion: "failure"},
				{ID: 2, Conclusion: "failure"},
				{ID: 3, Conclusion: "timed_out"},
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

	runsResp, err := client.FetchRuns(context.Background(), "myorg/myrepo", FetchRunsOptions{Count: 20})
	if err != nil {
		t.Fatalf("FetchRuns() error = %v", err)
	}

	// All failures = 0% success rate
	successCount := 0
	for _, run := range runsResp.WorkflowRuns {
		if run.Conclusion == "success" {
			successCount++
		}
	}
	if successCount != 0 {
		t.Errorf("expected 0 successful runs, got %d", successCount)
	}
}

func TestSource_Fetch_MixedConclusions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := WorkflowRunsResponse{
			TotalCount: 10,
			WorkflowRuns: []WorkflowRun{
				{ID: 1, Conclusion: "success"},
				{ID: 2, Conclusion: "success"},
				{ID: 3, Conclusion: "neutral"},
				{ID: 4, Conclusion: "skipped"},
				{ID: 5, Conclusion: "cancelled"},
				{ID: 6, Conclusion: "failure"},
				{ID: 7, Conclusion: "success"},
				{ID: 8, Conclusion: "timed_out"},
				{ID: 9, Conclusion: "success"},
				{ID: 10, Conclusion: "success"},
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

	runsResp, err := client.FetchRuns(context.Background(), "myorg/myrepo", FetchRunsOptions{Count: 20})
	if err != nil {
		t.Fatalf("FetchRuns() error = %v", err)
	}

	// Count successes: 5 out of 10 = 50%
	successCount := 0
	for _, run := range runsResp.WorkflowRuns {
		if run.Conclusion == "success" {
			successCount++
		}
	}
	if successCount != 5 {
		t.Errorf("expected 5 successful runs, got %d", successCount)
	}
}
