package gitutil

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

// setupGitRepo creates a temporary git repository for testing.
func setupGitRepo(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	return tmpDir
}

// withDir changes to the given directory for the duration of the test.
func withDir(t *testing.T, dir string) {
	t.Helper()
	t.Chdir(dir)
}

func TestWriteRef_Simple(t *testing.T) {
	repoDir := setupGitRepo(t)
	withDir(t, repoDir)

	ref := "refs/confvis/test-write"
	content := []byte("hello world\n")

	if err := WriteRef(ref, content, ""); err != nil {
		t.Fatalf("WriteRef() error = %v", err)
	}

	if !RefExists(ref) {
		t.Fatal("ref should exist after WriteRef")
	}

	got, err := ReadRefContent(ref)
	if err != nil {
		t.Fatalf("ReadRefContent() error = %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("ReadRefContent() = %q, want %q", string(got), string(content))
	}
}

func TestWriteRef_Overwrite(t *testing.T) {
	repoDir := setupGitRepo(t)
	withDir(t, repoDir)

	ref := "refs/confvis/test-overwrite"

	if err := WriteRef(ref, []byte("first"), ""); err != nil {
		t.Fatalf("WriteRef() first error = %v", err)
	}
	if err := WriteRef(ref, []byte("second"), ""); err != nil {
		t.Fatalf("WriteRef() second error = %v", err)
	}

	got, err := ReadRefContent(ref)
	if err != nil {
		t.Fatalf("ReadRefContent() error = %v", err)
	}
	if string(got) != "second" {
		t.Errorf("ReadRefContent() = %q, want %q", string(got), "second")
	}
}

func TestWriteRef_CAS_Success(t *testing.T) {
	repoDir := setupGitRepo(t)
	withDir(t, repoDir)

	ref := "refs/confvis/test-cas"

	if err := WriteRef(ref, []byte("v1"), ""); err != nil {
		t.Fatalf("WriteRef() error = %v", err)
	}

	sha, ok := ReadRef(ref)
	if !ok {
		t.Fatal("ReadRef() returned false for existing ref")
	}

	// CAS with correct old SHA should succeed
	if err := WriteRef(ref, []byte("v2"), sha); err != nil {
		t.Fatalf("WriteRef() CAS error = %v", err)
	}

	got, err := ReadRefContent(ref)
	if err != nil {
		t.Fatalf("ReadRefContent() error = %v", err)
	}
	if string(got) != "v2" {
		t.Errorf("ReadRefContent() = %q, want %q", string(got), "v2")
	}
}

func TestWriteRef_CAS_Conflict(t *testing.T) {
	repoDir := setupGitRepo(t)
	withDir(t, repoDir)

	ref := "refs/confvis/test-cas-conflict"

	if err := WriteRef(ref, []byte("v1"), ""); err != nil {
		t.Fatalf("WriteRef() error = %v", err)
	}

	sha, ok := ReadRef(ref)
	if !ok {
		t.Fatal("ReadRef() returned false for existing ref")
	}

	// Intervening write changes the ref
	if err := WriteRef(ref, []byte("v2-intervening"), ""); err != nil {
		t.Fatalf("WriteRef() intervening error = %v", err)
	}

	// CAS with stale SHA should fail
	err := WriteRef(ref, []byte("v3-should-fail"), sha)
	if err == nil {
		t.Fatal("WriteRef() CAS with stale SHA should fail")
	}
	if !errors.Is(err, ErrRefConflict) {
		t.Errorf("WriteRef() error = %v, want ErrRefConflict", err)
	}

	// Original intervening write should be preserved
	got, err := ReadRefContent(ref)
	if err != nil {
		t.Fatalf("ReadRefContent() error = %v", err)
	}
	if string(got) != "v2-intervening" {
		t.Errorf("ReadRefContent() = %q, want %q (intervening write preserved)", string(got), "v2-intervening")
	}
}

func TestWriteRef_CAS_CreateOnly(t *testing.T) {
	repoDir := setupGitRepo(t)
	withDir(t, repoDir)

	ref := "refs/confvis/test-create-only"

	// ZeroSHA = create-only — should succeed on new ref
	if err := WriteRef(ref, []byte("created"), ZeroSHA); err != nil {
		t.Fatalf("WriteRef() create-only error = %v", err)
	}

	got, err := ReadRefContent(ref)
	if err != nil {
		t.Fatalf("ReadRefContent() error = %v", err)
	}
	if string(got) != "created" {
		t.Errorf("ReadRefContent() = %q, want %q", string(got), "created")
	}

	// ZeroSHA again should fail — ref already exists
	err = WriteRef(ref, []byte("duplicate"), ZeroSHA)
	if err == nil {
		t.Fatal("WriteRef() ZeroSHA on existing ref should fail")
	}
	if !errors.Is(err, ErrRefConflict) {
		t.Errorf("WriteRef() error = %v, want ErrRefConflict", err)
	}
}

func TestReadRef_Exists(t *testing.T) {
	repoDir := setupGitRepo(t)
	withDir(t, repoDir)

	ref := "refs/confvis/test-readref"
	if err := WriteRef(ref, []byte("data"), ""); err != nil {
		t.Fatalf("WriteRef() error = %v", err)
	}

	sha, ok := ReadRef(ref)
	if !ok {
		t.Fatal("ReadRef() returned false for existing ref")
	}
	if sha == "" {
		t.Error("ReadRef() returned empty SHA for existing ref")
	}
	if len(sha) != 40 {
		t.Errorf("ReadRef() SHA length = %d, want 40", len(sha))
	}
}

func TestReadRef_NotExists(t *testing.T) {
	repoDir := setupGitRepo(t)
	withDir(t, repoDir)

	sha, ok := ReadRef("refs/confvis/nonexistent")
	if ok {
		t.Errorf("ReadRef() returned true for non-existent ref, sha = %q", sha)
	}
	if sha != "" {
		t.Errorf("ReadRef() SHA = %q, want empty for non-existent ref", sha)
	}
}

func TestReadRefContent_Simple(t *testing.T) {
	repoDir := setupGitRepo(t)
	withDir(t, repoDir)

	ref := "refs/confvis/test-readcontent"
	content := []byte(`{"key":"value"}` + "\n")

	if err := WriteRef(ref, content, ""); err != nil {
		t.Fatalf("WriteRef() error = %v", err)
	}

	got, err := ReadRefContent(ref)
	if err != nil {
		t.Fatalf("ReadRefContent() error = %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("ReadRefContent() = %q, want %q", string(got), string(content))
	}
}

func TestReadRefContent_NotExists(t *testing.T) {
	repoDir := setupGitRepo(t)
	withDir(t, repoDir)

	got, err := ReadRefContent("refs/confvis/nonexistent")
	if err != nil {
		t.Fatalf("ReadRefContent() error = %v, want nil for non-existent ref", err)
	}
	if got != nil {
		t.Errorf("ReadRefContent() = %q, want nil for non-existent ref", string(got))
	}
}

// setupGitRepoWithCommit creates a temporary git repo with an initial commit.
func setupGitRepoWithCommit(t *testing.T, branch string) string {
	t.Helper()

	args := []string{"init"}
	if branch != "" {
		args = append(args, "-b", branch)
	}

	tmpDir := t.TempDir()

	cmd := exec.Command("git", args...)
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	testFile := tmpDir + "/test.txt"
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	return tmpDir
}

func TestGetCurrentCommit(t *testing.T) {
	repoDir := setupGitRepoWithCommit(t, "")
	withDir(t, repoDir)

	commit := GetCurrentCommit()
	if commit == "" {
		t.Error("expected non-empty commit hash")
	}
	if len(commit) != 40 {
		t.Errorf("expected 40-character commit hash, got %d characters: %s", len(commit), commit)
	}
}

func TestGetCurrentBranch(t *testing.T) {
	repoDir := setupGitRepoWithCommit(t, "main")
	withDir(t, repoDir)

	branch := GetCurrentBranch()
	if branch != "main" {
		t.Errorf("expected branch 'main', got %q", branch)
	}
}
