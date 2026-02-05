package semgrep

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/scoring"
)

const sourceName = "semgrep"

// Environment variable names for configuration.
const (
	EnvCommand = "SEMGREP_CMD"
)

// Source implements the sources.Source interface for Semgrep.
type Source struct {
	// Stdin can be set for testing or when reading from pipe
	Stdin io.Reader
}

func init() {
	sources.Register(&Source{})
}

// Name returns the source identifier.
func (s *Source) Name() string {
	return sourceName
}

// Fetch runs Semgrep on the specified path and converts results to a confidence report.
func (s *Source) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
	command := sources.ResolveCommand(opts, "semgrep-cmd", EnvCommand)

	// Resolve config (rules) from Extra options
	config := ""
	if opts.Extra != nil {
		config = opts.Extra["config"]
	}

	// Check if we should read from stdin
	fromStdin := false
	if opts.Extra != nil {
		fromStdin = opts.Extra["from-stdin"] == "true"
	}

	// If reading from stdin, parse the piped output
	if fromStdin {
		stdin := s.Stdin
		if stdin == nil {
			stdin = os.Stdin
		}
		return s.fetchFromReader(stdin, opts)
	}

	// Path is provided via Project field (matches CLI pattern of -p flag)
	path := opts.Project
	if path == "" {
		path = "."
	}

	// Create client
	client := NewClient(command)

	// Run scan
	report, err := client.Scan(ctx, path, config)
	if err != nil {
		return nil, fmt.Errorf("scanning with semgrep: %w", err)
	}

	return s.buildReport(report, opts, path)
}

// fetchFromReader parses semgrep output from a reader.
func (s *Source) fetchFromReader(r io.Reader, opts sources.Options) (*confidence.Report, error) {
	report, err := parseFromReader(r)
	if err != nil {
		return nil, err
	}

	path := opts.Project
	if path == "" {
		path = "."
	}

	return s.buildReport(report, opts, path)
}

// buildReport creates a confidence report from semgrep results.
func (s *Source) buildReport(report *Report, opts sources.Options, path string) (*confidence.Report, error) {
	// Aggregate counts
	counts := countFromResults(report.Results)

	// Determine title
	title := opts.Title
	if title == "" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}
		title = filepath.Base(absPath)
	}

	// Build factors with severity-based scoring
	factors := []confidence.Factor{
		{
			Name:        "Error Findings",
			Score:       scoring.SeverityScore(counts.Error, penaltyError),
			Weight:      weightError,
			Description: fmt.Sprintf("%d errors", counts.Error),
		},
		{
			Name:        "Warning Findings",
			Score:       scoring.SeverityScore(counts.Warning, penaltyWarning),
			Weight:      weightWarning,
			Description: fmt.Sprintf("%d warnings", counts.Warning),
		},
		{
			Name:        "Info Findings",
			Score:       scoring.SeverityScore(counts.Info, penaltyInfo),
			Weight:      weightInfo,
			Description: fmt.Sprintf("%d info", counts.Info),
		},
	}

	return scoring.BuildReport(title, sourceName, opts.Threshold, factors), nil
}
