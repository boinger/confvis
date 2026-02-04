// Package baseline provides storage and comparison of confidence baselines.
package baseline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/boinger/confvis/internal/confidence"
)

// DefaultBaselineRef is the default git ref for storing baselines.
const DefaultBaselineRef = "refs/confvis/baseline"

// DefaultBaselineFile is the default file path for storing baselines.
const DefaultBaselineFile = ".confvis-baseline.json"

// gitCommandTimeout is the timeout for git commands.
const gitCommandTimeout = 30 * time.Second

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
		Commit:  GetCurrentCommit(),
		Branch:  GetCurrentBranch(),
	}
}

// GitRefStorage implements git ref-based baseline storage.
type GitRefStorage struct{}

// NewGitRefStorage creates a new GitRefStorage.
func NewGitRefStorage() *GitRefStorage {
	return &GitRefStorage{}
}

// Read reads a baseline from a git ref.
// Returns nil, nil if the ref doesn't exist (not an error condition).
func (g *GitRefStorage) Read(ref string) (*Baseline, error) {
	if !refExists(ref) {
		return nil, nil //nolint:nilnil // nil baseline with no error means "not found"
	}

	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "cat-file", "-p", ref)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("reading git ref %s: %w: %s", ref, err, stderr.String())
	}

	var baseline Baseline
	if err := json.Unmarshal(stdout.Bytes(), &baseline); err != nil {
		return nil, fmt.Errorf("parsing baseline from ref %s: %w", ref, err)
	}

	return &baseline, nil
}

// Write writes a baseline to a git ref.
func (g *GitRefStorage) Write(ref string, b *Baseline) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	content, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling baseline: %w", err)
	}
	content = append(content, '\n')

	// Create a blob object with the content
	cmd := exec.CommandContext(ctx, "git", "hash-object", "-w", "--stdin")
	cmd.Stdin = bytes.NewReader(content)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("creating git blob: %w: %s", err, stderr.String())
	}

	sha := strings.TrimSpace(stdout.String())

	// Update the ref to point to the blob
	cmd = exec.CommandContext(ctx, "git", "update-ref", ref, sha)
	stderr.Reset()
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("updating git ref %s: %w: %s", ref, err, stderr.String())
	}

	return nil
}

// FileStorage implements file-based baseline storage.
type FileStorage struct{}

// NewFileStorage creates a new FileStorage.
func NewFileStorage() *FileStorage {
	return &FileStorage{}
}

// Read reads a baseline from a file.
// Returns nil, nil if the file doesn't exist (not an error condition).
func (f *FileStorage) Read(path string) (*Baseline, error) {
	data, err := os.ReadFile(path)
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

	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("writing baseline file %s: %w", path, err)
	}

	return nil
}

// GetCurrentCommit returns the current git commit hash, or empty string if not in a repo.
func GetCurrentCommit() string {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return ""
	}

	return strings.TrimSpace(stdout.String())
}

// GetCurrentBranch returns the current git branch name, or empty string if not on a branch.
func GetCurrentBranch() string {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return ""
	}

	branch := strings.TrimSpace(stdout.String())
	// "HEAD" means detached head state
	if branch == "HEAD" {
		return ""
	}
	return branch
}

// IsGitRepo checks if the current directory is inside a git repository.
func IsGitRepo() bool {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// refExists checks if a git ref exists.
func refExists(ref string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "show-ref", "--verify", "--quiet", ref)
	return cmd.Run() == nil
}

// ReadFromGitRef reads a baseline from a git ref using the default storage.
func ReadFromGitRef(ref string) (*Baseline, error) {
	return NewGitRefStorage().Read(ref)
}

// WriteToGitRef writes a baseline to a git ref using the default storage.
func WriteToGitRef(ref string, b *Baseline) error {
	return NewGitRefStorage().Write(ref, b)
}

// ReadFromFile reads a baseline from a file using the default storage.
func ReadFromFile(path string) (*Baseline, error) {
	return NewFileStorage().Read(path)
}

// WriteToFile writes a baseline to a file using the default storage.
func WriteToFile(path string, b *Baseline) error {
	return NewFileStorage().Write(path, b)
}
