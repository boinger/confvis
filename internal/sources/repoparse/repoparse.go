// Package repoparse provides utilities for parsing repository identifiers.
package repoparse

import (
	"fmt"
	"strings"
)

// Parse splits "owner/repo" into separate parts.
// Returns an error if the format is invalid.
func Parse(project string) (owner, repo string, err error) {
	if project == "" {
		return "", "", fmt.Errorf("repository required: use --project owner/repo")
	}
	parts := strings.SplitN(project, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repository format %q: expected owner/repo", project)
	}
	return parts[0], parts[1], nil
}

// MustParse is like Parse but returns empty strings on error.
// Useful for URL builders where an empty result is acceptable.
func MustParse(project string) (owner, repo string) {
	owner, repo, _ = Parse(project)
	return owner, repo
}
