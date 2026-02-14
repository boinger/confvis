package cli

import (
	"fmt"
	"io"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/gauge"
)

// generateSVGBadge generates the SVG badge based on badge type.
func generateSVGBadge(w io.Writer, report *confidence.Report, deps *GaugeDeps) error {
	switch deps.BadgeType {
	case "flat":
		return generateFlatBadge(w, report, deps)
	case "sparkline":
		return generateSparklineBadge(w, report, deps)
	default: // "gauge"
		return generateGaugeBadge(w, report, deps)
	}
}

func generateFlatBadge(w io.Writer, report *confidence.Report, deps *GaugeDeps) error {
	flatOpts := gauge.FlatOptions{
		ColorOptions: gauge.ColorOptions{
			DarkMode:    deps.Dark,
			Style:       deps.Style,
			GreenAbove:  deps.GreenAbove,
			YellowAbove: deps.YellowAbove,
		},
		Label: deps.Label,
		Icon:  deps.Icon,
	}
	if err := gauge.GenerateFlat(w, report, flatOpts); err != nil {
		return fmt.Errorf("generating flat badge: %w", err)
	}
	return nil
}

func generateSparklineBadge(w io.Writer, report *confidence.Report, deps *GaugeDeps) error {
	useGitRef, historyRef, historyFile := resolveHistoryStorage(deps)

	scores, err := loadHistoryScores(deps, useGitRef, historyRef, historyFile)
	if err != nil {
		return err
	}
	scores = append(scores, report.ScoreValue())

	sparkOpts := gauge.SparklineOptions{
		ColorOptions: gauge.ColorOptions{
			DarkMode:    deps.Dark,
			Style:       deps.Style,
			GreenAbove:  deps.GreenAbove,
			YellowAbove: deps.YellowAbove,
		},
		Width:  deps.Width,
		Height: deps.Height,
		Scores: scores,
	}
	if sparkOpts.Width == 200 {
		sparkOpts.Width = 120
	}
	if sparkOpts.Height == 120 {
		sparkOpts.Height = 28
	}
	if err := gauge.GenerateSparkline(w, report, sparkOpts); err != nil {
		return fmt.Errorf("generating sparkline: %w", err)
	}

	return appendToHistory(deps, useGitRef, historyRef, historyFile, report.ScoreValue())
}

func generateGaugeBadge(w io.Writer, report *confidence.Report, deps *GaugeDeps) error {
	opts := gauge.Options{
		ColorOptions: gauge.ColorOptions{
			DarkMode:    deps.Dark,
			Style:       deps.Style,
			GreenAbove:  deps.GreenAbove,
			YellowAbove: deps.YellowAbove,
		},
		Width:  deps.Width,
		Height: deps.Height,
	}
	if err := gauge.Generate(w, report, opts); err != nil {
		return fmt.Errorf("generating gauge: %w", err)
	}
	return nil
}
