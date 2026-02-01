// Package history provides reading and writing of score history for sparkline generation.
package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
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
	f, err := os.Open(path)
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
func AppendToFile(path string, entry Entry) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening history file for append: %w", err)
	}
	defer func() { _ = f.Close() }()

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
