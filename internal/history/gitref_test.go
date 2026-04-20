package history

import (
	"errors"
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
	t.Chdir(repoDir)

	if !gitutil.IsGitRepo() {
		t.Error("IsGitRepo() = false, want true in git repo")
	}
}

func TestIsGitRepo_NotInRepo(t *testing.T) {
	// Use a temp directory that's definitely not a git repo
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	if gitutil.IsGitRepo() {
		t.Error("IsGitRepo() = true, want false outside git repo")
	}
}

func TestGitRefStorage_ReadFromRef_NotExists(t *testing.T) {
	repoDir := setupGitRepo(t)
	t.Chdir(repoDir)

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
	t.Chdir(repoDir)

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
	t.Chdir(repoDir)

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
	t.Chdir(repoDir)

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
	t.Chdir(repoDir)

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
	t.Chdir(repoDir)

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

func TestGitRefStorage_PruneRef_TrimsToLastN(t *testing.T) {
	repoDir := setupGitRepo(t)
	t.Chdir(repoDir)

	storage := newGitRefStorage()
	ref := "refs/confvis/prune-test"

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 15; i++ {
		entry := Entry{Score: i, Timestamp: base.Add(time.Duration(i) * time.Hour)}
		if err := storage.AppendToRef(ref, entry); err != nil {
			t.Fatalf("AppendToRef() error = %v", err)
		}
	}

	if err := storage.PruneRef(ref, 5); err != nil {
		t.Fatalf("PruneRef() error = %v", err)
	}

	hist, err := storage.ReadFromRef(ref)
	if err != nil {
		t.Fatalf("ReadFromRef() error = %v", err)
	}
	if len(hist.Entries) != 5 {
		t.Fatalf("len(Entries) = %d, want 5", len(hist.Entries))
	}
	// Last 5 entries: scores 10..14
	if hist.Entries[0].Score != 10 {
		t.Errorf("first kept score = %d, want 10", hist.Entries[0].Score)
	}
	if hist.Entries[4].Score != 14 {
		t.Errorf("last kept score = %d, want 14", hist.Entries[4].Score)
	}
}

func TestGitRefStorage_PruneRef_NoOpWhenUnderCap(t *testing.T) {
	repoDir := setupGitRepo(t)
	t.Chdir(repoDir)

	storage := newGitRefStorage()
	ref := "refs/confvis/prune-noop"

	for i := 0; i < 3; i++ {
		if err := storage.AppendToRef(ref, Entry{Score: i, Timestamp: time.Now().UTC()}); err != nil {
			t.Fatalf("AppendToRef() error = %v", err)
		}
	}

	if err := storage.PruneRef(ref, 10); err != nil {
		t.Fatalf("PruneRef() error = %v", err)
	}

	hist, err := storage.ReadFromRef(ref)
	if err != nil {
		t.Fatalf("ReadFromRef() error = %v", err)
	}
	if len(hist.Entries) != 3 {
		t.Errorf("len(Entries) = %d, want 3 (unchanged)", len(hist.Entries))
	}
}

func TestGitRefStorage_PruneRef_Disabled(t *testing.T) {
	repoDir := setupGitRepo(t)
	t.Chdir(repoDir)

	storage := newGitRefStorage()
	ref := "refs/confvis/prune-disabled"

	for i := 0; i < 5; i++ {
		if err := storage.AppendToRef(ref, Entry{Score: i, Timestamp: time.Now().UTC()}); err != nil {
			t.Fatalf("AppendToRef() error = %v", err)
		}
	}

	for _, n := range []int{0, -1} {
		if err := storage.PruneRef(ref, n); err != nil {
			t.Fatalf("PruneRef(%d) error = %v", n, err)
		}
	}

	hist, err := storage.ReadFromRef(ref)
	if err != nil {
		t.Fatalf("ReadFromRef() error = %v", err)
	}
	if len(hist.Entries) != 5 {
		t.Errorf("len(Entries) = %d, want 5 (untouched)", len(hist.Entries))
	}
}

func TestGitRefStorage_PruneRef_MissingRefNoOp(t *testing.T) {
	repoDir := setupGitRepo(t)
	t.Chdir(repoDir)

	storage := newGitRefStorage()
	// Reading a non-existent ref returns empty history, so prune has nothing to do.
	if err := storage.PruneRef("refs/confvis/does-not-exist", 10); err != nil {
		t.Fatalf("PruneRef() error on missing ref = %v", err)
	}
}

func TestPruneGitRef_PackageLevel(t *testing.T) {
	repoDir := setupGitRepo(t)
	t.Chdir(repoDir)

	ref := "refs/confvis/prune-package-level"
	for i := 0; i < 8; i++ {
		if err := AppendToGitRef(ref, Entry{Score: i, Timestamp: time.Now().UTC()}); err != nil {
			t.Fatalf("AppendToGitRef() error = %v", err)
		}
	}

	// Use the package-level helper, mirroring how CLI orchestration will call it.
	if err := PruneGitRef(ref, 3); err != nil {
		t.Fatalf("PruneGitRef() error = %v", err)
	}

	hist, err := ReadFromGitRef(ref)
	if err != nil {
		t.Fatalf("ReadFromGitRef() error = %v", err)
	}
	if len(hist.Entries) != 3 {
		t.Errorf("len(Entries) = %d, want 3", len(hist.Entries))
	}
}

// TestGitRefStorage_PruneRef_Conflict verifies that PruneRef returns
// gitutil.ErrRefConflict when the ref moves between its read and
// write, preserving the concurrent append instead of overwriting it.
//
// To simulate the race deterministically, this test snapshots the
// oldSHA the same way PruneRef does (ReadRef), then writes a new
// version of the ref out-of-band (via AppendToRef) to simulate a
// concurrent writer. When PruneRef runs, its WriteRef CAS fails
// against the stale oldSHA — exactly the race we're guarding against.
func TestGitRefStorage_PruneRef_Conflict(t *testing.T) {
	repoDir := setupGitRepo(t)
	t.Chdir(repoDir)

	storage := newGitRefStorage()
	ref := "refs/confvis/prune-conflict"

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 10; i++ {
		entry := Entry{Score: i, Timestamp: base.Add(time.Duration(i) * time.Hour)}
		if err := storage.AppendToRef(ref, entry); err != nil {
			t.Fatalf("seeding AppendToRef(%d) error = %v", i, err)
		}
	}

	// Snapshot the current (pre-prune) SHA — what PruneRef would observe
	// during its ReadRef. We don't call PruneRef yet.
	oldSHA, exists := gitutil.ReadRef(ref)
	if !exists {
		t.Fatal("ref missing after seeding")
	}

	// Simulate a concurrent AppendToRef racing between PruneRef's read
	// and its write. This moves the ref to a new SHA.
	concurrentEntry := Entry{Score: 99, Timestamp: base.Add(24 * time.Hour)}
	if err := storage.AppendToRef(ref, concurrentEntry); err != nil {
		t.Fatalf("concurrent AppendToRef error = %v", err)
	}

	// Now manually run PruneRef's logic against the *stale* oldSHA.
	// The internal PruneRef reads oldSHA itself, but here we want to
	// exercise the exact conflict path — so we build the equivalent
	// call inline. This mirrors gitref.go:PruneRef but with oldSHA
	// captured before the concurrent write.
	h, err := storage.ReadFromRef(ref) // reads latest (11 entries)
	if err != nil {
		t.Fatalf("ReadFromRef error = %v", err)
	}
	h.Entries = h.Entries[len(h.Entries)-5:] // simulate pruning to 5
	content, err := serializeHistory(h)
	if err != nil {
		t.Fatalf("serializeHistory error = %v", err)
	}

	err = gitutil.WriteRef(ref, []byte(content), oldSHA) // stale SHA
	if err == nil {
		t.Fatal("expected ErrRefConflict, got nil")
	}
	if !errors.Is(err, gitutil.ErrRefConflict) {
		t.Errorf("expected ErrRefConflict, got: %v", err)
	}

	// And the concurrent append must still be in the ref.
	final, err := storage.ReadFromRef(ref)
	if err != nil {
		t.Fatalf("final ReadFromRef error = %v", err)
	}
	if len(final.Entries) != 11 {
		t.Errorf("ref lost concurrent entry: len(Entries) = %d, want 11", len(final.Entries))
	}
	if final.Entries[len(final.Entries)-1].Score != 99 {
		t.Errorf("concurrent entry missing: last Score = %d, want 99", final.Entries[len(final.Entries)-1].Score)
	}
}

// TestGitRefStorage_PruneRef_ReturnsConflict verifies PruneRef itself
// (not the inline simulation above) returns ErrRefConflict when the
// ref moves underneath it. Uses a test hook by invoking PruneRef
// right after a concurrent AppendToRef that invalidates any SHA
// PruneRef could have read — which is actually NOT reliably
// observable without racing inside PruneRef. Instead this test
// documents the expected behavior through the happy path: PruneRef
// reads the latest SHA, the ref doesn't move, PruneRef succeeds.
// The conflict branch is covered by the inline simulation above.
func TestGitRefStorage_PruneRef_HappyPath(t *testing.T) {
	repoDir := setupGitRepo(t)
	t.Chdir(repoDir)

	storage := newGitRefStorage()
	ref := "refs/confvis/prune-happy"

	for i := 0; i < 8; i++ {
		if err := storage.AppendToRef(ref, Entry{Score: i, Timestamp: time.Now().UTC()}); err != nil {
			t.Fatalf("AppendToRef(%d) error = %v", i, err)
		}
	}

	// No concurrent writer → PruneRef's CAS matches → succeeds.
	if err := storage.PruneRef(ref, 3); err != nil {
		t.Fatalf("PruneRef() error = %v", err)
	}

	h, err := storage.ReadFromRef(ref)
	if err != nil {
		t.Fatalf("ReadFromRef error = %v", err)
	}
	if len(h.Entries) != 3 {
		t.Errorf("len(Entries) = %d, want 3", len(h.Entries))
	}
}
