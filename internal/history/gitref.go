package history

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
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
	// Check if the ref exists - if not, return empty history (not an error)
	if !gitutil.RefExists(ref) {
		return &History{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitutil.CommandTimeout)
	defer cancel()

	// Read the content from the ref
	cmd := exec.CommandContext(ctx, gitutil.ResolveGitPath(), "cat-file", "-p", ref)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("reading git ref %s: %w: %s", ref, err, stderr.String())
	}

	// Parse the JSON lines content
	return parseHistoryContent(stdout.String())
}

// WriteToRef writes history to a git ref.
// Creates or updates the ref to point to a blob containing the history.
func (g *GitRefStorage) WriteToRef(ref string, h *History) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitutil.CommandTimeout)
	defer cancel()

	// Serialize history to JSON lines
	content, err := serializeHistory(h)
	if err != nil {
		return fmt.Errorf("serializing history: %w", err)
	}

	// Create a blob object with the content
	cmd := exec.CommandContext(ctx, gitutil.ResolveGitPath(), "hash-object", "-w", "--stdin")
	cmd.Stdin = strings.NewReader(content)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("creating git blob: %w: %s", err, stderr.String())
	}

	sha := strings.TrimSpace(stdout.String())

	// Update the ref to point to the blob
	cmd = exec.CommandContext(ctx, gitutil.ResolveGitPath(), "update-ref", ref, sha)
	stderr.Reset()
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("updating git ref %s: %w: %s", ref, err, stderr.String())
	}

	return nil
}

// AppendToRef reads existing history from a ref, appends an entry, and writes back.
func (g *GitRefStorage) AppendToRef(ref string, entry Entry) error {
	// Read existing history
	h, err := g.ReadFromRef(ref)
	if err != nil {
		return fmt.Errorf("reading existing history: %w", err)
	}

	// Append the new entry
	h.Entries = append(h.Entries, entry)

	// Write back
	return g.WriteToRef(ref, h)
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
