package history

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/boinger/confvis/internal/gitutil"
)

// setupGitRepo creates a temporary git repository for testing.
// Returns the path to the repo and a cleanup function.
func setupGitRepo(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git user (required for some operations)
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

func TestIsGitRepo_InRepo(t *testing.T) {
	repoDir := setupGitRepo(t)

	// Change to repo directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("changing to repo directory: %v", err)
	}

	if !gitutil.IsGitRepo() {
		t.Error("IsGitRepo() = false, want true in git repo")
	}
}

func TestIsGitRepo_NotInRepo(t *testing.T) {
	// Use a temp directory that's definitely not a git repo
	tmpDir := t.TempDir()

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp directory: %v", err)
	}

	if gitutil.IsGitRepo() {
		t.Error("IsGitRepo() = true, want false outside git repo")
	}
}

func TestGitRefStorage_ReadFromRef_NotExists(t *testing.T) {
	repoDir := setupGitRepo(t)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("changing to repo directory: %v", err)
	}

	storage := newGitRefStorage()
	hist, err := storage.ReadFromRef("refs/confvis/nonexistent")
	if err != nil {
		t.Fatalf("ReadFromRef() error = %v", err)
	}

	if len(hist.Entries) != 0 {
		t.Errorf("ReadFromRef() entries = %d, want 0 for non-existent ref", len(hist.Entries))
	}
}

func TestGitRefStorage_WriteAndRead(t *testing.T) {
	repoDir := setupGitRepo(t)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("changing to repo directory: %v", err)
	}

	storage := newGitRefStorage()

	// Create history with entries
	hist := &History{
		Entries: []Entry{
			{Score: 80, Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
			{Score: 85, Timestamp: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)},
			{Score: 90, Timestamp: time.Date(2024, 1, 3, 10, 0, 0, 0, time.UTC)},
		},
	}

	ref := "refs/confvis/test-history"

	// Write history
	if err := storage.WriteToRef(ref, hist); err != nil {
		t.Fatalf("WriteToRef() error = %v", err)
	}

	// Verify ref exists
	cmd := exec.Command("git", "show-ref", "--verify", ref)
	if err := cmd.Run(); err != nil {
		t.Fatalf("ref %s should exist after WriteToRef", ref)
	}

	// Read back
	readHist, err := storage.ReadFromRef(ref)
	if err != nil {
		t.Fatalf("ReadFromRef() error = %v", err)
	}

	if len(readHist.Entries) != 3 {
		t.Fatalf("ReadFromRef() entries = %d, want 3", len(readHist.Entries))
	}

	if readHist.Entries[0].Score != 80 {
		t.Errorf("Entry[0].Score = %d, want 80", readHist.Entries[0].Score)
	}
	if readHist.Entries[2].Score != 90 {
		t.Errorf("Entry[2].Score = %d, want 90", readHist.Entries[2].Score)
	}
}

func TestGitRefStorage_AppendToRef(t *testing.T) {
	repoDir := setupGitRepo(t)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("changing to repo directory: %v", err)
	}

	storage := newGitRefStorage()
	ref := "refs/confvis/append-test"

	// Append first entry to non-existent ref
	entry1 := Entry{Score: 80, Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)}
	if err := storage.AppendToRef(ref, entry1); err != nil {
		t.Fatalf("AppendToRef() first entry error = %v", err)
	}

	// Append second entry
	entry2 := Entry{Score: 85, Timestamp: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)}
	if err := storage.AppendToRef(ref, entry2); err != nil {
		t.Fatalf("AppendToRef() second entry error = %v", err)
	}

	// Read back
	hist, err := storage.ReadFromRef(ref)
	if err != nil {
		t.Fatalf("ReadFromRef() error = %v", err)
	}

	if len(hist.Entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(hist.Entries))
	}

	if hist.Entries[0].Score != 80 {
		t.Errorf("Entry[0].Score = %d, want 80", hist.Entries[0].Score)
	}
	if hist.Entries[1].Score != 85 {
		t.Errorf("Entry[1].Score = %d, want 85", hist.Entries[1].Score)
	}
}

func TestGitRefStorage_DefaultRef(t *testing.T) {
	if DefaultHistoryRef != "refs/confvis/history" {
		t.Errorf("DefaultHistoryRef = %q, want %q", DefaultHistoryRef, "refs/confvis/history")
	}
}

func TestGitRefStorage_WriteEmptyHistory(t *testing.T) {
	repoDir := setupGitRepo(t)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("changing to repo directory: %v", err)
	}

	storage := newGitRefStorage()
	ref := "refs/confvis/empty-test"

	// Write empty history
	hist := &History{}
	if err := storage.WriteToRef(ref, hist); err != nil {
		t.Fatalf("WriteToRef() empty history error = %v", err)
	}

	// Read back
	readHist, err := storage.ReadFromRef(ref)
	if err != nil {
		t.Fatalf("ReadFromRef() error = %v", err)
	}

	if len(readHist.Entries) != 0 {
		t.Errorf("entries = %d, want 0", len(readHist.Entries))
	}
}

func TestParseHistoryContent(t *testing.T) {
	content := `{"score":80,"timestamp":"2024-01-01T10:00:00Z"}
{"score":85,"timestamp":"2024-01-02T10:00:00Z"}
`

	hist, err := parseHistoryContent(content)
	if err != nil {
		t.Fatalf("parseHistoryContent() error = %v", err)
	}

	if len(hist.Entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(hist.Entries))
	}

	if hist.Entries[0].Score != 80 {
		t.Errorf("Entry[0].Score = %d, want 80", hist.Entries[0].Score)
	}
}

func TestParseHistoryContent_EmptyLines(t *testing.T) {
	content := `{"score":80,"timestamp":"2024-01-01T10:00:00Z"}

{"score":85,"timestamp":"2024-01-02T10:00:00Z"}
`

	hist, err := parseHistoryContent(content)
	if err != nil {
		t.Fatalf("parseHistoryContent() error = %v", err)
	}

	if len(hist.Entries) != 2 {
		t.Errorf("entries = %d, want 2 (ignoring empty lines)", len(hist.Entries))
	}
}

func TestParseHistoryContent_InvalidJSON(t *testing.T) {
	content := `{"score":80,"timestamp":"2024-01-01T10:00:00Z"}
not valid json
`

	_, err := parseHistoryContent(content)
	if err == nil {
		t.Error("parseHistoryContent() expected error for invalid JSON")
	}
}

func TestSerializeHistory(t *testing.T) {
	hist := &History{
		Entries: []Entry{
			{Score: 80, Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
			{Score: 85, Timestamp: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)},
		},
	}

	content, err := serializeHistory(hist)
	if err != nil {
		t.Fatalf("serializeHistory() error = %v", err)
	}

	// Parse back to verify round-trip
	parsed, err := parseHistoryContent(content)
	if err != nil {
		t.Fatalf("parseHistoryContent() error = %v", err)
	}

	if len(parsed.Entries) != 2 {
		t.Fatalf("round-trip entries = %d, want 2", len(parsed.Entries))
	}

	if parsed.Entries[0].Score != 80 {
		t.Errorf("round-trip Entry[0].Score = %d, want 80", parsed.Entries[0].Score)
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	repoDir := setupGitRepo(t)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("changing to repo directory: %v", err)
	}

	ref := "refs/confvis/pkg-level-test"

	// Test ReadFromGitRef on non-existent ref
	hist, err := ReadFromGitRef(ref)
	if err != nil {
		t.Fatalf("ReadFromGitRef() error = %v", err)
	}
	if len(hist.Entries) != 0 {
		t.Errorf("ReadFromGitRef() entries = %d, want 0", len(hist.Entries))
	}

	// Test WriteToGitRef
	hist = &History{
		Entries: []Entry{
			{Score: 75, Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)},
		},
	}
	if err := WriteToGitRef(ref, hist); err != nil {
		t.Fatalf("WriteToGitRef() error = %v", err)
	}

	// Test AppendToGitRef
	entry := Entry{Score: 80, Timestamp: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)}
	if err := AppendToGitRef(ref, entry); err != nil {
		t.Fatalf("AppendToGitRef() error = %v", err)
	}

	// Verify
	hist, err = ReadFromGitRef(ref)
	if err != nil {
		t.Fatalf("ReadFromGitRef() after append error = %v", err)
	}
	if len(hist.Entries) != 2 {
		t.Errorf("entries after append = %d, want 2", len(hist.Entries))
	}
}

func TestGitRefStorage_VerifyWithGitCatFile(t *testing.T) {
	repoDir := setupGitRepo(t)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("changing to repo directory: %v", err)
	}

	storage := newGitRefStorage()
	ref := "refs/confvis/verify-test"

	hist := &History{
		Entries: []Entry{
			{Score: 95, Timestamp: time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)},
		},
	}

	if err := storage.WriteToRef(ref, hist); err != nil {
		t.Fatalf("WriteToRef() error = %v", err)
	}

	// Use git cat-file directly to verify content
	cmd := exec.Command("git", "cat-file", "-p", ref)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git cat-file error = %v", err)
	}

	// Verify the content contains the score
	outputStr := string(output)
	if len(outputStr) == 0 {
		t.Error("git cat-file output should not be empty")
	}

	// Parse the output to verify it's valid JSON lines
	parsed, err := parseHistoryContent(outputStr)
	if err != nil {
		t.Fatalf("parsing git cat-file output: %v", err)
	}

	if len(parsed.Entries) != 1 || parsed.Entries[0].Score != 95 {
		t.Errorf("git cat-file content mismatch: got %v", parsed.Entries)
	}
}
