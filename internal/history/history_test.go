package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadFile_NotExists(t *testing.T) {
	hist, err := ReadFile("/nonexistent/path/file.jsonl")
	if err != nil {
		t.Fatalf("ReadFile() error = %v, want nil for non-existent file", err)
	}
	if len(hist.Entries) != 0 {
		t.Errorf("ReadFile() entries = %d, want 0 for non-existent file", len(hist.Entries))
	}
}

func TestReadFile_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.jsonl")

	content := `{"score": 80, "timestamp": "2024-01-01T10:00:00Z"}
{"score": 85, "timestamp": "2024-01-02T10:00:00Z"}
{"score": 90, "timestamp": "2024-01-03T10:00:00Z"}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	hist, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if len(hist.Entries) != 3 {
		t.Errorf("ReadFile() entries = %d, want 3", len(hist.Entries))
	}

	if hist.Entries[0].Score != 80 {
		t.Errorf("Entry[0].Score = %d, want 80", hist.Entries[0].Score)
	}
	if hist.Entries[2].Score != 90 {
		t.Errorf("Entry[2].Score = %d, want 90", hist.Entries[2].Score)
	}
}

func TestReadFile_EmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.jsonl")

	content := `{"score": 80, "timestamp": "2024-01-01T10:00:00Z"}

{"score": 85, "timestamp": "2024-01-02T10:00:00Z"}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	hist, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if len(hist.Entries) != 2 {
		t.Errorf("ReadFile() entries = %d, want 2 (skipping empty lines)", len(hist.Entries))
	}
}

func TestReadFile_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.jsonl")

	content := `{"score": 80, "timestamp": "2024-01-01T10:00:00Z"}
not valid json
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	_, err := ReadFile(path)
	if err == nil {
		t.Fatal("ReadFile() expected error for invalid JSON")
	}
}

func TestAppendToFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.jsonl")

	entry := Entry{
		Score:     85,
		Timestamp: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	if err := AppendToFile(path, entry); err != nil {
		t.Fatalf("AppendToFile() error = %v", err)
	}

	// Read back
	hist, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if len(hist.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(hist.Entries))
	}
	if hist.Entries[0].Score != 85 {
		t.Errorf("Entry.Score = %d, want 85", hist.Entries[0].Score)
	}
}

func TestAppendToFile_Append(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.jsonl")

	// Write initial entry
	entry1 := Entry{Score: 80, Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)}
	if err := AppendToFile(path, entry1); err != nil {
		t.Fatalf("AppendToFile() error = %v", err)
	}

	// Append second entry
	entry2 := Entry{Score: 85, Timestamp: time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)}
	if err := AppendToFile(path, entry2); err != nil {
		t.Fatalf("AppendToFile() error = %v", err)
	}

	// Read back
	hist, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if len(hist.Entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(hist.Entries))
	}
}

func TestHistory_Last(t *testing.T) {
	hist := &History{
		Entries: []Entry{
			{Score: 70},
			{Score: 75},
			{Score: 80},
			{Score: 85},
			{Score: 90},
		},
	}

	tests := []struct {
		n       int
		want    int
		wantLen int
	}{
		{3, 80, 3},  // Last 3: [80, 85, 90]
		{5, 70, 5},  // Last 5: all
		{10, 70, 5}, // More than available: all
		{0, 0, 0},   // Zero: empty
		{-1, 0, 0},  // Negative: empty
	}

	for _, tt := range tests {
		got := hist.Last(tt.n)
		if len(got) != tt.wantLen {
			t.Errorf("Last(%d) len = %d, want %d", tt.n, len(got), tt.wantLen)
		}
		if tt.wantLen > 0 && got[0].Score != tt.want {
			t.Errorf("Last(%d)[0].Score = %d, want %d", tt.n, got[0].Score, tt.want)
		}
	}
}

func TestHistory_Scores(t *testing.T) {
	hist := &History{
		Entries: []Entry{
			{Score: 70},
			{Score: 80},
			{Score: 90},
		},
	}

	scores := hist.Scores()
	if len(scores) != 3 {
		t.Fatalf("Scores() len = %d, want 3", len(scores))
	}

	expected := []int{70, 80, 90}
	for i, want := range expected {
		if scores[i] != want {
			t.Errorf("Scores()[%d] = %d, want %d", i, scores[i], want)
		}
	}
}

func TestNewEntry(t *testing.T) {
	before := time.Now().UTC()
	entry := NewEntry(85)
	after := time.Now().UTC()

	if entry.Score != 85 {
		t.Errorf("NewEntry().Score = %d, want 85", entry.Score)
	}

	if entry.Timestamp.Before(before) || entry.Timestamp.After(after) {
		t.Errorf("NewEntry().Timestamp outside expected range")
	}
}
