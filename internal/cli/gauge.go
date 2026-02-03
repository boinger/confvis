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
	gaugeCmd.Flags().StringVarP(&gaugeFormat, "format", "f", "svg", "output format: svg, json, text, markdown, or github-comment")
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

// GaugeDeps contains dependencies for the gauge command.
type GaugeDeps struct {
	FS               FileSystem
	Stdin            io.Reader
	Stdout           io.Writer
	Stderr           io.Writer
	Verbose          bool
	Quiet            bool
	ExitFunc         func(int)
	HistoryReader    func(string) (*history.History, error)
	HistoryAppender  func(string, history.Entry) error
	Config           string
	Output           string
	Format           string
	Style            string
	BadgeType        string
	Label            string
	InputFormat      string
	Compare          string
	HistoryFile      string
	Width            int
	Height           int
	FailUnder        int
	GreenAbove       int
	YellowAbove      int
	HistoryCount     int
	Dark             bool
	FailOnRegression bool
}

func runGauge(_ *cobra.Command, _ []string) error {
	return gaugeImpl(&GaugeDeps{
		FS:               DefaultFileSystem,
		Stdin:            os.Stdin,
		Stdout:           os.Stdout,
		Stderr:           os.Stderr,
		Verbose:          verbose,
		Quiet:            quiet,
		ExitFunc:         os.Exit,
		HistoryReader:    history.ReadFile,
		HistoryAppender:  history.AppendToFile,
		Config:           gaugeConfig,
		Output:           gaugeOutput,
		Format:           gaugeFormat,
		Style:            gaugeStyle,
		BadgeType:        gaugeBadgeType,
		Label:            gaugeLabel,
		InputFormat:      gaugeInputFormat,
		Compare:          gaugeCompare,
		HistoryFile:      gaugeHistoryFile,
		Width:            gaugeWidth,
		Height:           gaugeHeight,
		FailUnder:        gaugeFailUnder,
		GreenAbove:       gaugeGreenAbove,
		YellowAbove:      gaugeYellowAbove,
		HistoryCount:     gaugeHistoryCount,
		Dark:             gaugeDark,
		FailOnRegression: gaugeFailOnRegression,
	})
}

func gaugeImpl(deps *GaugeDeps) error {
	// Validate format
	switch deps.Format {
	case "svg", "json", "text", "markdown", "github-comment":
		// valid
	default:
		return fmt.Errorf("invalid format %q: must be svg, json, text, markdown, or github-comment", deps.Format)
	}

	// Validate badge type
	switch deps.BadgeType {
	case "gauge", "flat", "sparkline":
		// valid
	default:
		return fmt.Errorf("invalid badge-type %q: must be gauge, flat, or sparkline", deps.BadgeType)
	}

	// Validate and convert input format
	var inputFormat confidence.Format
	switch deps.InputFormat {
	case "auto":
		inputFormat = confidence.FormatAuto
	case "json":
		inputFormat = confidence.FormatJSON
	case "yaml":
		inputFormat = confidence.FormatYAML
	default:
		return fmt.Errorf("invalid input-format %q: must be auto, json, or yaml", deps.InputFormat)
	}

	var report *confidence.Report
	var err error

	if deps.Config == "-" {
		// For stdin, use JSON by default unless explicitly specified
		if inputFormat == confidence.FormatAuto {
			inputFormat = confidence.FormatJSON
		}
		report, err = confidence.ParseWithFormat(deps.Stdin, inputFormat)
	} else {
		// Read file using injected filesystem
		var reader io.Reader
		reader, inputFormat, err = openConfigFile(deps.FS, deps.Config, inputFormat)
		if err != nil {
			return fmt.Errorf("parsing config: %w", err)
		}
		report, err = confidence.ParseWithFormat(reader, inputFormat)
	}
	if err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	// Parse baseline report if comparing
	var baseline *confidence.Report
	var delta int
	if deps.Compare != "" {
		reader, format, err := openConfigFile(deps.FS, deps.Compare, confidence.FormatAuto)
		if err != nil {
			return fmt.Errorf("parsing baseline: %w", err)
		}
		baseline, err = confidence.ParseWithFormat(reader, format)
		if err != nil {
			return fmt.Errorf("parsing baseline: %w", err)
		}
		delta = report.Score - baseline.Score
	}

	// Suppress verbose output when writing to stdout
	outputToStdout := deps.Output == "-"
	showVerbose := deps.Verbose && !deps.Quiet && !outputToStdout

	if showVerbose {
		fmt.Printf("Generating %s for %q (score: %d, threshold: %d)\n",
			deps.Format, report.Title, report.Score, report.Threshold)
	}

	var w io.Writer
	if outputToStdout {
		w = deps.Stdout
	} else {
		f, err := deps.FS.Create(deps.Output)
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

	if err := writeFormatOutput(w, deps.Format, report, baseline, delta, deps); err != nil {
		return err
	}

	if showVerbose {
		fmt.Printf("Wrote %s to %s\n", deps.Format, deps.Output)
	}

	if deps.FailUnder > 0 && report.Score < deps.FailUnder {
		if !deps.Quiet {
			_, _ = fmt.Fprintf(deps.Stderr, "Score %d is below threshold %d\n", report.Score, deps.FailUnder)
		}
		deps.ExitFunc(1)
	}

	if deps.FailOnRegression && baseline != nil && delta < 0 {
		if !deps.Quiet {
			_, _ = fmt.Fprintf(deps.Stderr, "Score regressed from %d to %d (%d)\n", baseline.Score, report.Score, delta)
		}
		deps.ExitFunc(1)
	}

	return nil
}

// writeFormatOutput dispatches to the appropriate format writer.
func writeFormatOutput(w io.Writer, format string, report *confidence.Report, baseline *confidence.Report, delta int, deps *GaugeDeps) error {
	switch format {
	case "svg":
		return generateSVGBadge(w, report, deps)
	case "json":
		return writeJSON(w, report, baseline, delta)
	case "text":
		return writeText(w, report.Score, baseline, delta)
	case "markdown":
		return writeMarkdown(w, report, baseline, delta)
	case "github-comment":
		return writeGitHubComment(w, report, baseline)
	}
	return nil
}

// writeJSON generates JSON output for the report.
func writeJSON(w io.Writer, report *confidence.Report, baseline *confidence.Report, delta int) error {
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
	return nil
}

// writeText generates plain text output for the report.
func writeText(w io.Writer, score int, baseline *confidence.Report, delta int) error {
	if baseline != nil {
		sign := "+"
		if delta < 0 {
			sign = ""
		}
		if _, err := fmt.Fprintf(w, "%d (%s%d)\n", score, sign, delta); err != nil {
			return fmt.Errorf("writing text output: %w", err)
		}
	} else {
		if _, err := fmt.Fprintf(w, "%d\n", score); err != nil {
			return fmt.Errorf("writing text output: %w", err)
		}
	}
	return nil
}

// generateSVGBadge generates the SVG badge based on badge type.
func generateSVGBadge(w io.Writer, report *confidence.Report, deps *GaugeDeps) error {
	switch deps.BadgeType {
	case "flat":
		flatOpts := gauge.FlatOptions{
			Label:       deps.Label,
			DarkMode:    deps.Dark,
			Style:       deps.Style,
			GreenAbove:  deps.GreenAbove,
			YellowAbove: deps.YellowAbove,
		}
		if err := gauge.GenerateFlat(w, report, flatOpts); err != nil {
			return fmt.Errorf("generating flat badge: %w", err)
		}

	case "sparkline":
		// Read history and generate sparkline
		var scores []int
		if deps.HistoryFile != "" {
			hist, err := deps.HistoryReader(deps.HistoryFile)
			if err != nil {
				return fmt.Errorf("reading history: %w", err)
			}
			// Get last N entries plus current score
			entries := hist.Last(deps.HistoryCount - 1)
			for _, e := range entries {
				scores = append(scores, e.Score)
			}
		}
		// Append current score
		scores = append(scores, report.Score)

		sparkOpts := gauge.SparklineOptions{
			Width:       deps.Width,
			Height:      deps.Height,
			Scores:      scores,
			DarkMode:    deps.Dark,
			Style:       deps.Style,
			GreenAbove:  deps.GreenAbove,
			YellowAbove: deps.YellowAbove,
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
		if deps.HistoryFile != "" {
			entry := history.NewEntry(report.Score)
			if err := deps.HistoryAppender(deps.HistoryFile, entry); err != nil {
				return fmt.Errorf("appending to history: %w", err)
			}
		}

	default: // "gauge"
		opts := gauge.Options{
			Width:       deps.Width,
			Height:      deps.Height,
			Style:       deps.Style,
			DarkMode:    deps.Dark,
			GreenAbove:  deps.GreenAbove,
			YellowAbove: deps.YellowAbove,
		}
		if err := gauge.Generate(w, report, opts); err != nil {
			return fmt.Errorf("generating gauge: %w", err)
		}
	}

	return nil
}

// writeGitHubComment generates GitHub-flavored markdown output for PR comments.
func writeGitHubComment(w io.Writer, report *confidence.Report, baseline *confidence.Report) error {
	if err := writeGitHubCommentHeader(w, report); err != nil {
		return err
	}

	if err := writeGitHubCommentBaseline(w, report, baseline); err != nil {
		return err
	}

	if err := writeGitHubCommentFactors(w, report.Factors); err != nil {
		return err
	}

	return writeGitHubCommentFooter(w, report.Version)
}

func writeGitHubCommentHeader(w io.Writer, report *confidence.Report) error {
	statusEmoji := ":white_check_mark:"
	statusText := "Passed"
	if !report.Passed() {
		statusEmoji = ":x:"
		statusText = "Failed"
	}

	if _, err := fmt.Fprintf(w, "## Confidence Report: %s\n\n", report.Title); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "| Metric | Value |"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "|--------|-------|"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "| Score | **%d%%** %s |\n", report.Score, statusEmoji); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "| Threshold | %d%% |\n", report.Threshold); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "| Status | %s |\n", statusText)
	return err
}

func writeGitHubCommentBaseline(w io.Writer, report *confidence.Report, baseline *confidence.Report) error {
	if baseline == nil {
		return nil
	}
	delta := report.Score - baseline.Score
	deltaEmoji := ":left_right_arrow:"
	if delta > 0 {
		deltaEmoji = ":arrow_up:"
	} else if delta < 0 {
		deltaEmoji = ":arrow_down:"
	}
	_, err := fmt.Fprintf(w, "| Change | %+d %s |\n", delta, deltaEmoji)
	return err
}

func writeGitHubCommentFactors(w io.Writer, factors []confidence.Factor) error {
	if len(factors) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, "\n<details>"); err != nil {
		return err
	}
	if _, err := fmt.Fprint(w, "<summary>Factor Breakdown</summary>\n\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "| Factor | Score | Weight | Description |"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "|--------|-------|--------|-------------|"); err != nil {
		return err
	}
	for _, f := range factors {
		desc := f.Description
		if desc == "" {
			desc = "-"
		}
		if _, err := fmt.Fprintf(w, "| %s | %d | %d | %s |\n", f.Name, f.Score, f.Weight, desc); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w, "\n</details>")
	return err
}

func writeGitHubCommentFooter(w io.Writer, version string) error {
	if version == "" {
		version = "unknown"
	}
	_, err := fmt.Fprintf(w, "\n---\n<sub>Generated by confvis v%s</sub>\n", version)
	return err
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
