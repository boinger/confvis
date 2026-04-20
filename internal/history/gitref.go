package history

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/boinger/confvis/internal/gitutil"
)

// DefaultHistoryRef is the default git ref for storing history.
const DefaultHistoryRef = "refs/confvis/history"

// GitRefReader provides an interface for reading history from git refs.
// This allows for dependency injection in tests.
type GitRefReader interface {
	ReadFromRef(ref string) (*History, error)
}

// GitRefWriter provides an interface for writing history to git refs.
type GitRefWriter interface {
	WriteToRef(ref string, h *History) error
}

// GitRefStorage implements git ref-based history storage.
type GitRefStorage struct{}

// newGitRefStorage creates a new GitRefStorage.
func newGitRefStorage() *GitRefStorage {
	return &GitRefStorage{}
}

// ReadFromRef reads history from a git ref.
// Returns an empty history if the ref doesn't exist.
func (g *GitRefStorage) ReadFromRef(ref string) (*History, error) {
	content, err := gitutil.ReadRefContent(ref)
	if err != nil {
		return nil, fmt.Errorf("reading history ref %s: %w", ref, err)
	}
	if content == nil {
		return &History{}, nil
	}

	return parseHistoryContent(string(content))
}

// WriteToRef writes history to a git ref.
// Creates or updates the ref to point to a blob containing the history.
func (g *GitRefStorage) WriteToRef(ref string, h *History) error {
	content, err := serializeHistory(h)
	if err != nil {
		return fmt.Errorf("serializing history: %w", err)
	}

	if err := gitutil.WriteRef(ref, []byte(content), ""); err != nil {
		return fmt.Errorf("writing history to ref %s: %w", ref, err)
	}

	return nil
}

// AppendToRef reads existing history from a ref, appends an entry, and writes
// back atomically using compare-and-swap. Returns gitutil.ErrRefConflict if
// a concurrent writer modified the ref between read and write.
func (g *GitRefStorage) AppendToRef(ref string, entry Entry) error {
	oldSHA, exists := gitutil.ReadRef(ref)
	if !exists {
		oldSHA = gitutil.ZeroSHA
	}

	h, err := g.ReadFromRef(ref)
	if err != nil {
		return fmt.Errorf("reading existing history: %w", err)
	}

	h.Entries = append(h.Entries, entry)

	content, err := serializeHistory(h)
	if err != nil {
		return fmt.Errorf("serializing history: %w", err)
	}

	if err := gitutil.WriteRef(ref, []byte(content), oldSHA); err != nil {
		return fmt.Errorf("appending to history ref %s: %w", ref, err)
	}

	return nil
}

// parseHistoryContent parses JSON lines content into a History.
func parseHistoryContent(content string) (*History, error) {
	var entries []Entry

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("parsing line %d: %w", i+1, err)
		}
		entries = append(entries, entry)
	}

	return &History{Entries: entries}, nil
}

// serializeHistory serializes a History to JSON lines format.
func serializeHistory(h *History) (string, error) {
	var buf bytes.Buffer

	for _, entry := range h.Entries {
		data, err := json.Marshal(entry)
		if err != nil {
			return "", fmt.Errorf("marshaling entry: %w", err)
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}

	return buf.String(), nil
}

// PruneRef rewrites the history stored at ref to keep only the last
// maxEntries entries. A maxEntries of 0 or less disables pruning (no-op).
// If the ref doesn't exist or contains fewer entries than maxEntries,
// PruneRef is a no-op.
//
// PruneRef uses compare-and-swap via gitutil.WriteRef, matching the
// convention established by AppendToRef. If the ref moves between the
// read and the write (a concurrent AppendToRef), PruneRef returns
// gitutil.ErrRefConflict without losing the concurrent append. Callers
// should treat ErrRefConflict as benign — the concurrent append
// succeeded, and the next prune call will see the new state.
func (g *GitRefStorage) PruneRef(ref string, maxEntries int) error {
	if maxEntries <= 0 {
		return nil
	}

	oldSHA, exists := gitutil.ReadRef(ref)
	if !exists {
		return nil // nothing to prune
	}

	h, err := g.ReadFromRef(ref)
	if err != nil {
		return fmt.Errorf("reading history for prune: %w", err)
	}
	if len(h.Entries) <= maxEntries {
		return nil
	}

	h.Entries = h.Entries[len(h.Entries)-maxEntries:]
	content, err := serializeHistory(h)
	if err != nil {
		return fmt.Errorf("serializing pruned history: %w", err)
	}
	return gitutil.WriteRef(ref, []byte(content), oldSHA)
}

// Package-level functions for convenience

// ReadFromGitRef reads history from a git ref using the default storage.
func ReadFromGitRef(ref string) (*History, error) {
	return newGitRefStorage().ReadFromRef(ref)
}

// WriteToGitRef writes history to a git ref using the default storage.
func WriteToGitRef(ref string, h *History) error {
	return newGitRefStorage().WriteToRef(ref, h)
}

// AppendToGitRef appends an entry to history stored in a git ref.
func AppendToGitRef(ref string, entry Entry) error {
	return newGitRefStorage().AppendToRef(ref, entry)
}

// PruneGitRef trims history stored in a git ref to the last maxEntries entries.
func PruneGitRef(ref string, maxEntries int) error {
	return newGitRefStorage().PruneRef(ref, maxEntries)
}
