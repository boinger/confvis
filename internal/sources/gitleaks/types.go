// Package gitleaks provides a source for detecting secrets using GitLeaks.
package gitleaks

// Report represents the JSON output from gitleaks detect --report-format json.
// Note: gitleaks outputs an array of findings directly.
type Report []Finding

// Finding represents a single secret detected by GitLeaks.
type Finding struct {
	Description string `json:"Description"`
	StartLine   int    `json:"StartLine"`
	EndLine     int    `json:"EndLine"`
	StartColumn int    `json:"StartColumn"`
	EndColumn   int    `json:"EndColumn"`
	Match       string `json:"Match"`    // The matched text (may be redacted)
	Secret      string `json:"Secret"`   // The actual secret (may be redacted)
	File        string `json:"File"`     // Path to the file containing the secret
	SymlinkFile string `json:"Symlink"`  // Symlink path if applicable
	Commit      string `json:"Commit"`   // Git commit SHA
	Entropy     float64 `json:"Entropy"` // Shannon entropy of the secret
	Author      string `json:"Author"`
	Email       string `json:"Email"`
	Date        string `json:"Date"`
	Message     string `json:"Message"` // Commit message
	Tags        []string `json:"Tags"`
	RuleID      string `json:"RuleID"` // Rule that triggered the finding
	Fingerprint string `json:"Fingerprint"` // Unique identifier for this finding
}

// FindingCounts aggregates findings by category.
// GitLeaks doesn't have severity levels, so we categorize by type:
// - secrets: actual credentials (API keys, passwords, etc.)
// - all: total count for reference
type FindingCounts struct {
	Secrets int // Count of detected secrets
}

// Severity penalties for secret detection.
// Leaked secrets are critical security issues.
const (
	penaltyPerSecret = 25 // Each secret is a critical issue
)

// Factor weights.
const (
	weightSecrets = 100 // Secrets are the only factor and get full weight
)
