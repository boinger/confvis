// Package trivy provides a source for fetching vulnerability metrics from Trivy.
package trivy

import (
	"fmt"
	"os"

	"github.com/boinger/confvis/internal/sources/scoring"
)

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

// countFromResults aggregates vulnerability counts from scan results.
func countFromResults(results []Result) scoring.SeverityCounts {
	var counts scoring.SeverityCounts
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
				// "UNKNOWN" is expected for unrated vulnerabilities, don't warn
				if vuln.Severity != "" && vuln.Severity != "UNKNOWN" {
					fmt.Fprintf(os.Stderr, "Warning: unknown trivy severity %q, finding not counted\n", vuln.Severity)
				}
			}
		}
	}
	return counts
}

