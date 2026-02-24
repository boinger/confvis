package cli

import (
	"fmt"

	"github.com/boinger/confvis/internal/baseline"
	"github.com/boinger/confvis/internal/confidence"
)

// BaselineConfig holds configuration for loading baselines.
type BaselineConfig struct {
	CompareBaseline      bool
	Compare              string // explicit baseline file path
	BaselineRef          string
	BaselineFile         string
	FS                   FileSystem
	IsGitRepo            func() bool
	BaselineGitRefReader func(string) (*baseline.Baseline, error)
	BaselineFileReader   func(string) (*baseline.Baseline, error)
}

// LoadBaseline loads a baseline report based on configuration.
// Returns the baseline report and score delta, or nil if no baseline comparison is requested.
func LoadBaseline(cfg BaselineConfig, currentScore int) (*confidence.Report, int, error) {
	if cfg.CompareBaseline {
		b, err := resolveBaselineFromConfig(cfg)
		if err != nil {
			return nil, 0, fmt.Errorf("loading baseline: %w", err)
		}
		if b == nil {
			return nil, 0, nil
		}
		return &b.Report, currentScore - b.ScoreValue(), nil
	}

	if cfg.Compare != "" {
		loader := &ReportLoader{FS: cfg.FS, Config: cfg.Compare}
		baselineReport, err := loader.LoadReport()
		if err != nil {
			return nil, 0, fmt.Errorf("loading baseline: %w", err)
		}
		return baselineReport, currentScore - baselineReport.ScoreValue(), nil
	}

	return nil, 0, nil
}

// resolveBaselineFromConfig fetches the baseline from git ref or file.
// Returns nil if no baseline exists (not an error).
func resolveBaselineFromConfig(cfg BaselineConfig) (*baseline.Baseline, error) {
	// Try file first if specified
	if cfg.BaselineFile != "" {
		b, err := cfg.BaselineFileReader(cfg.BaselineFile)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	// Try git ref if in a repo
	if cfg.IsGitRepo != nil && cfg.IsGitRepo() {
		ref := cfg.BaselineRef
		if ref == "" {
			ref = baseline.DefaultBaselineRef
		}
		b, err := cfg.BaselineGitRefReader(ref)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	// Not in a git repo and no file specified
	return nil, nil //nolint:nilnil // nil baseline with no error means "not found"
}
