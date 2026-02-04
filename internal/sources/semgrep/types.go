// Package semgrep provides a source for fetching security findings from Semgrep.
package semgrep

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

// CountFromResults aggregates finding counts from scan results.
func CountFromResults(results []Result) FindingCounts {
	var counts FindingCounts
	for _, result := range results {
		switch result.Extra.Severity {
		case "ERROR":
			counts.Error++
		case "WARNING":
			counts.Warning++
		case "INFO":
			counts.Info++
		}
	}
	return counts
}

// Severity penalties (points deducted per issue).
const (
	PenaltyError   = 20
	PenaltyWarning = 10
	PenaltyInfo    = 2
)

// Factor weights.
const (
	WeightError   = 40
	WeightWarning = 35
	WeightInfo    = 25
)

// SeverityScore calculates a score based on finding count and penalty.
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
