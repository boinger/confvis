package cli

import (
	"fmt"
	"strconv"
	"strings"
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
