package scoring

import (
	"time"

	"github.com/boinger/confvis/internal/confidence"
)

// BuildReport creates a confidence.Report with standard fields.
// This eliminates duplicated report building code across sources.
func BuildReport(title, sourceName string, threshold int, factors []confidence.Factor) *confidence.Report {
	report := &confidence.Report{
		Title:       title,
		Threshold:   threshold,
		Source:      sourceName,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Factors:     factors,
	}
	report.Score = report.CalculateScore()
	return report
}
