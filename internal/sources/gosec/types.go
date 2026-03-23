// Package gosec provides a source for fetching security findings from Gosec.
package gosec

import (
	"fmt"
	"os"
	"strings"
)

// Report represents the JSON output from gosec -fmt=json.
type Report struct {
	Issues []Issue    `json:"Issues"`
	Stats  Stats      `json:"Stats"`
	Golang GolangInfo `json:"GolangInfo"`
}

// Issue represents a single Gosec finding.
type Issue struct {
	Severity   string `json:"severity"`   // HIGH, MEDIUM, LOW
	Confidence string `json:"confidence"` // HIGH, MEDIUM, LOW
	CWE        CWE    `json:"cwe"`
	RuleID     string `json:"rule_id"`
	Details    string `json:"details"`
	File       string `json:"file"`
	Code       string `json:"code"`
	Line       string `json:"line"`
	Column     string `json:"column"`
}

// CWE contains the Common Weakness Enumeration info.
type CWE struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// Stats contains scan statistics.
type Stats struct {
	Files int `json:"files"`
	Lines int `json:"lines"`
	NoSec int `json:"nosec"`
	Found int `json:"found"`
}

// GolangInfo contains Go version information.
type GolangInfo struct {
	Version string `json:"version"`
}

// FindingCounts aggregates findings by severity.
type FindingCounts struct {
	High   int
	Medium int
	Low    int
}

// countFromIssues aggregates finding counts from gosec issues.
func countFromIssues(issues []Issue) FindingCounts {
	var counts FindingCounts
	for _, issue := range issues {
		switch strings.ToUpper(issue.Severity) {
		case "HIGH":
			counts.High++
		case "MEDIUM":
			counts.Medium++
		case "LOW":
			counts.Low++
		default:
			if issue.Severity != "" {
				fmt.Fprintf(os.Stderr, "Warning: unknown gosec severity %q, finding not counted\n", issue.Severity)
			}
		}
	}
	return counts
}

// Severity penalties (points deducted per issue).
// Gosec reports only HIGH/MEDIUM/LOW (no CRITICAL tier). Penalties match the
// default HIGH/MEDIUM values but LOW is softer (3 vs 5) because gosec LOW
// findings are often informational style issues rather than exploitable flaws.
const (
	penaltyHigh   = 20
	penaltyMedium = 10
	penaltyLow    = 3
)

// Factor weights.
// Weights are shifted upward (50/35/15) compared to the default 4-tier split
// (40/30/20/10) because three tiers must sum to 100. HIGH gets the most weight
// since gosec HIGH findings (e.g. hardcoded credentials, SQL injection) are
// the closest equivalent to CRITICAL in other scanners.
const (
	weightHigh   = 50
	weightMedium = 35
	weightLow    = 15
)
