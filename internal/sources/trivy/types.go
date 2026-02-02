// Package trivy provides a source for fetching vulnerability metrics from Trivy.
package trivy

// Report represents the JSON output from trivy fs --format json.
type Report struct {
	Results []Result `json:"Results"`
}

// Result represents a single scan result (e.g., for a specific file or package).
type Result struct {
	Target          string          `json:"Target"`
	Class           string          `json:"Class"`
	Type            string          `json:"Type"`
	Vulnerabilities []Vulnerability `json:"Vulnerabilities"`
}

// Vulnerability represents a single vulnerability found.
type Vulnerability struct {
	VulnerabilityID  string   `json:"VulnerabilityID"`
	PkgName          string   `json:"PkgName"`
	InstalledVersion string   `json:"InstalledVersion"`
	FixedVersion     string   `json:"FixedVersion"`
	Severity         string   `json:"Severity"`
	Title            string   `json:"Title"`
	Description      string   `json:"Description"`
	References       []string `json:"References"`
}

// IssueCounts aggregates vulnerabilities by severity.
type IssueCounts struct {
	Critical int
	High     int
	Medium   int
	Low      int
	Unknown  int
}

// CountFromResults aggregates vulnerability counts from scan results.
func CountFromResults(results []Result) IssueCounts {
	var counts IssueCounts
	for _, result := range results {
		for _, vuln := range result.Vulnerabilities {
			switch vuln.Severity {
			case "CRITICAL":
				counts.Critical++
			case "HIGH":
				counts.High++
			case "MEDIUM":
				counts.Medium++
			case "LOW":
				counts.Low++
			default:
				counts.Unknown++
			}
		}
	}
	return counts
}

// Severity penalties (points deducted per issue).
// Same as Snyk for consistency.
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

// SeverityScore calculates a score based on vulnerability count and penalty.
// Returns 100 if count is 0, otherwise decreases by penalty per issue (min 0).
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
