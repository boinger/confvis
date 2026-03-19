package cli

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/boinger/confvis/internal/checks"
	"github.com/boinger/confvis/internal/confidence"
)

// ============================================================================
// Check GitHub Command Tests
// ============================================================================

func defaultCheckGitHubDeps(fs *MockFileSystem) *CheckGitHubDeps {
	return &CheckGitHubDeps{
		FS:         fs,
		Stdin:      strings.NewReader(""),
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		Verbose:    false,
		Quiet:      false,
		Config:     "config.json",
		Owner:      "testowner",
		Repo:       "testrepo",
		SHA:        "abc123",
		Name:       "Confidence Score",
		Token:      "test-token",
		APIURL:     "",
		AutoDetect: false,
		Timeout:    time.Second,
		CreateCheck: func(ctx context.Context, client *checks.GitHubClient,
			report *confidence.Report, opts checks.CreateCheckOptions) (*checks.CheckRunResponse, error) {
			return &checks.CheckRunResponse{
				ID:      12345,
				HTMLURL: "https://github.com/testowner/testrepo/runs/12345",
				Status:  "completed",
			}, nil
		},
		LoadGitHubEnv: func() (*checks.GitHubEnv, error) {
			return nil, fmt.Errorf("not in GitHub Actions")
		},
	}
}

func TestCheckGitHubImpl_BasicFlow(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultCheckGitHubDeps(fs)
	deps.Stdout = &stdout

	err := checkGitHubImpl(deps)
	if err != nil {
		t.Fatalf("checkGitHubImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Created check run") {
		t.Error("expected check run creation message")
	}
	if !strings.Contains(output, "passed") {
		t.Error("expected passed status for score above threshold")
	}
}

func TestCheckGitHubImpl_FailingScore(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 60, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultCheckGitHubDeps(fs)
	deps.Stdout = &stdout

	err := checkGitHubImpl(deps)
	if err != nil {
		t.Fatalf("checkGitHubImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "failed") {
		t.Error("expected failed status for score below threshold")
	}
}

func TestCheckGitHubImpl_QuietMode(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultCheckGitHubDeps(fs)
	deps.Stdout = &stdout
	deps.Quiet = true

	err := checkGitHubImpl(deps)
	if err != nil {
		t.Fatalf("checkGitHubImpl() error = %v", err)
	}

	if stdout.Len() > 0 {
		t.Errorf("quiet mode should suppress output, got: %s", stdout.String())
	}
}

func TestCheckGitHubImpl_StdinInput(t *testing.T) {
	fs := NewMockFileSystem()

	var stdout bytes.Buffer
	deps := defaultCheckGitHubDeps(fs)
	deps.Config = "-"
	deps.Stdin = strings.NewReader(`{"title": "Stdin Test", "score": 90, "threshold": 80}`)
	deps.Stdout = &stdout

	err := checkGitHubImpl(deps)
	if err != nil {
		t.Fatalf("checkGitHubImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Created check run") {
		t.Error("expected check run creation message")
	}
}

func TestCheckGitHubImpl_MissingToken(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCheckGitHubDeps(fs)
	deps.Token = ""
	deps.AutoDetect = false

	err := checkGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if !strings.Contains(err.Error(), "token required") {
		t.Errorf("error = %q, want to contain 'token required'", err.Error())
	}
}

func TestCheckGitHubImpl_MissingOwner(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCheckGitHubDeps(fs)
	deps.Owner = ""

	err := checkGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error for missing owner")
	}
	if !strings.Contains(err.Error(), "owner required") {
		t.Errorf("error = %q, want to contain 'owner required'", err.Error())
	}
}

func TestCheckGitHubImpl_MissingRepo(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCheckGitHubDeps(fs)
	deps.Repo = ""

	err := checkGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error for missing repo")
	}
	if !strings.Contains(err.Error(), "repo required") {
		t.Errorf("error = %q, want to contain 'repo required'", err.Error())
	}
}

func TestCheckGitHubImpl_MissingSHA(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCheckGitHubDeps(fs)
	deps.SHA = ""

	err := checkGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error for missing SHA")
	}
	if !strings.Contains(err.Error(), "SHA required") {
		t.Errorf("error = %q, want to contain 'SHA required'", err.Error())
	}
}

func TestCheckGitHubImpl_AutoDetect(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultCheckGitHubDeps(fs)
	deps.Stdout = &stdout
	deps.Token = "" // Will be filled from env
	deps.Owner = "" // Will be filled from env
	deps.Repo = ""  // Will be filled from env
	deps.SHA = ""   // Will be filled from env
	deps.AutoDetect = true
	deps.LoadGitHubEnv = func() (*checks.GitHubEnv, error) {
		return &checks.GitHubEnv{
			Token:      "env-token",
			Repository: "envowner/envrepo",
			SHA:        "env123",
			APIURL:     "https://api.github.example.com",
		}, nil
	}

	err := checkGitHubImpl(deps)
	if err != nil {
		t.Fatalf("checkGitHubImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Created check run") {
		t.Error("expected check run creation message")
	}
}

func TestCheckGitHubImpl_AutoDetectWithFlagOverrides(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var calledOwner, calledRepo string
	deps := defaultCheckGitHubDeps(fs)
	deps.Owner = "flag-owner" // Override from flag
	deps.Repo = ""            // Will be filled from env
	deps.SHA = ""             // Will be filled from env
	deps.Token = ""           // Will be filled from env
	deps.AutoDetect = true
	deps.LoadGitHubEnv = func() (*checks.GitHubEnv, error) {
		return &checks.GitHubEnv{
			Token:      "env-token",
			Repository: "envowner/envrepo",
			SHA:        "env123",
		}, nil
	}
	deps.CreateCheck = func(ctx context.Context, client *checks.GitHubClient,
		report *confidence.Report, opts checks.CreateCheckOptions) (*checks.CheckRunResponse, error) {
		calledOwner = opts.Owner
		calledRepo = opts.Repo
		return &checks.CheckRunResponse{
			ID:      12345,
			HTMLURL: "https://github.com/test/runs/12345",
			Status:  "completed",
		}, nil
	}

	err := checkGitHubImpl(deps)
	if err != nil {
		t.Fatalf("checkGitHubImpl() error = %v", err)
	}

	// Flag should override env
	if calledOwner != "flag-owner" {
		t.Errorf("owner = %q, want 'flag-owner' (flag should override env)", calledOwner)
	}
	// Env should fill missing repo
	if calledRepo != "envrepo" {
		t.Errorf("repo = %q, want 'envrepo' (from env)", calledRepo)
	}
}

func TestCheckGitHubImpl_AutoDetectEnvError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCheckGitHubDeps(fs)
	deps.Token = ""
	deps.AutoDetect = true
	deps.LoadGitHubEnv = func() (*checks.GitHubEnv, error) {
		return nil, fmt.Errorf("GITHUB_TOKEN not set")
	}

	err := checkGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error when env loading fails and no token provided")
	}
	if !strings.Contains(err.Error(), "GitHub environment") {
		t.Errorf("error = %q, want to contain 'GitHub environment'", err.Error())
	}
}

func TestCheckGitHubImpl_CreateCheckError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCheckGitHubDeps(fs)
	deps.CreateCheck = func(ctx context.Context, client *checks.GitHubClient,
		report *confidence.Report, opts checks.CreateCheckOptions) (*checks.CheckRunResponse, error) {
		return nil, fmt.Errorf("API error: unauthorized")
	}

	err := checkGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error when create check fails")
	}
	if !strings.Contains(err.Error(), "creating check run") {
		t.Errorf("error = %q, want to contain 'creating check run'", err.Error())
	}
}

func TestCheckGitHubImpl_InvalidConfig(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `invalid json`)

	deps := defaultCheckGitHubDeps(fs)

	err := checkGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
	if !strings.Contains(err.Error(), "parsing config") {
		t.Errorf("error = %q, want to contain 'parsing config'", err.Error())
	}
}

func TestCheckGitHubImpl_ConfigFileNotFound(t *testing.T) {
	fs := NewMockFileSystem()
	// Don't set config.json - it will be missing

	deps := defaultCheckGitHubDeps(fs)

	err := checkGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error when config file not found")
	}
	if !strings.Contains(err.Error(), "opening config") {
		t.Errorf("error = %q, want to contain 'opening config'", err.Error())
	}
}

func TestCheckGitHubImpl_DryRun(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75, "factors": [{"name": "Coverage", "score": 90, "weight": 50}]}`)

	var stdout bytes.Buffer
	deps := defaultCheckGitHubDeps(fs)
	deps.Stdout = &stdout
	deps.DryRun = true

	err := checkGitHubImpl(deps)
	if err != nil {
		t.Fatalf("checkGitHubImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "DRY RUN") {
		t.Error("expected DRY RUN header")
	}
	if !strings.Contains(output, "testowner/testrepo") {
		t.Error("expected repo in output")
	}
	if !strings.Contains(output, "abc123") {
		t.Error("expected SHA in output")
	}
	if !strings.Contains(output, "No changes made") {
		t.Error("expected no-changes message")
	}
}

