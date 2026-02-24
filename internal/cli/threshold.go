package cli

import (
	"fmt"

	"github.com/boinger/confvis/internal/confidence"
)

// ThresholdConfig holds configuration for threshold checks.
type ThresholdConfig struct {
	FailUnder        int
	FailOnRegression bool
	FactorThresholds map[string]int
}

// ThresholdResult holds the result of threshold checks.
type ThresholdResult struct {
	ScorePassed    bool     // true if score >= FailUnder (or FailUnder not set)
	BaselinePassed bool     // true if not regressed (or regression check not enabled)
	FactorsPassed  bool     // true if all factor thresholds met
	FactorFailures []string // per-factor failure messages
}

// Passed returns true if all threshold checks passed.
func (r ThresholdResult) Passed() bool {
	return r.ScorePassed && r.BaselinePassed && r.FactorsPassed
}

// CheckThresholds evaluates threshold conditions against a report.
// baseline and delta are only used if cfg.FailOnRegression is true.
func CheckThresholds(report *confidence.Report, baselineReport *confidence.Report, delta int, cfg ThresholdConfig) ThresholdResult {
	result := ThresholdResult{
		ScorePassed:    true,
		BaselinePassed: true,
		FactorsPassed:  true,
	}

	if cfg.FailUnder > 0 && report.ScoreValue() < cfg.FailUnder {
		result.ScorePassed = false
	}

	if cfg.FailOnRegression && baselineReport != nil && delta < 0 {
		result.BaselinePassed = false
	}

	passed, failures := checkFactorThresholds(report, cfg.FactorThresholds)
	if !passed {
		result.FactorsPassed = false
		result.FactorFailures = failures
	}

	return result
}

// checkFactorThresholds validates that all factors meet their thresholds.
// Thresholds are checked in this precedence order:
// 1. CLI/config override (overrides map)
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
