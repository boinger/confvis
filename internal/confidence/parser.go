package confidence

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ParseFile reads and parses a confidence report from a file path.
func ParseFile(path string) (*Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}

	report, err := Parse(f)
	if closeErr := f.Close(); closeErr != nil && err == nil {
		return nil, fmt.Errorf("closing file: %w", closeErr)
	}
	return report, err
}

// Parse reads and parses a confidence report from an io.Reader.
// If score is omitted (0) but factors exist, the score is automatically
// calculated as a weighted average of factor scores.
func Parse(r io.Reader) (*Report, error) {
	var report Report
	if err := json.NewDecoder(r).Decode(&report); err != nil {
		return nil, fmt.Errorf("decoding JSON: %w", err)
	}

	// Auto-calculate score from factors if score is omitted
	if report.Score == 0 && len(report.Factors) > 0 {
		report.Score = report.CalculateScore()
	}

	if err := Validate(&report); err != nil {
		return nil, err
	}

	return &report, nil
}

// Validate checks that a report has valid data.
func Validate(r *Report) error {
	if r.Title == "" {
		return fmt.Errorf("validation: title is required")
	}

	if r.Score < 0 || r.Score > 100 {
		return fmt.Errorf("validation: score must be between 0 and 100, got %d", r.Score)
	}

	if r.Threshold < 0 || r.Threshold > 100 {
		return fmt.Errorf("validation: threshold must be between 0 and 100, got %d", r.Threshold)
	}

	for i, f := range r.Factors {
		if f.Name == "" {
			return fmt.Errorf("validation: factor[%d] name is required", i)
		}
		if f.Score < 0 || f.Score > 100 {
			return fmt.Errorf("validation: factor[%d] score must be between 0 and 100, got %d", i, f.Score)
		}
		if f.Weight < 0 || f.Weight > 100 {
			return fmt.Errorf("validation: factor[%d] weight must be between 0 and 100, got %d", i, f.Weight)
		}
	}

	if r.Thresholds != nil {
		if r.Thresholds.GreenAbove < 0 || r.Thresholds.GreenAbove > 100 {
			return fmt.Errorf("validation: thresholds.greenAbove must be between 0 and 100, got %d", r.Thresholds.GreenAbove)
		}
		if r.Thresholds.YellowAbove < 0 || r.Thresholds.YellowAbove > 100 {
			return fmt.Errorf("validation: thresholds.yellowAbove must be between 0 and 100, got %d", r.Thresholds.YellowAbove)
		}
		if r.Thresholds.GreenAbove < r.Thresholds.YellowAbove {
			return fmt.Errorf("validation: thresholds.greenAbove (%d) must be >= thresholds.yellowAbove (%d)",
				r.Thresholds.GreenAbove, r.Thresholds.YellowAbove)
		}
	}

	return nil
}
