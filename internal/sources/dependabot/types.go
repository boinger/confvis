// Package dependabot provides a source for fetching vulnerability alerts from GitHub Dependabot.
package dependabot

// AlertsResponse represents the response from /repos/{owner}/{repo}/dependabot/alerts.
// Note: GitHub returns an array of alerts, not a wrapper object.
type AlertsResponse []Alert

// Alert represents a single Dependabot alert.
type Alert struct {
	Number            int              `json:"number"`
	State             string           `json:"state"` // "open", "dismissed", "fixed"
	SecurityAdvisory  SecurityAdvisory `json:"security_advisory"`
	SecurityVulnerability *SecurityVulnerability `json:"security_vulnerability,omitempty"`
	HTMLURL           string           `json:"html_url"`
}

// SecurityAdvisory contains information about the security advisory.
type SecurityAdvisory struct {
	GHSAID      string   `json:"ghsa_id"`
	CVE         *string  `json:"cve_id"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	Severity    string   `json:"severity"` // "critical", "high", "medium", "low"
}

// SecurityVulnerability contains vulnerability-specific information.
type SecurityVulnerability struct {
	Package          Package `json:"package"`
	Severity         string  `json:"severity"`
	VulnerableVersionRange string `json:"vulnerable_version_range"`
}

// Package represents the vulnerable package.
type Package struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
}

// AlertCounts holds vulnerability counts by severity.
type AlertCounts struct {
	Critical int
	High     int
	Medium   int
	Low      int
}

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
