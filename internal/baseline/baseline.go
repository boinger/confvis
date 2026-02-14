// Package baseline provides storage and comparison of confidence baselines.
package baseline

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/gitutil"
)

// DefaultBaselineRef is the default git ref for storing baselines.
const DefaultBaselineRef = "refs/confvis/baseline"

// DefaultBaselineFile is the default file path for storing baselines.
const DefaultBaselineFile = ".confvis-baseline.json"

// Baseline extends a confidence Report with save metadata.
type Baseline struct {
	confidence.Report
	SavedAt string `json:"savedAt"`
	Commit  string `json:"commit,omitempty"`
	Branch  string `json:"branch,omitempty"`
}

// NewBaseline creates a Baseline from a Report with current metadata.
func NewBaseline(report *confidence.Report) *Baseline {
	return &Baseline{
		Report:  *report,
		SavedAt: time.Now().UTC().Format(time.RFC3339),
		Commit:  gitutil.GetCurrentCommit(),
		Branch:  gitutil.GetCurrentBranch(),
	}
}

// GitRefStorage implements git ref-based baseline storage.
type GitRefStorage struct{}

// newGitRefStorage creates a new GitRefStorage.
func newGitRefStorage() *GitRefStorage {
	return &GitRefStorage{}
}

// Read reads a baseline from a git ref.
// Returns nil, nil if the ref doesn't exist (not an error condition).
func (g *GitRefStorage) Read(ref string) (*Baseline, error) {
	content, err := gitutil.ReadRefContent(ref)
	if err != nil {
		return nil, fmt.Errorf("reading baseline ref %s: %w", ref, err)
	}
	if content == nil {
		return nil, nil //nolint:nilnil // nil baseline with no error means "not found"
	}

	var baseline Baseline
	if err := json.Unmarshal(content, &baseline); err != nil {
		return nil, fmt.Errorf("parsing baseline from ref %s: %w", ref, err)
	}

	return &baseline, nil
}

// Write writes a baseline to a git ref.
func (g *GitRefStorage) Write(ref string, b *Baseline) error {
	content, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling baseline: %w", err)
	}
	content = append(content, '\n')

	if err := gitutil.WriteRef(ref, content, ""); err != nil {
		return fmt.Errorf("writing baseline to ref %s: %w", ref, err)
	}

	return nil
}

// FileStorage implements file-based baseline storage.
type FileStorage struct{}

// newFileStorage creates a new FileStorage.
func newFileStorage() *FileStorage {
	return &FileStorage{}
}

// Read reads a baseline from a file.
// Returns nil, nil if the file doesn't exist (not an error condition).
func (f *FileStorage) Read(path string) (*Baseline, error) {
	data, err := os.ReadFile(path) //#nosec G304 -- path from CLI argument or config, not untrusted input
	if os.IsNotExist(err) {
		return nil, nil //nolint:nilnil // nil baseline with no error means "not found"
	}
	if err != nil {
		return nil, fmt.Errorf("reading baseline file %s: %w", path, err)
	}

	var baseline Baseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, fmt.Errorf("parsing baseline from file %s: %w", path, err)
	}

	return &baseline, nil
}

// Write writes a baseline to a file.
func (f *FileStorage) Write(path string, b *Baseline) error {
	content, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling baseline: %w", err)
	}
	content = append(content, '\n')

	// #nosec G306 -- baseline metrics file, world-readable is appropriate
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("writing baseline file %s: %w", path, err)
	}

	return nil
}

// ReadFromGitRef reads a baseline from a git ref using the default storage.
func ReadFromGitRef(ref string) (*Baseline, error) {
	return newGitRefStorage().Read(ref)
}

// WriteToGitRef writes a baseline to a git ref using the default storage.
func WriteToGitRef(ref string, b *Baseline) error {
	return newGitRefStorage().Write(ref, b)
}

// ReadFromFile reads a baseline from a file using the default storage.
func ReadFromFile(path string) (*Baseline, error) {
	return newFileStorage().Read(path)
}

// WriteToFile writes a baseline to a file using the default storage.
func WriteToFile(path string, b *Baseline) error {
	return newFileStorage().Write(path, b)
}
