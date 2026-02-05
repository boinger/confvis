package grype

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/scoring"
)

const sourceName = "grype"

// Environment variable names for configuration.
const (
	EnvCommand = "GRYPE_CMD"
)

// Source implements the sources.Source interface for Grype.
type Source struct{}

func init() {
	sources.Register(&Source{})
}

// Name returns the source identifier.
func (s *Source) Name() string {
	return sourceName
}

// Fetch runs Grype on the specified target and converts results to a confidence report.
func (s *Source) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
	command := sources.ResolveCommand(opts, "grype-cmd", EnvCommand)

	// Target is provided via Project field (matches CLI pattern of -p flag)
	// Can be a path, container image, or SBOM file
	target := opts.Project
	if target == "" {
		target = "."
	}

	// Create client
	client := NewClient(command)

	// Run scan
	report, err := client.Scan(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("scanning with grype: %w", err)
	}

	// Aggregate counts
	counts := countFromMatches(report.Matches)

	// Determine title
	title := opts.Title
	if title == "" {
		// Try to make a nice title from the target
		title = deriveTitle(target)
	}

	// Build factors with severity-based scoring
	factors := scoring.BuildVulnFactors(
		counts,
		scoring.DefaultPenalties(),
		scoring.DefaultWeights(),
		"",
	)

	return scoring.BuildReport(title, sourceName, opts.Threshold, factors), nil
}

// deriveTitle creates a title from the target path or image name.
func deriveTitle(target string) string {
	// If it looks like a container image (contains : for tag), return as-is
	if looksLikeContainerImage(target) {
		return target
	}

	// For filesystem paths, use the directory name
	absPath, err := filepath.Abs(target)
	if err != nil {
		return target
	}
	return filepath.Base(absPath)
}

// looksLikeContainerImage returns true if target looks like a container image reference.
// Container images typically have a : for the tag or don't start with . or /
func looksLikeContainerImage(target string) bool {
	if len(target) == 0 {
		return false
	}
	// If starts with . or / it's a filesystem path
	if target[0] == '.' || target[0] == '/' {
		return false
	}
	// If contains : it's likely an image with tag
	for _, c := range target {
		if c == ':' {
			return true
		}
	}
	// Single word without path separators could be an image
	return filepath.Dir(target) == "."
}
