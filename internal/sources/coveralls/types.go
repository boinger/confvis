// Package coveralls provides a source for fetching coverage metrics from Coveralls.
package coveralls

// ReportResponse represents the response from /github/{owner}/{repo}.json.
type ReportResponse struct {
	RepoName       string  `json:"repo_name"`        // Full repo name (e.g., "owner/repo")
	BadgeURL       string  `json:"badge_url"`        // URL to the coverage badge
	CoveredPercent float64 `json:"covered_percent"`  // Coverage percentage (0-100)
	CoverageChange float64 `json:"coverage_change"`  // Change from previous build
	RepoTokens     int     `json:"repo_tokens"`      // Number of repo tokens
	CreatedAt      string  `json:"created_at"`       // When the repo was added
	UpdatedAt      string  `json:"updated_at"`       // Last update time
}
