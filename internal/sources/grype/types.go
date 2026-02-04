// Package grype provides a source for fetching vulnerability metrics from Grype.
package grype

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

// IssueCounts aggregates vulnerabilities by severity.
type IssueCounts struct {
	Critical int
	High     int
	Medium   int
	Low      int
	Unknown  int
}

// CountFromMatches aggregates vulnerability counts from scan matches.
func CountFromMatches(matches []Match) IssueCounts {
	var counts IssueCounts
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
		default:
			counts.Unknown++
		}
	}
	return counts
}

// Severity penalties (points deducted per issue).
// Same as Trivy/Snyk for consistency.
const (
	PenaltyCritical = 33
	PenaltyHigh     = 20
	PenaltyMedium   = 10
	PenaltyLow      = 5
)

// Factor weights.
const (
	WeightCritical = 40
	WeightHigh     = 30
	WeightMedium   = 20
	WeightLow      = 10
)

