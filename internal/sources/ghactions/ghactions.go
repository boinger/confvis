package ghactions

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
)

const sourceName = "github-actions"

// Environment variable names for configuration.
const (
	EnvToken  = "GITHUB_TOKEN"
	EnvAPIURL = "GITHUB_API_URL"
)

// Default values.
const (
	DefaultRunCount = 20
)

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
	// Resolve token from option or environment
	token := opts.Token
	if token == "" {
		token = os.Getenv(EnvToken)
	}
	if token == "" {
		return nil, fmt.Errorf("GitHub token required: use --token flag or set %s", EnvToken)
	}

	// Resolve API URL from option or environment
	apiURL := opts.URL
	if apiURL == "" {
		apiURL = os.Getenv(EnvAPIURL)
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

	// Create client with timeout
	timeout := time.Duration(opts.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := NewClient(apiURL, token, timeout)

	// Fetch workflow runs
	runsResp, err := client.FetchRuns(ctx, opts.Project, FetchRunsOptions{
		Workflow: workflow,
		Event:    event,
		Count:    runCount,
	})
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
			URL:         client.ActionsURL(opts.Project),
		},
	}

	// Build report
	result := &confidence.Report{
		Title:       title,
		Threshold:   opts.Threshold,
		Source:      sourceName,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Factors:     factors,
	}

	// Calculate weighted score
	result.Score = result.CalculateScore()

	return result, nil
}
