package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/boinger/confvis/internal/baseline"
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
	gaugeIcon             string
	gaugeInputFormat      string
	gaugeHistoryFile      string
	gaugeHistoryCount     int
	gaugeHistoryRef       string
	gaugeHistoryAuto      bool
	gaugeCompareBaseline  bool
	gaugeBaselineRef      string
	gaugeBaselineFile     string
	gaugeFactorThresholds []string
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
	// Note: defaults are set to 0/"" here; actual defaults come from config.go getters
	gaugeCmd.Flags().IntVar(&gaugeWidth, "width", 0, "gauge width in pixels (svg only)")
	gaugeCmd.Flags().IntVar(&gaugeHeight, "height", 0, "gauge height in pixels (svg only)")
	gaugeCmd.Flags().StringVar(&gaugeStyle, "style", "", "color scheme: github, minimal, corporate, high-contrast (svg only)")
	gaugeCmd.Flags().BoolVar(&gaugeDark, "dark", false, "use dark mode colors (svg only)")
	gaugeCmd.Flags().IntVar(&gaugeFailUnder, "fail-under", 0, "exit non-zero if score is below this value")
	gaugeCmd.Flags().IntVar(&gaugeGreenAbove, "green-above", 0, "score threshold for green color (overrides JSON config)")
	gaugeCmd.Flags().IntVar(&gaugeYellowAbove, "yellow-above", 0, "score threshold for yellow color (overrides JSON config)")
	gaugeCmd.Flags().StringVar(&gaugeCompare, "compare", "", "path to baseline report JSON for comparison")
	gaugeCmd.Flags().BoolVar(&gaugeCompareBaseline, "compare-baseline", false, "auto-fetch baseline from ref/file and compare")
	gaugeCmd.Flags().StringVar(&gaugeBaselineRef, "baseline-ref", "", "git ref for baseline storage (default: refs/confvis/baseline)")
	gaugeCmd.Flags().StringVar(&gaugeBaselineFile, "baseline-file", "", "file path fallback for non-git repos")
	gaugeCmd.Flags().BoolVar(&gaugeFailOnRegression, "fail-on-regression", false, "exit non-zero if score decreased from baseline")
	gaugeCmd.Flags().StringVar(&gaugeBadgeType, "badge-type", "", "badge type: gauge, flat, or sparkline")
	gaugeCmd.Flags().StringVar(&gaugeLabel, "label", "", "custom label for flat badge (defaults to report title)")
	gaugeCmd.Flags().StringVar(&gaugeIcon, "icon", "", "SVG path data for flat badge icon")
	gaugeCmd.Flags().StringVar(&gaugeHistoryFile, "history-file", "", "path to history file for sparkline (JSON lines format)")
	gaugeCmd.Flags().IntVar(&gaugeHistoryCount, "history-count", 0, "number of historical points to show in sparkline")
	gaugeCmd.Flags().StringVar(&gaugeHistoryRef, "history-ref", "", "git ref for storing history (default: refs/confvis/history)")
	gaugeCmd.Flags().BoolVar(&gaugeHistoryAuto, "history-auto", false, "auto-detect history storage: use git ref if in repo, else file")
	gaugeCmd.Flags().StringSliceVar(&gaugeFactorThresholds, "factor-threshold", nil, "per-factor threshold in 'Name:threshold' format (can be repeated)")

	// Bind flags to viper for config file support
	bindGaugeFlags(gaugeCmd)

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
	FS                    FileSystem
	Stdin                 io.Reader
	Stdout                io.Writer
	Stderr                io.Writer
	Verbose               bool
	Quiet                 bool
	ExitFunc              func(int)
	HistoryReader         func(string) (*history.History, error)
	HistoryAppender       func(string, history.Entry) error
	GitRefReader          func(string) (*history.History, error)
	GitRefAppender        func(string, history.Entry) error
	BaselineGitRefReader  func(string) (*baseline.Baseline, error)
	BaselineFileReader    func(string) (*baseline.Baseline, error)
	IsGitRepo             func() bool
	Config                string
	Output                string
	Format                string
	Style                 string
	BadgeType             string
	Label                 string
	Icon                  string
	InputFormat           string
	Compare               string
	HistoryFile           string
	HistoryRef            string
	BaselineRef           string
	BaselineFile          string
	FactorThresholds      map[string]int
	Width                 int
	Height                int
	FailUnder             int
	GreenAbove            int
	YellowAbove           int
	HistoryCount          int
	Dark                  bool
	FailOnRegression      bool
	HistoryAuto           bool
	CompareBaseline       bool
}

func runGauge(_ *cobra.Command, _ []string) error {
	// Parse factor thresholds from CLI and config
	factorThresholds, err := parseFactorThresholds(gaugeFactorThresholds, getGaugeFactorThresholds())
	if err != nil {
		return err
	}

	// Use config getters which handle config < env < flag precedence
	return gaugeImpl(&GaugeDeps{
		FS:                   DefaultFileSystem,
		Stdin:                os.Stdin,
		Stdout:               os.Stdout,
		Stderr:               os.Stderr,
		Verbose:              verbose,
		Quiet:                quiet,
		ExitFunc:             os.Exit,
		HistoryReader:        history.ReadFile,
		HistoryAppender:      history.AppendToFile,
		GitRefReader:         history.ReadFromGitRef,
		GitRefAppender:       history.AppendToGitRef,
		BaselineGitRefReader: baseline.ReadFromGitRef,
		BaselineFileReader:   baseline.ReadFromFile,
		IsGitRepo:            baseline.IsGitRepo,
		Config:               gaugeConfig,
		Output:               gaugeOutput,
		Format:               gaugeFormat,
		Style:                getGaugeStyle(),
		BadgeType:            getGaugeBadgeType(),
		Label:                gaugeLabel,
		Icon:                 gaugeIcon,
		InputFormat:          gaugeInputFormat,
		Compare:              gaugeCompare,
		HistoryFile:          getGaugeHistoryFile(),
		HistoryRef:           getGaugeHistoryRef(),
		BaselineRef:          getGaugeBaselineRef(),
		BaselineFile:         getGaugeBaselineFile(),
		FactorThresholds:     factorThresholds,
		Width:                getGaugeWidth(),
		Height:               getGaugeHeight(),
		FailUnder:            getGaugeFailUnder(),
		GreenAbove:           getGaugeGreenAbove(),
		YellowAbove:          getGaugeYellowAbove(),
		HistoryCount:         getGaugeHistoryCount(),
		Dark:                 getGaugeDark(),
		FailOnRegression:     gaugeFailOnRegression,
		HistoryAuto:          getGaugeHistoryAuto(),
		CompareBaseline:      getGaugeCompareBaseline(),
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
	baselineReport, delta, err := loadBaselineForComparison(deps, report.Score)
	if err != nil {
		return err
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

	if err := writeFormatOutput(w, deps.Format, report, baselineReport, delta, deps); err != nil {
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

	if deps.FailOnRegression && baselineReport != nil && delta < 0 {
		if !deps.Quiet {
			_, _ = fmt.Fprintf(deps.Stderr, "Score regressed from %d to %d (%d)\n", baselineReport.Score, report.Score, delta)
		}
		deps.ExitFunc(1)
	}

	// Check per-factor thresholds
	if passed, failures := checkFactorThresholds(report, deps.FactorThresholds); !passed {
		if !deps.Quiet {
			for _, failure := range failures {
				_, _ = fmt.Fprintf(deps.Stderr, "Factor threshold failed: %s\n", failure)
			}
		}
		deps.ExitFunc(1)
	}

	return nil
}

// writeFormatOutput dispatches to the appropriate format writer.
func writeFormatOutput(w io.Writer, format string, report *confidence.Report, baselineReport *confidence.Report, delta int, deps *GaugeDeps) error {
	switch format {
	case "svg":
		return generateSVGBadge(w, report, deps)
	case "json":
		return writeJSON(w, report, baselineReport, delta)
	case "text":
		return writeText(w, report.Score, baselineReport, delta)
	case "markdown":
		return writeMarkdown(w, report, baselineReport, delta)
	case "github-comment":
		return writeGitHubComment(w, report, baselineReport)
	}
	return nil
}

// writeJSON generates JSON output for the report.
func writeJSON(w io.Writer, report *confidence.Report, baselineReport *confidence.Report, delta int) error {
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
	if baselineReport != nil {
		output.Baseline = &baselineReport.Score
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
func writeText(w io.Writer, score int, baselineReport *confidence.Report, delta int) error {
	if baselineReport != nil {
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
			Icon:        deps.Icon,
			DarkMode:    deps.Dark,
			Style:       deps.Style,
			GreenAbove:  deps.GreenAbove,
			YellowAbove: deps.YellowAbove,
		}
		if err := gauge.GenerateFlat(w, report, flatOpts); err != nil {
			return fmt.Errorf("generating flat badge: %w", err)
		}

	case "sparkline":
		// Determine history storage mode
		useGitRef, historyRef, historyFile := resolveHistoryStorage(deps)

		// Read history and generate sparkline
		var scores []int
		if useGitRef && historyRef != "" {
			hist, err := deps.GitRefReader(historyRef)
			if err != nil {
				return fmt.Errorf("reading history from git ref: %w", err)
			}
			entries := hist.Last(deps.HistoryCount - 1)
			for _, e := range entries {
				scores = append(scores, e.Score)
			}
		} else if historyFile != "" {
			hist, err := deps.HistoryReader(historyFile)
			if err != nil {
				return fmt.Errorf("reading history: %w", err)
			}
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

		// Append to history storage
		entry := history.NewEntry(report.Score)
		if useGitRef && historyRef != "" {
			if err := deps.GitRefAppender(historyRef, entry); err != nil {
				return fmt.Errorf("appending to history git ref: %w", err)
			}
		} else if historyFile != "" {
			if err := deps.HistoryAppender(historyFile, entry); err != nil {
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

// resolveHistoryStorage determines which history storage mode to use.
// Returns: useGitRef (bool), historyRef (string), historyFile (string).
// Logic:
//   - If --history-ref is explicitly set, use git ref storage
//   - If --history-auto is set, auto-detect: git ref if in repo, else file
//   - If --history-file is set, use file storage
//   - Otherwise, no history storage
func resolveHistoryStorage(deps *GaugeDeps) (useGitRef bool, historyRef string, historyFile string) {
	// Explicit --history-ref takes precedence
	if deps.HistoryRef != "" {
		return true, deps.HistoryRef, ""
	}

	// --history-auto: detect storage mode
	if deps.HistoryAuto {
		if deps.IsGitRepo != nil && deps.IsGitRepo() {
			// In a git repo, use git ref storage with default ref
			return true, history.DefaultHistoryRef, ""
		}
		// Not in a git repo, fall back to default file
		defaultFile := ".confvis-history.jsonl"
		if deps.HistoryFile != "" {
			defaultFile = deps.HistoryFile
		}
		return false, "", defaultFile
	}

	// Explicit --history-file
	if deps.HistoryFile != "" {
		return false, "", deps.HistoryFile
	}

	// No history storage configured
	return false, "", ""
}

// loadBaselineForComparison loads a baseline report based on deps configuration.
// Returns the baseline report and score delta, or nil if no baseline comparison is requested.
func loadBaselineForComparison(deps *GaugeDeps, currentScore int) (*confidence.Report, int, error) {
	if deps.CompareBaseline {
		b, err := resolveBaseline(deps)
		if err != nil {
			return nil, 0, fmt.Errorf("loading baseline: %w", err)
		}
		if b == nil {
			return nil, 0, nil
		}
		return &b.Report, currentScore - b.Score, nil
	}

	if deps.Compare != "" {
		reader, format, err := openConfigFile(deps.FS, deps.Compare, confidence.FormatAuto)
		if err != nil {
			return nil, 0, fmt.Errorf("parsing baseline: %w", err)
		}
		baselineReport, err := confidence.ParseWithFormat(reader, format)
		if err != nil {
			return nil, 0, fmt.Errorf("parsing baseline: %w", err)
		}
		return baselineReport, currentScore - baselineReport.Score, nil
	}

	return nil, 0, nil
}

// resolveBaseline fetches the baseline from git ref or file.
// Returns nil if no baseline exists (not an error).
func resolveBaseline(deps *GaugeDeps) (*baseline.Baseline, error) {
	// Try file first if specified
	if deps.BaselineFile != "" {
		b, err := deps.BaselineFileReader(deps.BaselineFile)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	// Try git ref if in a repo
	if deps.IsGitRepo != nil && deps.IsGitRepo() {
		ref := deps.BaselineRef
		if ref == "" {
			ref = baseline.DefaultBaselineRef
		}
		b, err := deps.BaselineGitRefReader(ref)
		if err != nil {
			return nil, err
		}
		return b, nil
	}

	// Not in a git repo and no file specified
	return nil, nil //nolint:nilnil // nil baseline with no error means "not found"
}

// writeGitHubComment generates GitHub-flavored markdown output for PR comments.
func writeGitHubComment(w io.Writer, report *confidence.Report, baselineReport *confidence.Report) error {
	if err := writeGitHubCommentHeader(w, report); err != nil {
		return err
	}

	if err := writeGitHubCommentBaseline(w, report, baselineReport); err != nil {
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

func writeGitHubCommentBaseline(w io.Writer, report *confidence.Report, baselineReport *confidence.Report) error {
	if baselineReport == nil {
		return nil
	}
	delta := report.Score - baselineReport.Score
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
func writeMarkdown(w io.Writer, report *confidence.Report, baselineReport *confidence.Report, delta int) error {
	status := report.EffectivePassLabel()
	if !report.Passed() {
		status = report.EffectiveFailLabel()
	}

	// Header with title, score, and status
	if baselineReport != nil {
		sign := "+"
		if delta < 0 {
			sign = ""
		}
		if _, err := fmt.Fprintf(w, "## %s: %d%% (%s) [%s%d from %d%%]\n\n", report.Title, report.Score, status, sign, delta, baselineReport.Score); err != nil {
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
