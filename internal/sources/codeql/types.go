// Package codeql provides a source for fetching code scanning alerts from GitHub CodeQL.
package codeql

// AlertsResponse represents the response from /repos/{owner}/{repo}/code-scanning/alerts.
// Note: GitHub returns an array of alerts, not a wrapper object.
type AlertsResponse []Alert

// Alert represents a single code scanning alert.
type Alert struct {
	Number             int    `json:"number"`
	State              string `json:"state"` // "open", "dismissed", "fixed"
	Rule               Rule   `json:"rule"`
	Tool               Tool   `json:"tool"`
	MostRecentInstance Instance `json:"most_recent_instance"`
	HTMLURL            string `json:"html_url"`
}

// Rule contains information about the rule that triggered the alert.
type Rule struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Severity              string   `json:"severity"`                // "none", "note", "warning", "error"
	SecuritySeverityLevel string   `json:"security_severity_level"` // "low", "medium", "high", "critical"
	Description           string   `json:"description"`
	Tags                  []string `json:"tags"`
}

// Tool identifies the code scanning tool that produced the alert.
type Tool struct {
	Name    string  `json:"name"`
	Version *string `json:"version"`
}

// Instance represents a specific occurrence of the alert.
type Instance struct {
	Ref      string   `json:"ref"`
	State    string   `json:"state"`
	Location Location `json:"location"`
}

// Location identifies where the alert was found.
type Location struct {
	Path        string `json:"path"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	StartColumn int    `json:"start_column"`
	EndColumn   int    `json:"end_column"`
}

// AlertCounts holds alert counts by severity.
type AlertCounts struct {
	Critical int
	High     int
	Medium   int
	Low      int
}
