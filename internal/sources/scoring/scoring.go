// Package scoring provides shared scoring utilities for vulnerability sources.
package scoring

import (
	"fmt"
	"os"
	"strings"
)

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

// CountSeverity increments the appropriate counter based on severity string.
// If warnOnUnknown is true, prints a warning for unknown severity strings.
func CountSeverity(counts *SeverityCounts, severity, toolName string, warnOnUnknown bool) {
	switch strings.ToLower(severity) {
	case "critical":
		counts.Critical++
	case "high":
		counts.High++
	case "medium":
		counts.Medium++
	case "low":
		counts.Low++
	default:
		sev := strings.ToLower(severity)
		if warnOnUnknown && sev != "" {
			fmt.Fprintf(os.Stderr, "Warning: unknown %s severity %q, alert not counted\n", toolName, sev)
		}
	}
}
