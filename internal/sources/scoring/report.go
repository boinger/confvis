package scoring

import (
	"fmt"
	"time"

	"github.com/boinger/confvis/internal/confidence"
)

// BuildReport creates a confidence.Report with standard fields.
// This eliminates duplicated report building code across sources.
// Panics if the constructed report fails validation, which indicates a programming error.
func BuildReport(title, sourceName string, threshold int, factors []confidence.Factor) *confidence.Report {
	report := &confidence.Report{
		Title:       title,
		Threshold:   threshold,
		Source:      sourceName,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Factors:     factors,
	}
	score := report.CalculateScore()
	report.Score = &score

	if err := confidence.Validate(report); err != nil {
		panic(fmt.Sprintf("BuildReport produced invalid report: %v", err))
	}

	return report
}
