package confidence

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Format represents the input file format.
type Format string

const (
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
	FormatAuto Format = "auto"
)

// ParseFileWithFormat reads and parses a confidence report from a file path
// using the specified format. If format is FormatAuto, it's detected from the extension.
func ParseFileWithFormat(path string, format Format) (*Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}

	// Auto-detect format from extension
	autoDetected := false
	if format == FormatAuto {
		format = DetectFormat(path)
		autoDetected = true
	}

	report, err := ParseWithFormat(f, format)
	if closeErr := f.Close(); closeErr != nil && err == nil {
		return nil, fmt.Errorf("closing file: %w", closeErr)
	}
	if err != nil && autoDetected && !IsKnownFormatExtension(path) {
		return nil, fmt.Errorf("parsing %q as %s (unrecognized extension; expected .json, .yaml, or .yml): %w",
			path, format, err)
	}
	return report, err
}

// DetectFormat returns the format based on file extension.
// Returns FormatJSON for .json files and unknown extensions.
// Callers can use IsKnownFormatExtension to check if the extension was recognized.
func DetectFormat(path string) Format {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return FormatYAML
	default:
		return FormatJSON
	}
}

// IsKnownFormatExtension returns true if the file extension maps to a recognized format.
func IsKnownFormatExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

// ParseWithFormat reads and parses a confidence report from an io.Reader
// using the specified format.
func ParseWithFormat(r io.Reader, format Format) (*Report, error) {
	var report Report
	var err error

	switch format {
	case FormatYAML:
		err = yaml.NewDecoder(r).Decode(&report)
		if err != nil {
			return nil, fmt.Errorf("decoding YAML: %w", err)
		}
	default: // JSON
		err = json.NewDecoder(r).Decode(&report)
		if err != nil {
			return nil, fmt.Errorf("decoding JSON: %w", err)
		}
	}

	// Auto-calculate score from factors if score is omitted (nil)
	if report.Score == nil && len(report.Factors) > 0 {
		score := report.CalculateScore()
		report.Score = &score
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

	if r.Score != nil && (*r.Score < 0 || *r.Score > 100) {
		return fmt.Errorf("validation: score must be between 0 and 100, got %d", *r.Score)
	}

	if r.Threshold < 0 || r.Threshold > 100 {
		return fmt.Errorf("validation: threshold must be between 0 and 100, got %d", r.Threshold)
	}

	for i, f := range r.Factors {
		if err := validateFactor(i, f); err != nil {
			return err
		}
	}

	if r.Thresholds != nil {
		if err := validateColorThresholds(r.Thresholds); err != nil {
			return err
		}
	}

	return nil
}

func validateFactor(i int, f Factor) error {
	if f.Name == "" {
		return fmt.Errorf("validation: factor[%d] name is required", i)
	}
	if f.Score < 0 || f.Score > 100 {
		return fmt.Errorf("validation: factor[%d] score must be between 0 and 100, got %d", i, f.Score)
	}
	if f.Weight < 0 || f.Weight > 100 {
		return fmt.Errorf("validation: factor[%d] weight must be between 0 and 100, got %d", i, f.Weight)
	}
	if f.Threshold < 0 || f.Threshold > 100 {
		return fmt.Errorf("validation: factor %q threshold must be between 0 and 100, got %d", f.Name, f.Threshold)
	}
	return nil
}

func validateColorThresholds(t *ColorThresholds) error {
	if t.GreenAbove < 0 || t.GreenAbove > 100 {
		return fmt.Errorf("validation: thresholds.greenAbove must be between 0 and 100, got %d", t.GreenAbove)
	}
	if t.YellowAbove < 0 || t.YellowAbove > 100 {
		return fmt.Errorf("validation: thresholds.yellowAbove must be between 0 and 100, got %d", t.YellowAbove)
	}
	if t.GreenAbove < t.YellowAbove {
		return fmt.Errorf("validation: thresholds.greenAbove (%d) must be >= thresholds.yellowAbove (%d)",
			t.GreenAbove, t.YellowAbove)
	}
	return nil
}
