package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/boinger/confvis/internal/baseline"
	"github.com/boinger/confvis/internal/confidence"
)

// ============================================================================
// baselineSaveImpl / baselineShowImpl Tests
// ============================================================================

func TestBaselineSaveImpl_ToFile(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	savedPath := ""
	deps := &BaselineDeps{
		FS:     fs,
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
		Config: "config.json",
		File:   "baseline.json",
		FileWriter: func(path string, b *baseline.Baseline) error {
			savedPath = path
			return nil
		},
	}

	err := baselineSaveImpl(deps)
	if err != nil {
		t.Fatalf("baselineSaveImpl() error = %v", err)
	}
	if savedPath != "baseline.json" {
		t.Errorf("saved to %q, want %q", savedPath, "baseline.json")
	}
}

func TestBaselineSaveImpl_ToGitRef(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	savedRef := ""
	deps := &BaselineDeps{
		FS:        fs,
		Stdin:     strings.NewReader(""),
		Stdout:    &stdout,
		Config:    "config.json",
		Ref:       "refs/confvis/baseline",
		IsGitRepo: func() bool { return true },
		GitRefWriter: func(ref string, b *baseline.Baseline) error {
			savedRef = ref
			return nil
		},
	}

	err := baselineSaveImpl(deps)
	if err != nil {
		t.Fatalf("baselineSaveImpl() error = %v", err)
	}
	if savedRef != "refs/confvis/baseline" {
		t.Errorf("saved ref = %q, want %q", savedRef, "refs/confvis/baseline")
	}
}

func TestBaselineSaveImpl_NotInGitRepo(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := &BaselineDeps{
		FS:        fs,
		Stdin:     strings.NewReader(""),
		Stdout:    &bytes.Buffer{},
		Config:    "config.json",
		IsGitRepo: func() bool { return false },
	}

	err := baselineSaveImpl(deps)
	if err == nil {
		t.Fatal("expected error when not in git repo and no file")
	}
	if !strings.Contains(err.Error(), "not in a git repository") {
		t.Errorf("error = %q, should mention not in git repo", err.Error())
	}
}

func TestBaselineShowImpl_TextFromFile(t *testing.T) {
	var stdout bytes.Buffer
	score := 85
	deps := &BaselineDeps{
		Stdout: &stdout,
		Format: "text",
		File:   "baseline.json",
		FileReader: func(path string) (*baseline.Baseline, error) {
			return &baseline.Baseline{
				Report:  confidence.Report{Score: &score, Title: "Test"},
				SavedAt: "2024-01-01T00:00:00Z",
				Commit:  "abc1234567890",
				Branch:  "main",
			}, nil
		},
	}

	err := baselineShowImpl(deps)
	if err != nil {
		t.Fatalf("baselineShowImpl() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "85%") {
		t.Errorf("output should contain 85%%, got: %s", stdout.String())
	}
}

func TestBaselineShowImpl_JSONFromGitRef(t *testing.T) {
	var stdout bytes.Buffer
	score := 90
	deps := &BaselineDeps{
		Stdout:    &stdout,
		Format:    "json",
		Ref:       "refs/confvis/baseline",
		IsGitRepo: func() bool { return true },
		GitRefReader: func(ref string) (*baseline.Baseline, error) {
			return &baseline.Baseline{
				Report: confidence.Report{Score: &score, Title: "Test"},
			}, nil
		},
	}

	err := baselineShowImpl(deps)
	if err != nil {
		t.Fatalf("baselineShowImpl() error = %v", err)
	}

	if !strings.Contains(stdout.String(), `"score": 90`) {
		t.Errorf("JSON output should contain score, got: %s", stdout.String())
	}
}

func TestBaselineShowImpl_NoBaselineFound(t *testing.T) {
	deps := &BaselineDeps{
		Stdout:    &bytes.Buffer{},
		Format:    "text",
		IsGitRepo: func() bool { return false },
	}

	err := baselineShowImpl(deps)
	if err == nil {
		t.Fatal("expected error when no baseline is found")
	}
	if !strings.Contains(err.Error(), "not in a git repository") {
		t.Errorf("error = %q, should mention not in git repo", err.Error())
	}
}
