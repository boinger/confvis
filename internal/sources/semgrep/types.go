// Package semgrep provides a source for fetching security findings from Semgrep.
package semgrep

import (
	"fmt"
	"os"
)

// Report represents the JSON output from semgrep --json.
type Report struct {
	Results []Result `json:"results"`
	Errors  []Error  `json:"errors,omitempty"`
}

// Result represents a single Semgrep finding.
type Result struct {
	CheckID string `json:"check_id"`
	Path    string `json:"path"`
	Start   struct {
		Line   int `json:"line"`
		Col    int `json:"col"`
		Offset int `json:"offset"`
	} `json:"start"`
	End struct {
		Line   int `json:"line"`
		Col    int `json:"col"`
		Offset int `json:"offset"`
	} `json:"end"`
	Extra Extra `json:"extra"`
}

// Extra contains additional metadata about the finding.
type Extra struct {
	Message  string            `json:"message"`
	Metadata Metadata          `json:"metadata,omitempty"`
	Severity string            `json:"severity"` // ERROR, WARNING, INFO
	Lines    string            `json:"lines,omitempty"`
	Fix      string            `json:"fix,omitempty"`
	Metavars map[string]string `json:"metavars,omitempty"`
}

// Metadata contains rule metadata.
type Metadata struct {
	Category        string   `json:"category,omitempty"`
	Confidence      string   `json:"confidence,omitempty"`
	CWE             []string `json:"cwe,omitempty"`
	Impact          string   `json:"impact,omitempty"`
	Likelihood      string   `json:"likelihood,omitempty"`
	OWASP           []string `json:"owasp,omitempty"`
	References      []string `json:"references,omitempty"`
	Subcategory     []string `json:"subcategory,omitempty"`
	Technology      []string `json:"technology,omitempty"`
	VulnerabilityClass []string `json:"vulnerability_class,omitempty"`
}

// Error represents a Semgrep error during scanning.
type Error struct {
	Code    int    `json:"code"`
	Level   string `json:"level"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
	Type    string `json:"type"`
}

// FindingCounts aggregates findings by severity.
type FindingCounts struct {
	Error   int
	Warning int
	Info    int
}

// countFromResults aggregates finding counts from scan results.
func countFromResults(results []Result) FindingCounts {
	var counts FindingCounts
	for _, result := range results {
		switch result.Extra.Severity {
		case "ERROR":
			counts.Error++
		case "WARNING":
			counts.Warning++
		case "INFO":
			counts.Info++
		default:
			if result.Extra.Severity != "" {
				fmt.Fprintf(os.Stderr, "Warning: unknown semgrep severity %q, finding not counted\n", result.Extra.Severity)
			}
		}
	}
	return counts
}

// Severity penalties (points deducted per issue).
// Semgrep uses ERROR/WARNING/INFO rather than the standard 4-tier severity.
// ERROR ≈ HIGH (same penalty 20), WARNING ≈ MEDIUM (same penalty 10),
// INFO is softer than LOW (2 vs 5) because semgrep INFO findings are
// typically style suggestions, not security-relevant.
const (
	penaltyError   = 20
	penaltyWarning = 10
	penaltyInfo    = 2
)

// Factor weights.
// Three tiers must sum to 100. INFO gets more relative weight (25) than the
// default LOW (10) because semgrep rules are curated — even INFO-level rules
// are intentionally authored and worth tracking, unlike scanner noise.
const (
	weightError   = 40
	weightWarning = 35
	weightInfo    = 25
)

