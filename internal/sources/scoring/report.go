package scoring

import (
	"fmt"
	"time"

	"github.com/boinger/confvis/internal/confidence"
)

// BuildReport creates a confidence.Report with standard fields.
// This eliminates duplicated report building code across sources.
// Returns an error if the constructed report fails validation.
func BuildReport(title, sourceName string, threshold int, factors []confidence.Factor) (*confidence.Report, error) {
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
		return nil, fmt.Errorf("BuildReport produced invalid report: %w", err)
	}

	return report, nil
}
