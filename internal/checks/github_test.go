package checks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/boinger/confvis/internal/confidence"
)

func TestNewGitHubClient(t *testing.T) {
	tests := []struct {
		name        string
		cfg         GitHubClientConfig
		wantBaseURL string
	}{
		{
			name:        "default base URL",
			cfg:         GitHubClientConfig{Token: "test-token"},
			wantBaseURL: defaultGitHubAPIURL,
		},
		{
			name:        "custom base URL",
			cfg:         GitHubClientConfig{BaseURL: "https://github.example.com/api/v3", Token: "test-token"},
			wantBaseURL: "https://github.example.com/api/v3",
		},
		{
			name:        "strips trailing slash",
			cfg:         GitHubClientConfig{BaseURL: "https://github.example.com/api/v3/", Token: "test-token"},
			wantBaseURL: "https://github.example.com/api/v3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewGitHubClient(tt.cfg)
			if client.baseURL != tt.wantBaseURL {
				t.Errorf("baseURL = %q, want %q", client.baseURL, tt.wantBaseURL)
			}
			if client.token != tt.cfg.Token {
				t.Errorf("token = %q, want %q", client.token, tt.cfg.Token)
			}
			if client.httpClient == nil {
				t.Error("httpClient is nil")
			}
		})
	}
}

func TestNewGitHubClientWithHTTP(t *testing.T) {
	customClient := &http.Client{}
	client := NewGitHubClientWithHTTP(
		GitHubClientConfig{Token: "test-token"},
		customClient,
	)

	if client.httpClient != customClient {
		t.Error("httpClient was not set to custom client")
	}
}

func TestGitHubClient_CreateCheck(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test Report",
		Score:     85,
		Threshold: 70,
		Factors: []confidence.Factor{
			{Name: "Coverage", Score: 90, Weight: 50},
			{Name: "Lint", Score: 80, Weight: 50},
		},
	}

	tests := []struct {
		name           string
		opts           CreateCheckOptions
		serverResponse int
		serverBody     string
		wantErr        bool
		wantErrContain string
	}{
		{
			name: "successful creation",
			opts: CreateCheckOptions{
				Owner: "testowner",
				Repo:  "testrepo",
				SHA:   "abc123",
			},
			serverResponse: http.StatusCreated,
			serverBody:     `{"id": 12345, "html_url": "https://github.com/testowner/testrepo/runs/12345", "status": "completed"}`,
			wantErr:        false,
		},
		{
			name: "uses default name",
			opts: CreateCheckOptions{
				Owner: "testowner",
				Repo:  "testrepo",
				SHA:   "abc123",
				Name:  "",
			},
			serverResponse: http.StatusCreated,
			serverBody:     `{"id": 12345, "html_url": "https://github.com/testowner/testrepo/runs/12345", "status": "completed"}`,
			wantErr:        false,
		},
		{
			name: "custom name",
			opts: CreateCheckOptions{
				Owner: "testowner",
				Repo:  "testrepo",
				SHA:   "abc123",
				Name:  "Custom Check",
			},
			serverResponse: http.StatusCreated,
			serverBody:     `{"id": 12345, "html_url": "https://github.com/testowner/testrepo/runs/12345", "status": "completed"}`,
			wantErr:        false,
		},
		{
			name: "missing owner",
			opts: CreateCheckOptions{
				Owner: "",
				Repo:  "testrepo",
				SHA:   "abc123",
			},
			wantErr:        true,
			wantErrContain: "owner and repo are required",
		},
		{
			name: "missing repo",
			opts: CreateCheckOptions{
				Owner: "testowner",
				Repo:  "",
				SHA:   "abc123",
			},
			wantErr:        true,
			wantErrContain: "owner and repo are required",
		},
		{
			name: "missing SHA",
			opts: CreateCheckOptions{
				Owner: "testowner",
				Repo:  "testrepo",
				SHA:   "",
			},
			wantErr:        true,
			wantErrContain: "SHA is required",
		},
		{
			name: "API error",
			opts: CreateCheckOptions{
				Owner: "testowner",
				Repo:  "testrepo",
				SHA:   "abc123",
			},
			serverResponse: http.StatusUnauthorized,
			serverBody:     `{"message": "Bad credentials"}`,
			wantErr:        true,
			wantErrContain: "API returned status 401",
		},
		{
			name: "server error",
			opts: CreateCheckOptions{
				Owner: "testowner",
				Repo:  "testrepo",
				SHA:   "abc123",
			},
			serverResponse: http.StatusInternalServerError,
			serverBody:     `{"message": "Internal server error"}`,
			wantErr:        true,
			wantErrContain: "API returned status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedRequest *CheckRunRequest
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify headers
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Errorf("Authorization header = %q, want %q", r.Header.Get("Authorization"), "Bearer test-token")
				}
				if r.Header.Get("Accept") != "application/vnd.github+json" {
					t.Errorf("Accept header = %q, want %q", r.Header.Get("Accept"), "application/vnd.github+json")
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Content-Type header = %q, want %q", r.Header.Get("Content-Type"), "application/json")
				}
				if r.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
					t.Errorf("X-GitHub-Api-Version header = %q, want %q", r.Header.Get("X-GitHub-Api-Version"), "2022-11-28")
				}

				// Parse request body
				if err := json.NewDecoder(r.Body).Decode(&receivedRequest); err != nil {
					t.Errorf("Failed to decode request body: %v", err)
				}

				w.WriteHeader(tt.serverResponse)
				_, _ = w.Write([]byte(tt.serverBody))
			}))
			defer server.Close()

			client := NewGitHubClientWithHTTP(
				GitHubClientConfig{BaseURL: server.URL, Token: "test-token"},
				server.Client(),
			)

			resp, err := client.CreateCheck(context.Background(), report, tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.wantErrContain != "" && !contains(err.Error(), tt.wantErrContain) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErrContain)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if resp.ID != 12345 {
				t.Errorf("ID = %d, want 12345", resp.ID)
			}

			// Verify the request was properly formed
			verifyCheckRequest(t, receivedRequest, tt.opts)
		})
	}
}

func verifyCheckRequest(t *testing.T, req *CheckRunRequest, opts CreateCheckOptions) {
	t.Helper()
	if req == nil {
		return
	}

	expectedName := opts.Name
	if expectedName == "" {
		expectedName = "Confidence Score"
	}
	if req.Name != expectedName {
		t.Errorf("request Name = %q, want %q", req.Name, expectedName)
	}
	if req.HeadSHA != opts.SHA {
		t.Errorf("request HeadSHA = %q, want %q", req.HeadSHA, opts.SHA)
	}
	if req.Status != "completed" {
		t.Errorf("request Status = %q, want %q", req.Status, "completed")
	}
	if req.Conclusion != "success" {
		t.Errorf("request Conclusion = %q, want %q", req.Conclusion, "success")
	}
}

func TestGitHubClient_CreateCheck_FailingScore(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test Report",
		Score:     60,
		Threshold: 70, // Score below threshold
	}

	var receivedRequest *CheckRunRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedRequest); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": 12345, "html_url": "https://github.com/test/repo/runs/12345", "status": "completed"}`))
	}))
	defer server.Close()

	client := NewGitHubClientWithHTTP(
		GitHubClientConfig{BaseURL: server.URL, Token: "test-token"},
		server.Client(),
	)

	_, err := client.CreateCheck(context.Background(), report, CreateCheckOptions{
		Owner: "testowner",
		Repo:  "testrepo",
		SHA:   "abc123",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if receivedRequest.Conclusion != "failure" {
		t.Errorf("Conclusion = %q, want %q for failing score", receivedRequest.Conclusion, "failure")
	}
}

func TestBuildCheckOutput(t *testing.T) {
	tests := []struct {
		name       string
		report     *confidence.Report
		wantTitle  string
		wantInSumm []string
	}{
		{
			name: "passing report",
			report: &confidence.Report{
				Title:       "Code Quality",
				Score:       85,
				Threshold:   70,
				Description: "Quality assessment",
			},
			wantTitle:  "Code Quality: 85% (Passed)",
			wantInSumm: []string{"**Score:** 85%", "**Threshold:** 70%", "Quality assessment"},
		},
		{
			name: "failing report",
			report: &confidence.Report{
				Title:     "Code Quality",
				Score:     60,
				Threshold: 70,
			},
			wantTitle:  "Code Quality: 60% (Failed)",
			wantInSumm: []string{"**Score:** 60%", "**Threshold:** 70%"},
		},
		{
			name: "with factors",
			report: &confidence.Report{
				Title:     "Code Quality",
				Score:     85,
				Threshold: 70,
				Factors: []confidence.Factor{
					{Name: "Coverage", Score: 90, Weight: 50},
					{Name: "Lint", Score: 80, Weight: 50},
				},
			},
			wantTitle:  "Code Quality: 85% (Passed)",
			wantInSumm: []string{"Factor Breakdown", "Coverage", "90%", "50%", "Lint", "80%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := buildCheckOutput(tt.report)

			if output.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", output.Title, tt.wantTitle)
			}

			for _, want := range tt.wantInSumm {
				if !contains(output.Summary, want) {
					t.Errorf("Summary missing %q, got: %s", want, output.Summary)
				}
			}
		})
	}
}

func TestLoadGitHubEnv(t *testing.T) {
	// t.Setenv automatically restores env vars after each subtest
	t.Run("all variables set", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "test-token")
		t.Setenv("GITHUB_REPOSITORY", "owner/repo")
		t.Setenv("GITHUB_SHA", "abc123")
		t.Setenv("GITHUB_API_URL", "https://api.github.example.com")

		env, err := LoadGitHubEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if env.Token != "test-token" {
			t.Errorf("Token = %q, want %q", env.Token, "test-token")
		}
		if env.Repository != "owner/repo" {
			t.Errorf("Repository = %q, want %q", env.Repository, "owner/repo")
		}
		if env.SHA != "abc123" {
			t.Errorf("SHA = %q, want %q", env.SHA, "abc123")
		}
		if env.APIURL != "https://api.github.example.com" {
			t.Errorf("APIURL = %q, want %q", env.APIURL, "https://api.github.example.com")
		}
	})

	t.Run("missing token", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("GITHUB_REPOSITORY", "owner/repo")

		_, err := LoadGitHubEnv()
		if err == nil {
			t.Error("expected error for missing token")
		}
		if !contains(err.Error(), "GITHUB_TOKEN") {
			t.Errorf("error = %q, want to mention GITHUB_TOKEN", err.Error())
		}
	})

	t.Run("only token set", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "test-token")
		t.Setenv("GITHUB_REPOSITORY", "")
		t.Setenv("GITHUB_SHA", "")
		t.Setenv("GITHUB_API_URL", "")

		env, err := LoadGitHubEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if env.Token != "test-token" {
			t.Errorf("Token = %q, want %q", env.Token, "test-token")
		}
		if env.Repository != "" {
			t.Errorf("Repository = %q, want empty", env.Repository)
		}
	})
}

func TestParseRepository(t *testing.T) {
	tests := []struct {
		input     string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{"owner/repo", "owner", "repo", false},
		{"my-org/my-repo", "my-org", "my-repo", false},
		{"owner/repo/extra", "owner", "repo/extra", false}, // SplitN(2) keeps extra in repo
		{"owner", "", "", true},
		{"/repo", "", "", true},
		{"owner/", "", "", true},
		{"", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			owner, repo, err := ParseRepository(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}

func TestGitHubClient_FindComment(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse int
		serverBody     string
		wantFound      bool
		wantErr        bool
	}{
		{
			name:           "comment found",
			serverResponse: http.StatusOK,
			serverBody:     `[{"id": 123, "body": "<!-- confvis-comment -->\nSome content", "html_url": "https://github.com/owner/repo/issues/1#issuecomment-123"}]`,
			wantFound:      true,
			wantErr:        false,
		},
		{
			name:           "no confvis comment",
			serverResponse: http.StatusOK,
			serverBody:     `[{"id": 456, "body": "Regular comment", "html_url": "https://github.com/owner/repo/issues/1#issuecomment-456"}]`,
			wantFound:      false,
			wantErr:        false,
		},
		{
			name:           "empty list",
			serverResponse: http.StatusOK,
			serverBody:     `[]`,
			wantFound:      false,
			wantErr:        false,
		},
		{
			name:           "API error",
			serverResponse: http.StatusUnauthorized,
			serverBody:     `{"message": "Bad credentials"}`,
			wantFound:      false,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverResponse)
				_, _ = w.Write([]byte(tt.serverBody))
			}))
			defer server.Close()

			client := NewGitHubClientWithHTTP(
				GitHubClientConfig{BaseURL: server.URL, Token: "test-token"},
				server.Client(),
			)

			resp, err := client.FindComment(context.Background(), CommentOptions{
				Owner: "owner",
				Repo:  "repo",
				PR:    1,
			})

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.wantFound && resp == nil {
				t.Error("expected comment, got nil")
			}
			if !tt.wantFound && resp != nil {
				t.Errorf("expected no comment, got %+v", resp)
			}
		})
	}
}

func TestGitHubClient_PostComment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": 789, "html_url": "https://github.com/owner/repo/issues/1#issuecomment-789", "body": "test comment"}`))
	}))
	defer server.Close()

	client := NewGitHubClientWithHTTP(
		GitHubClientConfig{BaseURL: server.URL, Token: "test-token"},
		server.Client(),
	)

	resp, err := client.PostComment(context.Background(), CommentOptions{
		Owner: "owner",
		Repo:  "repo",
		PR:    1,
	}, "test comment")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ID != 789 {
		t.Errorf("ID = %d, want 789", resp.ID)
	}
}

func TestGitHubClient_UpdateComment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 123, "html_url": "https://github.com/owner/repo/issues/1#issuecomment-123", "body": "updated"}`))
	}))
	defer server.Close()

	client := NewGitHubClientWithHTTP(
		GitHubClientConfig{BaseURL: server.URL, Token: "test-token"},
		server.Client(),
	)

	resp, err := client.UpdateComment(context.Background(), CommentOptions{
		Owner: "owner",
		Repo:  "repo",
		PR:    1,
	}, 123, "updated")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ID != 123 {
		t.Errorf("ID = %d, want 123", resp.ID)
	}
}

func TestGitHubClient_DeleteComment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewGitHubClientWithHTTP(
		GitHubClientConfig{BaseURL: server.URL, Token: "test-token"},
		server.Client(),
	)

	err := client.DeleteComment(context.Background(), CommentOptions{
		Owner: "owner",
		Repo:  "repo",
		PR:    1,
	}, 123)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGitHubClient_FindAllConfvisComments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"id": 1, "body": "<!-- confvis-comment -->\nFirst", "html_url": "url1"},
			{"id": 2, "body": "Regular comment", "html_url": "url2"},
			{"id": 3, "body": "<!-- confvis-comment -->\nSecond", "html_url": "url3"}
		]`))
	}))
	defer server.Close()

	client := NewGitHubClientWithHTTP(
		GitHubClientConfig{BaseURL: server.URL, Token: "test-token"},
		server.Client(),
	)

	comments, err := client.FindAllConfvisComments(context.Background(), CommentOptions{
		Owner: "owner",
		Repo:  "repo",
		PR:    1,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(comments) != 2 {
		t.Errorf("got %d comments, want 2", len(comments))
	}
}

func TestLoadGitHubEnvWithPR(t *testing.T) {
	t.Run("from pull_request event", func(t *testing.T) {
		// Create a temporary event file
		eventJSON := `{"pull_request": {"number": 42}}`
		tmpFile := t.TempDir() + "/event.json"
		if err := writeFile(tmpFile, eventJSON); err != nil {
			t.Fatalf("failed to write event file: %v", err)
		}

		t.Setenv("GITHUB_TOKEN", "test-token")
		t.Setenv("GITHUB_REPOSITORY", "owner/repo")
		t.Setenv("GITHUB_EVENT_PATH", tmpFile)
		t.Setenv("GITHUB_EVENT_NAME", "pull_request")

		env, prNumber, err := LoadGitHubEnvWithPR()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if env.Token != "test-token" {
			t.Errorf("Token = %q, want %q", env.Token, "test-token")
		}
		if prNumber != 42 {
			t.Errorf("prNumber = %d, want 42", prNumber)
		}
	})

	t.Run("from issue_comment event", func(t *testing.T) {
		eventJSON := `{"issue": {"number": 99}}`
		tmpFile := t.TempDir() + "/event.json"
		if err := writeFile(tmpFile, eventJSON); err != nil {
			t.Fatalf("failed to write event file: %v", err)
		}

		t.Setenv("GITHUB_TOKEN", "test-token")
		t.Setenv("GITHUB_REPOSITORY", "owner/repo")
		t.Setenv("GITHUB_EVENT_PATH", tmpFile)
		t.Setenv("GITHUB_EVENT_NAME", "issue_comment")

		env, prNumber, err := LoadGitHubEnvWithPR()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if env == nil {
			t.Fatal("env is nil")
		}
		if prNumber != 99 {
			t.Errorf("prNumber = %d, want 99", prNumber)
		}
	})

	t.Run("no event file", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "test-token")
		t.Setenv("GITHUB_REPOSITORY", "owner/repo")
		t.Setenv("GITHUB_EVENT_PATH", "")

		env, prNumber, err := LoadGitHubEnvWithPR()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if env == nil {
			t.Fatal("env is nil")
		}
		if prNumber != 0 {
			t.Errorf("prNumber = %d, want 0", prNumber)
		}
	})
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
