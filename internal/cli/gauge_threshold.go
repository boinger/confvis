package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/boinger/confvis/internal/confidence"
)

// parseFactorThresholds parses factor thresholds from CLI flags and config.
// CLI flags take precedence over config values.
// Format: "Factor Name:threshold"
func parseFactorThresholds(cliThresholds []string, configThresholds map[string]int) (map[string]int, error) {
	result := make(map[string]int)

	// Start with config values (lower precedence)
	for name, threshold := range configThresholds {
		result[name] = threshold
	}

	// Override with CLI values (higher precedence)
	for _, spec := range cliThresholds {
		name, threshold, err := parseFactorThresholdSpec(spec)
		if err != nil {
			return nil, err
		}
		result[name] = threshold
	}

	return result, nil
}

// parseFactorThresholdSpec parses a single factor threshold specification.
// Format: "Factor Name:threshold"
func parseFactorThresholdSpec(spec string) (string, int, error) {
	lastColon := strings.LastIndex(spec, ":")
	if lastColon == -1 {
		return "", 0, fmt.Errorf("invalid factor-threshold format %q: expected 'Name:threshold'", spec)
	}

	name := spec[:lastColon]
	thresholdStr := spec[lastColon+1:]

	if name == "" {
		return "", 0, fmt.Errorf("invalid factor-threshold format %q: factor name cannot be empty", spec)
	}

	threshold, err := strconv.Atoi(thresholdStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid factor-threshold format %q: threshold must be an integer", spec)
	}

	if threshold < 0 || threshold > 100 {
		return "", 0, fmt.Errorf("invalid factor-threshold format %q: threshold must be 0-100", spec)
	}

	return name, threshold, nil
}

// checkFactorThresholds validates that all factors meet their thresholds.
// Thresholds are checked in this precedence order:
// 1. CLI/config override (deps.FactorThresholds)
// 2. Factor's own Threshold field
// Returns (passed, failures) where failures lists factors that failed.
func checkFactorThresholds(report *confidence.Report, overrides map[string]int) (bool, []string) {
	var failures []string

	for _, factor := range report.Factors {
		// Determine effective threshold: override > factor.Threshold
		threshold := factor.Threshold
		if override, ok := overrides[factor.Name]; ok {
			threshold = override
		}

		// Skip if no threshold is set
		if threshold <= 0 {
			continue
		}

		if factor.Score < threshold {
			failures = append(failures, fmt.Sprintf("%s: %d < %d", factor.Name, factor.Score, threshold))
		}
	}

	return len(failures) == 0, failures
}
