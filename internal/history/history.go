// Package history provides reading and writing of score history for sparkline generation.
package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Entry represents a single historical score entry.
type Entry struct {
	Score     int       `json:"score"`
	Timestamp time.Time `json:"timestamp"`
}

// History holds a list of historical score entries.
type History struct {
	Entries []Entry
}

// ReadFile reads history entries from a JSON lines file.
// Each line should be a valid JSON object with score and timestamp.
// Returns an empty history if the file doesn't exist.
func ReadFile(path string) (*History, error) {
	f, err := os.Open(path) //#nosec G304 -- path from CLI argument or config, not untrusted input
	if os.IsNotExist(err) {
		return &History{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("opening history file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("parsing line %d: %w", lineNum, err)
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading history file: %w", err)
	}

	return &History{Entries: entries}, nil
}

// AppendToFile appends a new entry to the history file.
// Creates the file if it doesn't exist.
func AppendToFile(path string, entry Entry) (err error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) // #nosec G302 G304 -- history log file, world-readable is appropriate, path from CLI argument
	if err != nil {
		return fmt.Errorf("opening history file for append: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing history file: %w", cerr)
		}
	}()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing entry: %w", err)
	}

	return nil
}

// Last returns the last n entries from the history.
// If n is greater than the number of entries, returns all entries.
func (h *History) Last(n int) []Entry {
	if n <= 0 || len(h.Entries) == 0 {
		return nil
	}
	if n >= len(h.Entries) {
		return h.Entries
	}
	return h.Entries[len(h.Entries)-n:]
}

// Scores returns just the score values from the entries.
func (h *History) Scores() []int {
	scores := make([]int, len(h.Entries))
	for i, e := range h.Entries {
		scores[i] = e.Score
	}
	return scores
}

// NewEntry creates a new history entry with the current timestamp.
func NewEntry(score int) Entry {
	return Entry{
		Score:     score,
		Timestamp: time.Now().UTC(),
	}
}

// Prune rewrites the history file at path to keep only the last
// maxEntries rows. A maxEntries of 0 or less disables pruning (no-op).
// If the file contains fewer than maxEntries rows, Prune is a no-op.
// Missing files are not an error; pruning skips gracefully.
//
// Prune is intended to be called from CLI orchestration after a successful
// AppendToFile. It is not called from AppendToFile itself to keep the hot
// write path simple and to let callers control rotation cadence.
func Prune(path string, maxEntries int) error {
	if maxEntries <= 0 {
		return nil
	}

	hist, err := ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading history for prune: %w", err)
	}
	if len(hist.Entries) <= maxEntries {
		return nil
	}

	kept := hist.Entries[len(hist.Entries)-maxEntries:]

	// Write to a sibling temp file, then rename for atomicity. Rename is
	// atomic on POSIX within the same directory, so readers never see a
	// partially-written file.
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file for prune: %w", err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup on any error path.
	defer func() {
		if tmp != nil {
			_ = tmp.Close()
		}
		// If rename succeeded, tmpName no longer exists; Remove errors are ignored.
		_ = os.Remove(tmpName)
	}()

	enc := json.NewEncoder(tmp)
	for _, e := range kept {
		if err := enc.Encode(e); err != nil {
			return fmt.Errorf("writing pruned entry: %w", err)
		}
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing pruned history: %w", err)
	}
	tmp = nil // closed; skip defer's Close

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replacing history file: %w", err)
	}
	return nil
}
