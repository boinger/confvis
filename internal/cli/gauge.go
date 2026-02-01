package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/gauge"
	"github.com/boinger/confvis/internal/history"
)

var (
	gaugeConfig           string
	gaugeOutput           string
	gaugeWidth            int
	gaugeHeight           int
	gaugeDark             bool
	gaugeFailUnder        int
	gaugeFormat           string
	gaugeStyle            string
	gaugeGreenAbove       int
	gaugeYellowAbove      int
	gaugeCompare          string
	gaugeFailOnRegression bool
	gaugeBadgeType        string
	gaugeLabel            string
	gaugeInputFormat      string
	gaugeHistoryFile      string
	gaugeHistoryCount     int
)

var gaugeCmd = &cobra.Command{
	Use:   "gauge",
	Short: "Generate an SVG gauge badge",
	Long:  `Generate an SVG gauge badge from a confidence report (JSON or YAML).`,
	RunE:  runGauge,
}

func init() {
	gaugeCmd.Flags().StringVarP(&gaugeConfig, "config", "c", "", "path to confidence report (JSON/YAML), or - for stdin (required)")
	gaugeCmd.Flags().StringVar(&gaugeInputFormat, "input-format", "auto", "input format: auto, json, or yaml (auto-detects from extension)")
	gaugeCmd.Flags().StringVarP(&gaugeOutput, "output", "o", "", "output file path, or - for stdout (required)")
	gaugeCmd.Flags().StringVarP(&gaugeFormat, "format", "f", "svg", "output format: svg, json, text, or markdown")
	gaugeCmd.Flags().IntVar(&gaugeWidth, "width", 200, "gauge width in pixels (svg only)")
	gaugeCmd.Flags().IntVar(&gaugeHeight, "height", 120, "gauge height in pixels (svg only)")
	gaugeCmd.Flags().StringVar(&gaugeStyle, "style", "github", "color scheme: github, minimal, corporate, high-contrast (svg only)")
	gaugeCmd.Flags().BoolVar(&gaugeDark, "dark", false, "use dark mode colors (svg only)")
	gaugeCmd.Flags().IntVar(&gaugeFailUnder, "fail-under", 0, "exit non-zero if score is below this value")
	gaugeCmd.Flags().IntVar(&gaugeGreenAbove, "green-above", 0, "score threshold for green color (overrides JSON config)")
	gaugeCmd.Flags().IntVar(&gaugeYellowAbove, "yellow-above", 0, "score threshold for yellow color (overrides JSON config)")
	gaugeCmd.Flags().StringVar(&gaugeCompare, "compare", "", "path to baseline report JSON for comparison")
	gaugeCmd.Flags().BoolVar(&gaugeFailOnRegression, "fail-on-regression", false, "exit non-zero if score decreased from baseline (requires --compare)")
	gaugeCmd.Flags().StringVar(&gaugeBadgeType, "badge-type", "gauge", "badge type: gauge, flat, or sparkline")
	gaugeCmd.Flags().StringVar(&gaugeLabel, "label", "", "custom label for flat badge (defaults to report title)")
	gaugeCmd.Flags().StringVar(&gaugeHistoryFile, "history-file", "", "path to history file for sparkline (JSON lines format)")
	gaugeCmd.Flags().IntVar(&gaugeHistoryCount, "history-count", 10, "number of historical points to show in sparkline")

	if err := gaugeCmd.MarkFlagRequired("config"); err != nil {
		panic(err)
	}
	if err := gaugeCmd.MarkFlagRequired("output"); err != nil {
		panic(err)
	}

	rootCmd.AddCommand(gaugeCmd)
}

func runGauge(_ *cobra.Command, _ []string) error {
	// Validate format
	switch gaugeFormat {
	case "svg", "json", "text", "markdown":
		// valid
	default:
		return fmt.Errorf("invalid format %q: must be svg, json, text, or markdown", gaugeFormat)
	}

	// Validate badge type
	switch gaugeBadgeType {
	case "gauge", "flat", "sparkline":
		// valid
	default:
		return fmt.Errorf("invalid badge-type %q: must be gauge, flat, or sparkline", gaugeBadgeType)
	}

	// Validate and convert input format
	var inputFormat confidence.Format
	switch gaugeInputFormat {
	case "auto":
		inputFormat = confidence.FormatAuto
	case "json":
		inputFormat = confidence.FormatJSON
	case "yaml":
		inputFormat = confidence.FormatYAML
	default:
		return fmt.Errorf("invalid input-format %q: must be auto, json, or yaml", gaugeInputFormat)
	}

	var report *confidence.Report
	var err error

	if gaugeConfig == "-" {
		// For stdin, use JSON by default unless explicitly specified
		if inputFormat == confidence.FormatAuto {
			inputFormat = confidence.FormatJSON
		}
		report, err = confidence.ParseWithFormat(os.Stdin, inputFormat)
	} else {
		report, err = confidence.ParseFileWithFormat(gaugeConfig, inputFormat)
	}
	if err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	// Parse baseline report if comparing
	var baseline *confidence.Report
	var delta int
	if gaugeCompare != "" {
		baseline, err = confidence.ParseFile(gaugeCompare)
		if err != nil {
			return fmt.Errorf("parsing baseline: %w", err)
		}
		delta = report.Score - baseline.Score
	}

	// Suppress verbose output when writing to stdout
	outputToStdout := gaugeOutput == "-"
	showVerbose := verbose && !quiet && !outputToStdout

	if showVerbose {
		fmt.Printf("Generating %s for %q (score: %d, threshold: %d)\n",
			gaugeFormat, report.Title, report.Score, report.Threshold)
	}

	var w io.Writer
	if outputToStdout {
		w = os.Stdout
	} else {
		f, err := os.Create(gaugeOutput)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer func() {
			if cerr := f.Close(); cerr != nil && err == nil {
				err = fmt.Errorf("closing output file: %w", cerr)
			}
		}()
		w = f
	}

	switch gaugeFormat {
	case "svg":
		switch gaugeBadgeType {
		case "flat":
			flatOpts := gauge.FlatOptions{
				Label:       gaugeLabel,
				DarkMode:    gaugeDark,
				Style:       gaugeStyle,
				GreenAbove:  gaugeGreenAbove,
				YellowAbove: gaugeYellowAbove,
			}
			if err := gauge.GenerateFlat(w, report, flatOpts); err != nil {
				return fmt.Errorf("generating flat badge: %w", err)
			}

		case "sparkline":
			// Read history and generate sparkline
			var scores []int
			if gaugeHistoryFile != "" {
				hist, err := history.ReadFile(gaugeHistoryFile)
				if err != nil {
					return fmt.Errorf("reading history: %w", err)
				}
				// Get last N entries plus current score
				entries := hist.Last(gaugeHistoryCount - 1)
				for _, e := range entries {
					scores = append(scores, e.Score)
				}
			}
			// Append current score
			scores = append(scores, report.Score)

			sparkOpts := gauge.SparklineOptions{
				Width:       gaugeWidth,
				Height:      gaugeHeight,
				Scores:      scores,
				DarkMode:    gaugeDark,
				Style:       gaugeStyle,
				GreenAbove:  gaugeGreenAbove,
				YellowAbove: gaugeYellowAbove,
			}
			// Use smaller default size for sparkline
			if sparkOpts.Width == 200 {
				sparkOpts.Width = 120
			}
			if sparkOpts.Height == 120 {
				sparkOpts.Height = 28
			}
			if err := gauge.GenerateSparkline(w, report, sparkOpts); err != nil {
				return fmt.Errorf("generating sparkline: %w", err)
			}

			// Append to history file if specified
			if gaugeHistoryFile != "" {
				entry := history.NewEntry(report.Score)
				if err := history.AppendToFile(gaugeHistoryFile, entry); err != nil {
					return fmt.Errorf("appending to history: %w", err)
				}
			}

		default: // "gauge"
			opts := gauge.Options{
				Width:       gaugeWidth,
				Height:      gaugeHeight,
				Style:       gaugeStyle,
				DarkMode:    gaugeDark,
				GreenAbove:  gaugeGreenAbove,
				YellowAbove: gaugeYellowAbove,
			}
			if err := gauge.Generate(w, report, opts); err != nil {
				return fmt.Errorf("generating gauge: %w", err)
			}
		}

	case "json":
		output := struct {
			Title       string `json:"title"`
			Score       int    `json:"score"`
			Threshold   int    `json:"threshold"`
			Passed      bool   `json:"passed"`
			Version     string `json:"version,omitempty"`
			GeneratedAt string `json:"generatedAt,omitempty"`
			Source      string `json:"source,omitempty"`
			Baseline    *int   `json:"baseline,omitempty"`
			Delta       *int   `json:"delta,omitempty"`
		}{
			Title:       report.Title,
			Score:       report.Score,
			Threshold:   report.Threshold,
			Passed:      report.Passed(),
			Version:     report.Version,
			GeneratedAt: report.GeneratedAt,
			Source:      report.Source,
		}
		if baseline != nil {
			output.Baseline = &baseline.Score
			output.Delta = &delta
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(output); err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}

	case "text":
		if baseline != nil {
			sign := "+"
			if delta < 0 {
				sign = ""
			}
			if _, err := fmt.Fprintf(w, "%d (%s%d)\n", report.Score, sign, delta); err != nil {
				return fmt.Errorf("writing text output: %w", err)
			}
		} else {
			if _, err := fmt.Fprintf(w, "%d\n", report.Score); err != nil {
				return fmt.Errorf("writing text output: %w", err)
			}
		}

	case "markdown":
		if err := writeMarkdown(w, report, baseline, delta); err != nil {
			return fmt.Errorf("writing markdown output: %w", err)
		}
	}

	if showVerbose {
		fmt.Printf("Wrote %s to %s\n", gaugeFormat, gaugeOutput)
	}

	if gaugeFailUnder > 0 && report.Score < gaugeFailUnder {
		if !quiet {
			fmt.Fprintf(os.Stderr, "Score %d is below threshold %d\n", report.Score, gaugeFailUnder)
		}
		os.Exit(1)
	}

	if gaugeFailOnRegression && baseline != nil && delta < 0 {
		if !quiet {
			fmt.Fprintf(os.Stderr, "Score regressed from %d to %d (%d)\n", baseline.Score, report.Score, delta)
		}
		os.Exit(1)
	}

	return nil
}

// writeMarkdown generates markdown output for the report.
func writeMarkdown(w io.Writer, report *confidence.Report, baseline *confidence.Report, delta int) error {
	status := report.EffectivePassLabel()
	if !report.Passed() {
		status = report.EffectiveFailLabel()
	}

	// Header with title, score, and status
	if baseline != nil {
		sign := "+"
		if delta < 0 {
			sign = ""
		}
		if _, err := fmt.Fprintf(w, "## %s: %d%% (%s) [%s%d from %d%%]\n\n", report.Title, report.Score, status, sign, delta, baseline.Score); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(w, "## %s: %d%% (%s)\n\n", report.Title, report.Score, status); err != nil {
			return err
		}
	}

	// Description if present
	if report.Description != "" {
		if _, err := fmt.Fprintf(w, "%s\n\n", report.Description); err != nil {
			return err
		}
	}

	// Factors table if present
	if len(report.Factors) > 0 {
		if _, err := fmt.Fprintln(w, "| Factor | Score | Weight |"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "|--------|------:|-------:|"); err != nil {
			return err
		}
		for _, f := range report.Factors {
			name := f.Name
			if f.URL != "" {
				name = fmt.Sprintf("[%s](%s)", f.Name, f.URL)
			}
			if _, err := fmt.Fprintf(w, "| %s | %d%% | %d%% |\n", name, f.Score, f.Weight); err != nil {
				return err
			}
		}
	}

	return nil
}
