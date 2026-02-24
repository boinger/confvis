package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/boinger/confvis/internal/baseline"
	"github.com/boinger/confvis/internal/checks"
	"github.com/boinger/confvis/internal/confidence"
)

// ============================================================================
// commentGitHubImpl Tests
// ============================================================================

func defaultCommentGitHubDeps(fs FileSystem) *CommentGitHubDeps {
	return &CommentGitHubDeps{
		FS:         fs,
		Stdin:      nil,
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		Verbose:    false,
		Quiet:      false,
		Config:     "config.json",
		Owner:      "owner",
		Repo:       "repo",
		PR:         123,
		Token:      "test-token",
		APIURL:     "",
		Mode:       "update",
		AutoDetect: false,
		DryRun:     false,
		Timeout:    30 * time.Second,
		FindComment: func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions) (*checks.CommentResponse, error) {
			return nil, nil
		},
		PostComment: func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions, _ string) (*checks.CommentResponse, error) {
			return &checks.CommentResponse{ID: 1, HTMLURL: "https://github.com/owner/repo/pull/123#comment-1"}, nil
		},
		UpdateComment: func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions, _ int64, _ string) (*checks.CommentResponse, error) {
			return &checks.CommentResponse{ID: 1, HTMLURL: "https://github.com/owner/repo/pull/123#comment-1"}, nil
		},
		DeleteComment: func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions, _ int64) error {
			return nil
		},
		FindAllComments: func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions) ([]checks.CommentResponse, error) {
			return nil, nil
		},
		LoadGitHubEnvPR: func() (*checks.GitHubEnv, int, error) {
			return nil, 0, nil
		},
	}
}

func TestCommentGitHubImpl_CreateMode(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultCommentGitHubDeps(fs)
	deps.Stdout = &stdout
	deps.Mode = "create"

	err := commentGitHubImpl(deps)
	if err != nil {
		t.Fatalf("commentGitHubImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Posted comment") {
		t.Errorf("output = %q, want to contain 'Posted comment'", output)
	}
}

func TestCommentGitHubImpl_UpdateMode_ExistingComment(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultCommentGitHubDeps(fs)
	deps.Stdout = &stdout
	deps.Mode = "update"
	deps.FindComment = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions) (*checks.CommentResponse, error) {
		return &checks.CommentResponse{ID: 42, HTMLURL: "https://github.com/owner/repo/pull/123#comment-42"}, nil
	}

	updated := false
	deps.UpdateComment = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions, commentID int64, _ string) (*checks.CommentResponse, error) {
		if commentID != 42 {
			t.Errorf("UpdateComment commentID = %d, want 42", commentID)
		}
		updated = true
		return &checks.CommentResponse{ID: 42, HTMLURL: "https://github.com/owner/repo/pull/123#comment-42"}, nil
	}

	err := commentGitHubImpl(deps)
	if err != nil {
		t.Fatalf("commentGitHubImpl() error = %v", err)
	}

	if !updated {
		t.Error("UpdateComment should have been called")
	}
}

func TestCommentGitHubImpl_UpdateMode_NoExistingComment(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultCommentGitHubDeps(fs)
	deps.Stdout = &stdout
	deps.Mode = "update"
	deps.FindComment = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions) (*checks.CommentResponse, error) {
		return nil, nil //nolint:nilnil // "not found" is a valid test scenario
	}

	posted := false
	deps.PostComment = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions, _ string) (*checks.CommentResponse, error) {
		posted = true
		return &checks.CommentResponse{ID: 1, HTMLURL: "https://github.com/owner/repo/pull/123#comment-1"}, nil
	}

	err := commentGitHubImpl(deps)
	if err != nil {
		t.Fatalf("commentGitHubImpl() error = %v", err)
	}

	if !posted {
		t.Error("PostComment should have been called when no existing comment")
	}
}

func TestCommentGitHubImpl_ReplaceMode(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultCommentGitHubDeps(fs)
	deps.Stdout = &stdout
	deps.Mode = "replace"

	deletedIDs := []int64{}
	deps.FindAllComments = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions) ([]checks.CommentResponse, error) {
		return []checks.CommentResponse{
			{ID: 10, HTMLURL: "https://github.com/owner/repo/pull/123#comment-10"},
			{ID: 20, HTMLURL: "https://github.com/owner/repo/pull/123#comment-20"},
		}, nil
	}
	deps.DeleteComment = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions, commentID int64) error {
		deletedIDs = append(deletedIDs, commentID)
		return nil
	}

	posted := false
	deps.PostComment = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions, _ string) (*checks.CommentResponse, error) {
		posted = true
		return &checks.CommentResponse{ID: 30, HTMLURL: "https://github.com/owner/repo/pull/123#comment-30"}, nil
	}

	err := commentGitHubImpl(deps)
	if err != nil {
		t.Fatalf("commentGitHubImpl() error = %v", err)
	}

	if len(deletedIDs) != 2 {
		t.Errorf("expected 2 deletions, got %d", len(deletedIDs))
	}
	if !posted {
		t.Error("PostComment should have been called after deletions")
	}
}

func TestCommentGitHubImpl_DryRun(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultCommentGitHubDeps(fs)
	deps.Stdout = &stdout
	deps.DryRun = true

	// None of the API functions should be called in dry-run mode
	deps.PostComment = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions, _ string) (*checks.CommentResponse, error) {
		t.Error("PostComment should not be called in dry-run mode")
		return nil, nil //nolint:nilnil // test assertion function, return doesn't matter
	}

	err := commentGitHubImpl(deps)
	if err != nil {
		t.Fatalf("commentGitHubImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "DRY RUN") {
		t.Errorf("output = %q, want to contain 'DRY RUN'", output)
	}
	if !strings.Contains(output, "owner/repo") {
		t.Errorf("output = %q, want to contain 'owner/repo'", output)
	}
	if !strings.Contains(output, "No changes made") {
		t.Errorf("output = %q, want to contain 'No changes made'", output)
	}
}

func TestCommentGitHubImpl_InvalidMode(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCommentGitHubDeps(fs)
	deps.Mode = "invalid"

	err := commentGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
	if !strings.Contains(err.Error(), "invalid mode") {
		t.Errorf("error = %q, want to contain 'invalid mode'", err.Error())
	}
}

func TestCommentGitHubImpl_MissingToken(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCommentGitHubDeps(fs)
	deps.Token = ""
	deps.AutoDetect = false

	err := commentGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error when token is missing")
	}
	if !strings.Contains(err.Error(), "token required") {
		t.Errorf("error = %q, want to contain 'token required'", err.Error())
	}
}

func TestCommentGitHubImpl_MissingOwner(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCommentGitHubDeps(fs)
	deps.Owner = ""
	deps.AutoDetect = false

	err := commentGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error when owner is missing")
	}
	if !strings.Contains(err.Error(), "owner required") {
		t.Errorf("error = %q, want to contain 'owner required'", err.Error())
	}
}

func TestCommentGitHubImpl_MissingPR(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCommentGitHubDeps(fs)
	deps.PR = 0
	deps.AutoDetect = false

	err := commentGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error when PR is missing")
	}
	if !strings.Contains(err.Error(), "PR number required") {
		t.Errorf("error = %q, want to contain 'PR number required'", err.Error())
	}
}

func TestCommentGitHubImpl_InvalidConfig(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `invalid json`)

	deps := defaultCommentGitHubDeps(fs)

	err := commentGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
	if !strings.Contains(err.Error(), "parsing config") {
		t.Errorf("error = %q, want to contain 'parsing config'", err.Error())
	}
}

func TestCommentGitHubImpl_PostCommentError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCommentGitHubDeps(fs)
	deps.Mode = "create"
	deps.PostComment = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions, _ string) (*checks.CommentResponse, error) {
		return nil, errors.New("API error: rate limited")
	}

	err := commentGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error when PostComment fails")
	}
	if !strings.Contains(err.Error(), "creating comment") {
		t.Errorf("error = %q, want to contain 'creating comment'", err.Error())
	}
}

func TestCommentGitHubImpl_FindCommentError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCommentGitHubDeps(fs)
	deps.Mode = "update"
	deps.FindComment = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions) (*checks.CommentResponse, error) {
		return nil, errors.New("API error")
	}

	err := commentGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error when FindComment fails")
	}
	if !strings.Contains(err.Error(), "finding existing comment") {
		t.Errorf("error = %q, want to contain 'finding existing comment'", err.Error())
	}
}

func TestCommentGitHubImpl_UpdateCommentError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCommentGitHubDeps(fs)
	deps.Mode = "update"
	deps.FindComment = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions) (*checks.CommentResponse, error) {
		return &checks.CommentResponse{ID: 42}, nil
	}
	deps.UpdateComment = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions, _ int64, _ string) (*checks.CommentResponse, error) {
		return nil, errors.New("API error")
	}

	err := commentGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error when UpdateComment fails")
	}
	if !strings.Contains(err.Error(), "updating comment") {
		t.Errorf("error = %q, want to contain 'updating comment'", err.Error())
	}
}

func TestCommentGitHubImpl_DeleteCommentError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCommentGitHubDeps(fs)
	deps.Mode = "replace"
	deps.FindAllComments = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions) ([]checks.CommentResponse, error) {
		return []checks.CommentResponse{{ID: 10}}, nil
	}
	deps.DeleteComment = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions, _ int64) error {
		return errors.New("API error")
	}

	err := commentGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error when DeleteComment fails")
	}
	if !strings.Contains(err.Error(), "deleting comment") {
		t.Errorf("error = %q, want to contain 'deleting comment'", err.Error())
	}
}

func TestCommentGitHubImpl_QuietMode(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultCommentGitHubDeps(fs)
	deps.Stdout = &stdout
	deps.Quiet = true
	deps.Mode = "create"

	err := commentGitHubImpl(deps)
	if err != nil {
		t.Fatalf("commentGitHubImpl() error = %v", err)
	}

	// Quiet mode should not output anything
	if stdout.Len() > 0 {
		t.Errorf("quiet mode should produce no output, got: %q", stdout.String())
	}
}

func TestCommentGitHubImpl_FailedReport(t *testing.T) {
	fs := NewMockFileSystem()
	// Score below threshold = failed
	fs.SetFileContent("config.json", `{"title": "Test", "score": 50, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultCommentGitHubDeps(fs)
	deps.Stdout = &stdout
	deps.Mode = "create"

	err := commentGitHubImpl(deps)
	if err != nil {
		t.Fatalf("commentGitHubImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "failed") {
		t.Errorf("output = %q, want to contain 'failed'", output)
	}
}

func TestCommentGitHubImpl_StdinInput(t *testing.T) {
	fs := NewMockFileSystem()

	stdin := strings.NewReader(`{"title": "Stdin Test", "score": 90, "threshold": 75}`)
	var stdout bytes.Buffer

	deps := defaultCommentGitHubDeps(fs)
	deps.Stdin = stdin
	deps.Stdout = &stdout
	deps.Config = "-"
	deps.Mode = "create"

	err := commentGitHubImpl(deps)
	if err != nil {
		t.Fatalf("commentGitHubImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Posted comment") {
		t.Errorf("output = %q, want to contain 'Posted comment'", output)
	}
}

func TestCommentGitHubImpl_AutoDetect(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultCommentGitHubDeps(fs)
	deps.Stdout = &stdout
	deps.Owner = ""
	deps.Repo = ""
	deps.Token = ""
	deps.PR = 0
	deps.AutoDetect = true
	deps.Mode = "create"
	deps.LoadGitHubEnvPR = func() (*checks.GitHubEnv, int, error) {
		return &checks.GitHubEnv{
			Token:      "auto-token",
			Repository: "auto-owner/auto-repo",
			APIURL:     "https://api.github.com",
		}, 456, nil
	}

	err := commentGitHubImpl(deps)
	if err != nil {
		t.Fatalf("commentGitHubImpl() error = %v", err)
	}
}

func TestCommentGitHubImpl_AutoDetectError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCommentGitHubDeps(fs)
	deps.Token = "" // No explicit token
	deps.AutoDetect = true
	deps.LoadGitHubEnvPR = func() (*checks.GitHubEnv, int, error) {
		return nil, 0, errors.New("not in GitHub Actions")
	}

	err := commentGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error when auto-detect fails and no token")
	}
	if !strings.Contains(err.Error(), "loading GitHub environment") {
		t.Errorf("error = %q, want to contain 'loading GitHub environment'", err.Error())
	}
}

// ============================================================================
// Baseline Tests
// ============================================================================

func TestCommentGitHubImpl_BaselineCompare(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 80, "threshold": 75}`)

	var postedBody string
	deps := defaultCommentGitHubDeps(fs)
	deps.Mode = "create"
	deps.Baseline = BaselineConfig{
		Compare: "baseline.json",
		FS:      fs,
	}
	deps.PostComment = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions, body string) (*checks.CommentResponse, error) {
		postedBody = body
		return &checks.CommentResponse{ID: 1, HTMLURL: "https://github.com/owner/repo/pull/123#comment-1"}, nil
	}

	err := commentGitHubImpl(deps)
	if err != nil {
		t.Fatalf("commentGitHubImpl() error = %v", err)
	}

	if !strings.Contains(postedBody, "Change") {
		t.Errorf("posted body should contain 'Change' row, got:\n%s", postedBody)
	}
	if !strings.Contains(postedBody, ":arrow_up:") {
		t.Errorf("posted body should contain ':arrow_up:' for score increase, got:\n%s", postedBody)
	}
	if !strings.Contains(postedBody, "+5") {
		t.Errorf("posted body should contain '+5' delta, got:\n%s", postedBody)
	}
}

func TestCommentGitHubImpl_CompareBaseline_AutoFetch(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 90, "threshold": 75}`)

	baselineScore := 80
	var postedBody string
	deps := defaultCommentGitHubDeps(fs)
	deps.Mode = "create"
	deps.Baseline = BaselineConfig{
		CompareBaseline: true,
		BaselineFile:    "bl.json",
		BaselineFileReader: func(_ string) (*baseline.Baseline, error) {
			return &baseline.Baseline{
				Report: confidence.Report{Score: &baselineScore, Title: "BL", Threshold: 75},
			}, nil
		},
	}
	deps.PostComment = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions, body string) (*checks.CommentResponse, error) {
		postedBody = body
		return &checks.CommentResponse{ID: 1, HTMLURL: "https://github.com/owner/repo/pull/123#comment-1"}, nil
	}

	err := commentGitHubImpl(deps)
	if err != nil {
		t.Fatalf("commentGitHubImpl() error = %v", err)
	}

	if !strings.Contains(postedBody, "+10") {
		t.Errorf("posted body should contain '+10' delta, got:\n%s", postedBody)
	}
}

func TestCommentGitHubImpl_BaselineRegression(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 70, "threshold": 75}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 85, "threshold": 75}`)

	var postedBody string
	deps := defaultCommentGitHubDeps(fs)
	deps.Mode = "create"
	deps.Baseline = BaselineConfig{
		Compare: "baseline.json",
		FS:      fs,
	}
	deps.PostComment = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions, body string) (*checks.CommentResponse, error) {
		postedBody = body
		return &checks.CommentResponse{ID: 1, HTMLURL: "https://github.com/owner/repo/pull/123#comment-1"}, nil
	}

	err := commentGitHubImpl(deps)
	if err != nil {
		t.Fatalf("commentGitHubImpl() error = %v", err)
	}

	if !strings.Contains(postedBody, ":arrow_down:") {
		t.Errorf("posted body should contain ':arrow_down:' for regression, got:\n%s", postedBody)
	}
	if !strings.Contains(postedBody, "-15") {
		t.Errorf("posted body should contain '-15' delta, got:\n%s", postedBody)
	}
}

func TestCommentGitHubImpl_BaselineNoChange(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 85, "threshold": 75}`)

	var postedBody string
	deps := defaultCommentGitHubDeps(fs)
	deps.Mode = "create"
	deps.Baseline = BaselineConfig{
		Compare: "baseline.json",
		FS:      fs,
	}
	deps.PostComment = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions, body string) (*checks.CommentResponse, error) {
		postedBody = body
		return &checks.CommentResponse{ID: 1, HTMLURL: "https://github.com/owner/repo/pull/123#comment-1"}, nil
	}

	err := commentGitHubImpl(deps)
	if err != nil {
		t.Fatalf("commentGitHubImpl() error = %v", err)
	}

	if !strings.Contains(postedBody, ":left_right_arrow:") {
		t.Errorf("posted body should contain ':left_right_arrow:' for no change, got:\n%s", postedBody)
	}
}

func TestCommentGitHubImpl_BaselineError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCommentGitHubDeps(fs)
	deps.Baseline = BaselineConfig{
		CompareBaseline: true,
		BaselineFile:    "baseline.json",
		BaselineFileReader: func(_ string) (*baseline.Baseline, error) {
			return nil, errors.New("corrupted baseline")
		},
	}

	err := commentGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error for corrupted baseline")
	}
	if !strings.Contains(err.Error(), "loading baseline") {
		t.Errorf("error = %q, want to contain 'loading baseline'", err.Error())
	}
}

func TestCommentGitHubImpl_DryRun_WithBaseline(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 80, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultCommentGitHubDeps(fs)
	deps.Stdout = &stdout
	deps.DryRun = true
	deps.Baseline = BaselineConfig{
		Compare: "baseline.json",
		FS:      fs,
	}

	err := commentGitHubImpl(deps)
	if err != nil {
		t.Fatalf("commentGitHubImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Baseline:") {
		t.Errorf("dry-run output should contain 'Baseline:' line, got:\n%s", output)
	}
	if !strings.Contains(output, "80 -> 85 (+5)") {
		t.Errorf("dry-run output should show baseline delta, got:\n%s", output)
	}
}

func TestCommentGitHubImpl_NoBaseline_BackwardCompatible(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var postedBody string
	deps := defaultCommentGitHubDeps(fs)
	deps.Mode = "create"
	// No baseline config set — zero value BaselineConfig
	deps.PostComment = func(_ context.Context, _ *checks.GitHubClient, _ checks.CommentOptions, body string) (*checks.CommentResponse, error) {
		postedBody = body
		return &checks.CommentResponse{ID: 1, HTMLURL: "https://github.com/owner/repo/pull/123#comment-1"}, nil
	}

	err := commentGitHubImpl(deps)
	if err != nil {
		t.Fatalf("commentGitHubImpl() error = %v", err)
	}

	if strings.Contains(postedBody, "Change") {
		t.Errorf("without baseline, body should not contain 'Change' row, got:\n%s", postedBody)
	}
}

