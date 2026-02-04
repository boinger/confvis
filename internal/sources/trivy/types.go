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

// Penalty and weight constants are defined in the scoring package.
// trivy uses the default strict penalties shared with grype and snyk.

