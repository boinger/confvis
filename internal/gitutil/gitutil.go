// Package gitutil provides shared git helper functions used by baseline and history packages.
package gitutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// CommandTimeout is the timeout for git commands.
const CommandTimeout = 30 * time.Second

// ZeroSHA is the 40-character zero SHA used to indicate "ref does not exist"
// in git update-ref's compare-and-swap mode.
const ZeroSHA = "0000000000000000000000000000000000000000"

// ErrRefConflict is returned when a compare-and-swap write fails because the
// ref's current value doesn't match the expected old SHA.
var ErrRefConflict = errors.New("git ref conflict: current value differs from expected")

var (
	gitPath     string
	gitPathOnce sync.Once
)

// ResolveGitPath finds the git executable path once and caches it.
// This satisfies security scanners that warn about PATH-based command execution.
func ResolveGitPath() string {
	gitPathOnce.Do(func() {
		path, err := exec.LookPath("git")
		if err != nil {
			// Fall back to "git" if LookPath fails (will use PATH at runtime)
			gitPath = "git"
		} else {
			gitPath = path
		}
	})
	return gitPath
}

// GetCurrentCommit returns the current git commit hash, or empty string if not in a repo.
func GetCurrentCommit() string {
	ctx, cancel := context.WithTimeout(context.Background(), CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, ResolveGitPath(), "rev-parse", "HEAD") //#nosec G204 -- git path resolved via exec.LookPath, args are internal
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return ""
	}

	return strings.TrimSpace(stdout.String())
}

// GetCurrentBranch returns the current git branch name, or empty string if not on a branch.
func GetCurrentBranch() string {
	ctx, cancel := context.WithTimeout(context.Background(), CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, ResolveGitPath(), "rev-parse", "--abbrev-ref", "HEAD") //#nosec G204 -- git path resolved via exec.LookPath, args are internal
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
	ctx, cancel := context.WithTimeout(context.Background(), CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, ResolveGitPath(), "rev-parse", "--git-dir") //#nosec G204 -- git path resolved via exec.LookPath, args are internal
	return cmd.Run() == nil
}

// RefExists checks if a git ref exists.
func RefExists(ref string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, ResolveGitPath(), "show-ref", "--verify", "--quiet", ref) //#nosec G204 -- git path resolved via exec.LookPath, args are internal
	return cmd.Run() == nil
}

// ReadRef resolves a git ref to its current SHA. Returns ("", false) if the
// ref does not exist.
func ReadRef(ref string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, ResolveGitPath(), "rev-parse", "--verify", ref) //#nosec G204 -- git path resolved via exec.LookPath, args are internal
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", false
	}

	return strings.TrimSpace(stdout.String()), true
}

// ReadRefContent reads the blob content stored at a git ref. Returns
// (nil, nil) if the ref does not exist.
func ReadRefContent(ref string) ([]byte, error) {
	if !RefExists(ref) {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), CommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, ResolveGitPath(), "cat-file", "-p", ref) //#nosec G204 -- git path resolved via exec.LookPath, args are internal
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("reading git ref %s: %w: %s", ref, err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// WriteRef writes content as a blob and updates a git ref to point to it.
//
// The oldSHA parameter controls compare-and-swap behavior:
//   - "" (empty): unconditional write (no CAS)
//   - ZeroSHA: create-only (fails if ref already exists)
//   - any other SHA: CAS (fails if ref's current value != oldSHA)
//
// Returns ErrRefConflict if the CAS check fails.
func WriteRef(ref string, content []byte, oldSHA string) error {
	ctx, cancel := context.WithTimeout(context.Background(), CommandTimeout)
	defer cancel()

	// Step 1: Create a blob object with the content
	cmd := exec.CommandContext(ctx, ResolveGitPath(), "hash-object", "-w", "--stdin") //#nosec G204 -- git path resolved via exec.LookPath, args are internal
	cmd.Stdin = bytes.NewReader(content)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("creating git blob: %w: %s", err, stderr.String())
	}

	newSHA := strings.TrimSpace(stdout.String())

	// Step 2: Update the ref, optionally with CAS
	args := []string{"update-ref", ref, newSHA}
	if oldSHA != "" {
		args = append(args, oldSHA)
	}

	cmd = exec.CommandContext(ctx, ResolveGitPath(), args...) //#nosec G204 -- git path resolved via exec.LookPath, args are internal
	stderr.Reset()
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if oldSHA != "" {
			// CAS failure — the ref's current value didn't match oldSHA
			return fmt.Errorf("%w: updating ref %s: %s", ErrRefConflict, ref, stderr.String())
		}
		return fmt.Errorf("updating git ref %s: %w: %s", ref, err, stderr.String())
	}

	return nil
}
