// Package trufflehog provides a source for detecting secrets using TruffleHog.
package trufflehog

// Finding represents a single secret detected by TruffleHog.
// TruffleHog outputs JSON lines (one JSON object per line).
type Finding struct {
	SourceMetadata SourceMetadata `json:"SourceMetadata"`
	SourceID       int            `json:"SourceID"`
	SourceType     int            `json:"SourceType"`
	SourceName     string         `json:"SourceName"`
	DetectorType   int            `json:"DetectorType"`
	DetectorName   string         `json:"DetectorName"`
	DecoderName    string         `json:"DecoderName"`
	Verified       bool           `json:"Verified"` // True if TruffleHog verified the secret is valid
	Raw            string         `json:"Raw"`      // The raw secret value (may be redacted)
	RawV2          string         `json:"RawV2"`
	Redacted       string         `json:"Redacted"` // Redacted version of the secret
	ExtraData      map[string]string `json:"ExtraData,omitempty"`
	StructuredData *StructuredData `json:"StructuredData,omitempty"`
}

// SourceMetadata contains information about where the secret was found.
type SourceMetadata struct {
	Data Data `json:"Data"`
}

// Data contains the source-specific metadata.
type Data struct {
	Filesystem *FilesystemData `json:"Filesystem,omitempty"`
	Git        *GitData        `json:"Git,omitempty"`
}

// FilesystemData contains metadata for filesystem scans.
type FilesystemData struct {
	File string `json:"file"`
	Line int64  `json:"line"`
}

// GitData contains metadata for git repository scans.
type GitData struct {
	Commit     string `json:"commit"`
	File       string `json:"file"`
	Email      string `json:"email"`
	Repository string `json:"repository"`
	Timestamp  string `json:"timestamp"`
	Line       int64  `json:"line"`
}

// StructuredData contains structured information about the secret.
type StructuredData struct {
	Github   *GithubData   `json:"Github,omitempty"`
	TLSCert  *TLSCertData  `json:"TlsCertificate,omitempty"`
	Postgres *PostgresData `json:"Postgres,omitempty"`
}

// GithubData contains GitHub-specific data.
type GithubData struct {
	// Token-specific fields
}

// TLSCertData contains TLS certificate data.
type TLSCertData struct {
	// Certificate-specific fields
}

// PostgresData contains PostgreSQL connection data.
type PostgresData struct {
	// Connection-specific fields
}

// FindingCounts aggregates findings by verification status.
type FindingCounts struct {
	Verified   int // Verified secrets (confirmed valid)
	Unverified int // Unverified secrets (potentially valid)
}

// Severity penalties for secret detection.
// Verified secrets are more severe than unverified ones.
const (
	penaltyVerified   = 30 // Verified secrets are critical
	penaltyUnverified = 10 // Unverified secrets are warnings
)

// Factor weights.
const (
	weightVerified   = 60 // Verified secrets are weighted more heavily
	weightUnverified = 40
)
