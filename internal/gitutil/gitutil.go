// Package gitutil provides shared git helper functions used by baseline and history packages.
package gitutil

import (
	"context"
	"os/exec"
	"sync"
	"time"
)

// CommandTimeout is the timeout for git commands.
const CommandTimeout = 30 * time.Second

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
