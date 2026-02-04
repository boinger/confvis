package scoring

import (
	"fmt"

	"github.com/boinger/confvis/internal/confidence"
)

// SeverityCounts holds vulnerability counts by severity level.
type SeverityCounts struct {
	Critical int
	High     int
	Medium   int
	Low      int
}

// SeverityConfig defines penalties and weights for a severity level.
type SeverityConfig struct {
	Name    string
	Count   int
	Penalty int
	Weight  int
}

// VulnSeverityConfigs returns standard vulnerability severity configs.
// This is used by grype, trivy, snyk, and dependabot sources.
func VulnSeverityConfigs(counts SeverityCounts, penalties, weights [4]int) []SeverityConfig {
	return []SeverityConfig{
		{Name: "Critical Vulnerabilities", Count: counts.Critical, Penalty: penalties[0], Weight: weights[0]},
		{Name: "High Vulnerabilities", Count: counts.High, Penalty: penalties[1], Weight: weights[1]},
		{Name: "Medium Vulnerabilities", Count: counts.Medium, Penalty: penalties[2], Weight: weights[2]},
		{Name: "Low Vulnerabilities", Count: counts.Low, Penalty: penalties[3], Weight: weights[3]},
	}
}

// BuildSeverityFactors creates confidence factors from severity configs.
// The optional url parameter adds the same URL to all factors.
func BuildSeverityFactors(configs []SeverityConfig, url string) []confidence.Factor {
	factors := make([]confidence.Factor, len(configs))
	for i, cfg := range configs {
		factors[i] = confidence.Factor{
			Name:        cfg.Name,
			Score:       SeverityScore(cfg.Count, cfg.Penalty),
			Weight:      cfg.Weight,
			Description: formatDescription(cfg.Name, cfg.Count),
		}
		if url != "" {
			factors[i].URL = url
		}
	}
	return factors
}

// formatDescription creates a description like "3 critical" from factor name and count.
func formatDescription(name string, count int) string {
	// Extract severity level from name (e.g., "Critical Vulnerabilities" -> "critical")
	severity := extractSeverity(name)
	return fmt.Sprintf("%d %s", count, severity)
}

// extractSeverity extracts the severity level from a factor name.
func extractSeverity(name string) string {
	switch name {
	case "Critical Vulnerabilities":
		return "critical"
	case "High Vulnerabilities":
		return "high"
	case "Medium Vulnerabilities":
		return "medium"
	case "Low Vulnerabilities":
		return "low"
	case "Error Findings":
		return "errors"
	case "Warning Findings":
		return "warnings"
	case "Info Findings":
		return "info"
	default:
		return "issues"
	}
}
