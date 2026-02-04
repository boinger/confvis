// Package scoring provides shared scoring utilities for vulnerability sources.
package scoring

// SeverityScore calculates a score based on issue count and penalty.
// Returns 100 if count is 0, otherwise decreases by penalty per issue (min 0).
//
// This is a shared implementation used by vulnerability scanners (trivy, grype, snyk)
// and static analysis tools (semgrep) to convert issue counts into confidence scores.
func SeverityScore(count, penalty int) int {
	if count == 0 {
		return 100
	}
	score := 100 - (count * penalty)
	if score < 0 {
		return 0
	}
	return score
}
