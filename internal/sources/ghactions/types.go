// Package ghactions provides a source for fetching CI/CD metrics from GitHub Actions.
package ghactions

import "time"

// WorkflowRunsResponse represents the response from /repos/{owner}/{repo}/actions/runs.
type WorkflowRunsResponse struct {
	TotalCount   int           `json:"total_count"`
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

// WorkflowRun represents a single workflow run.
type WorkflowRun struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Status       string     `json:"status"`       // queued, in_progress, completed
	Conclusion   string     `json:"conclusion"`   // success, failure, neutral, cancelled, skipped, timed_out
	Event        string     `json:"event"`        // push, pull_request, etc.
	WorkflowID   int64      `json:"workflow_id"`
	RunNumber    int        `json:"run_number"`
	RunAttempt   int        `json:"run_attempt"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	HTMLURL      string     `json:"html_url"`
	WorkflowPath string     `json:"path"` // e.g., .github/workflows/ci.yml
}

