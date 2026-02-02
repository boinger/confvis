// Package codecov provides a source for fetching coverage metrics from Codecov.
package codecov

// ReportResponse represents the response from /api/v2/{service}/{owner}/repos/{repo}/report/.
type ReportResponse struct {
	Totals Totals `json:"totals"`
}

// Totals contains the coverage summary.
type Totals struct {
	Coverage float64 `json:"coverage"`
}
