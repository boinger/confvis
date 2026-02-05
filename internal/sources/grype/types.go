// Package grype provides a source for fetching vulnerability metrics from Grype.
package grype

import "github.com/boinger/confvis/internal/sources/scoring"

// Report represents the JSON output from grype -o json.
type Report struct {
	Matches []Match `json:"matches"`
}

// Match represents a single vulnerability match.
type Match struct {
	Vulnerability Vulnerability `json:"vulnerability"`
	Artifact      Artifact      `json:"artifact"`
}

// Vulnerability contains details about the vulnerability.
type Vulnerability struct {
	ID          string   `json:"id"`
	Severity    string   `json:"severity"` // Critical, High, Medium, Low, Negligible, Unknown
	Description string   `json:"description"`
	URLs        []string `json:"urls,omitempty"`
}

// Artifact represents the vulnerable artifact (package).
type Artifact struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"`
}

// countFromMatches aggregates vulnerability counts from scan matches.
func countFromMatches(matches []Match) scoring.SeverityCounts {
	var counts scoring.SeverityCounts
	for _, match := range matches {
		switch match.Vulnerability.Severity {
		case "Critical":
			counts.Critical++
		case "High":
			counts.High++
		case "Medium":
			counts.Medium++
		case "Low", "Negligible":
			counts.Low++
		}
	}
	return counts
}

