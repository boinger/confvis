package ghactions

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/scoring"
)

const sourceName = "github-actions"

// Fetcher defines the interface for fetching GitHub Actions data.
type Fetcher interface {
	FetchRuns(ctx context.Context, ownerRepo string, opts FetchRunsOptions) (*WorkflowRunsResponse, error)
	ActionsURL(ownerRepo string) string
}

// Environment variable names for configuration.
const (
	EnvToken  = "GITHUB_TOKEN"
	EnvAPIURL = "GITHUB_API_URL"
)

// Default values.
const (
	DefaultRunCount = 20
)

var configResolver = &sources.ConfigResolver{
	SourceName:     "GitHub",
	TokenEnvVar:    EnvToken,
	URLEnvVar:      EnvAPIURL,
	TokenRequired:  true,
	URLRequired:    false, // Has default
	DefaultTimeout: 30 * time.Second,
}

// Source implements the sources.Source interface for GitHub Actions.
type Source struct{}

func init() {
	sources.Register(&Source{})
}

// Name returns the source identifier.
func (s *Source) Name() string {
	return sourceName
}

// Fetch retrieves workflow metrics from GitHub Actions and converts them to a confidence report.
func (s *Source) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
	cfg, err := configResolver.Resolve(opts)
	if err != nil {
		return nil, err
	}

	// Parse extra options
	var workflow, event string
	runCount := DefaultRunCount
	if opts.Extra != nil {
		workflow = opts.Extra["workflow"]
		event = opts.Extra["event"]
		if countStr, ok := opts.Extra["count"]; ok && countStr != "" {
			if n, err := strconv.Atoi(countStr); err == nil && n > 0 {
				runCount = n
			}
		}
	}

	client := NewClient(cfg.URL, cfg.Token, cfg.Timeout)

	return s.FetchWithClient(ctx, client, opts, FetchRunsOptions{
		Workflow: workflow,
		Event:    event,
		Count:    runCount,
	})
}

// FetchWithClient retrieves workflow metrics using the provided Fetcher.
// This allows injecting mock clients for testing.
func (s *Source) FetchWithClient(ctx context.Context, fetcher Fetcher, opts sources.Options, runOpts FetchRunsOptions) (*confidence.Report, error) {
	// Fetch workflow runs
	runsResp, err := fetcher.FetchRuns(ctx, opts.Project, runOpts)
	if err != nil {
		return nil, err
	}

	// Calculate success rate
	successCount := 0
	totalCount := len(runsResp.WorkflowRuns)
	for _, run := range runsResp.WorkflowRuns {
		if run.Conclusion == "success" {
			successCount++
		}
	}

	var successRate int
	if totalCount > 0 {
		successRate = (successCount * 100) / totalCount
	}

	// Determine title
	title := opts.Title
	if title == "" {
		title = opts.Project
	}

	// Build description
	description := fmt.Sprintf("%d/%d successful runs", successCount, totalCount)

	// Build factors
	factors := []confidence.Factor{
		{
			Name:        "Workflow Success Rate",
			Score:       successRate,
			Weight:      100,
			Description: description,
			URL:         fetcher.ActionsURL(opts.Project),
		},
	}

	return scoring.BuildReport(title, sourceName, opts.Threshold, factors), nil
}
