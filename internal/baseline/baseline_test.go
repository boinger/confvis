package baseline

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/gitutil"
)

// intPtrT is a test helper that returns a pointer to an int.
func intPtrT(i int) *int { return &i }

func TestNewBaseline(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test Report",
		Score:     intPtrT(85),
		Threshold: 75,
	}

	baseline := NewBaseline(report)

	if baseline.Title != "Test Report" {
		t.Errorf("expected title 'Test Report', got %q", baseline.Title)
	}
	if baseline.ScoreValue() != 85 {
		t.Errorf("expected score 85, got %d", baseline.Score)
	}
	if baseline.Threshold != 75 {
		t.Errorf("expected threshold 75, got %d", baseline.Threshold)
	}
	if baseline.SavedAt == "" {
		t.Error("expected SavedAt to be set")
	}

	// Verify SavedAt is valid RFC3339
	if _, err := time.Parse(time.RFC3339, baseline.SavedAt); err != nil {
		t.Errorf("SavedAt is not valid RFC3339: %v", err)
	}
}

func TestFileStorage_ReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "baseline.json")

	storage := newFileStorage()

	// Read non-existent file should return nil
	b, err := storage.Read(path)
	if err != nil {
		t.Fatalf("unexpected error reading non-existent file: %v", err)
	}
	if b != nil {
		t.Error("expected nil for non-existent file")
	}

	// Write baseline
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtrT(80),
		Threshold: 70,
	}
	baseline := NewBaseline(report)
	baseline.Commit = "abc123"
	baseline.Branch = "main"

	if err := storage.Write(path, baseline); err != nil {
		t.Fatalf("failed to write baseline: %v", err)
	}

	// Read it back
	read, err := storage.Read(path)
	if err != nil {
		t.Fatalf("failed to read baseline: %v", err)
	}

	if read.Title != "Test" {
		t.Errorf("expected title 'Test', got %q", read.Title)
	}
	if read.ScoreValue() != 80 {
		t.Errorf("expected score 80, got %d", read.Score)
	}
	if read.Commit != "abc123" {
		t.Errorf("expected commit 'abc123', got %q", read.Commit)
	}
	if read.Branch != "main" {
		t.Errorf("expected branch 'main', got %q", read.Branch)
	}

	// Verify file content is pretty-printed
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if !strings.Contains(string(content), "  ") {
		t.Error("expected indented JSON output")
	}
}

func TestFileStorage_ReadInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.json")

	// Write invalid JSON
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	storage := newFileStorage()
	_, err := storage.Read(path)
	if err == nil {
		t.Error("expected error reading invalid JSON")
	}
}

func TestGitRefStorage_ReadWrite(t *testing.T) {
	// Create a temporary git repo
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git user
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

	// Change to temp dir for git operations
	oldDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	storage := newGitRefStorage()
	ref := "refs/confvis/test-baseline"

	// Read non-existent ref should return nil
	b, err := storage.Read(ref)
	if err != nil {
		t.Fatalf("unexpected error reading non-existent ref: %v", err)
	}
	if b != nil {
		t.Error("expected nil for non-existent ref")
	}

	// Write baseline
	report := &confidence.Report{
		Title:     "Git Test",
		Score:     intPtrT(90),
		Threshold: 75,
	}
	baseline := NewBaseline(report)
	baseline.Commit = "def456"
	baseline.Branch = "feature"

	if err := storage.Write(ref, baseline); err != nil {
		t.Fatalf("failed to write baseline: %v", err)
	}

	// Verify ref was created
	cmd = exec.Command("git", "show-ref", "--verify", ref)
	if err := cmd.Run(); err != nil {
		t.Error("ref should exist after write")
	}

	// Read it back
	read, err := storage.Read(ref)
	if err != nil {
		t.Fatalf("failed to read baseline: %v", err)
	}

	if read.Title != "Git Test" {
		t.Errorf("expected title 'Git Test', got %q", read.Title)
	}
	if read.ScoreValue() != 90 {
		t.Errorf("expected score 90, got %d", read.Score)
	}
	if read.Commit != "def456" {
		t.Errorf("expected commit 'def456', got %q", read.Commit)
	}
}

func TestIsGitRepo(t *testing.T) {
	// Test in a git repo
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	oldDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	if !gitutil.IsGitRepo() {
		t.Error("expected IsGitRepo to return true in a git repo")
	}

	// Test outside a git repo
	nonGitDir := t.TempDir()
	if err := os.Chdir(nonGitDir); err != nil {
		t.Fatalf("failed to change to non-git dir: %v", err)
	}

	if gitutil.IsGitRepo() {
		t.Error("expected IsGitRepo to return false outside a git repo")
	}

	_ = os.Chdir(oldDir)
}

func TestBaselineJSONSerialization(t *testing.T) {
	report := &confidence.Report{
		Title:     "Serialization Test",
		Score:     intPtrT(75),
		Threshold: 70,
		Factors: []confidence.Factor{
			{Name: "Test", Score: 80, Weight: 100},
		},
	}

	baseline := NewBaseline(report)
	baseline.Commit = "abc123def456"
	baseline.Branch = "develop"

	// Serialize
	data, err := json.Marshal(baseline)
	if err != nil {
		t.Fatalf("failed to marshal baseline: %v", err)
	}

	// Verify expected fields are present
	dataStr := string(data)
	expectedFields := []string{
		`"title":"Serialization Test"`,
		`"score":75`,
		`"threshold":70`,
		`"savedAt"`,
		`"commit":"abc123def456"`,
		`"branch":"develop"`,
		`"factors"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(dataStr, field) {
			t.Errorf("expected JSON to contain %q, got: %s", field, dataStr)
		}
	}

	// Deserialize
	var restored Baseline
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal baseline: %v", err)
	}

	if restored.Title != "Serialization Test" {
		t.Errorf("expected title 'Serialization Test', got %q", restored.Title)
	}
	if len(restored.Factors) != 1 {
		t.Errorf("expected 1 factor, got %d", len(restored.Factors))
	}
}

func TestReadFromGitRef_Convenience(t *testing.T) {
	// Create a temporary git repo
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	oldDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	// Non-existent ref
	b, err := ReadFromGitRef("refs/confvis/nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b != nil {
		t.Error("expected nil for non-existent ref")
	}
}

func TestReadFromFile_Convenience(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	// Non-existent file
	b, err := ReadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b != nil {
		t.Error("expected nil for non-existent file")
	}
}

func TestWriteToFile_Convenience(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	report := &confidence.Report{
		Title:     "Convenience Test",
		Score:     intPtrT(85),
		Threshold: 75,
	}
	baseline := NewBaseline(report)

	if err := WriteToFile(path, baseline); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected file to be created")
	}
}

func TestWriteToGitRef_Convenience(t *testing.T) {
	// Create a temporary git repo
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git user
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

	oldDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	ref := "refs/confvis/write-test"
	report := &confidence.Report{
		Title:     "WriteToGitRef Test",
		Score:     intPtrT(88),
		Threshold: 80,
	}
	baseline := NewBaseline(report)

	if err := WriteToGitRef(ref, baseline); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Verify ref was created
	cmd = exec.Command("git", "show-ref", "--verify", ref)
	if err := cmd.Run(); err != nil {
		t.Error("ref should exist after write")
	}

	// Read back and verify
	read, err := ReadFromGitRef(ref)
	if err != nil {
		t.Fatalf("failed to read back: %v", err)
	}
	if read.Title != "WriteToGitRef Test" {
		t.Errorf("expected title 'WriteToGitRef Test', got %q", read.Title)
	}
}
