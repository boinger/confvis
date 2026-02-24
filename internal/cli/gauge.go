package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/boinger/confvis/internal/baseline"
	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/gitutil"
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
	gaugeEmitJSON         string
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
	gaugeCmd.Flags().StringVarP(&gaugeOutput, "output", "o", "-", "output file path, or - for stdout (default: stdout)")
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
	gaugeCmd.Flags().StringVar(&gaugeEmitJSON, "emit-json", "", "also write JSON metadata to this path")

	// Bind flags to viper for config file support
	bindGaugeFlags(gaugeCmd)

	if err := gaugeCmd.MarkFlagRequired("config"); err != nil {
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
	EmitJSON              string
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
		IsGitRepo:            gitutil.IsGitRepo,
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
		EmitJSON:             gaugeEmitJSON,
	})
}

func gaugeImpl(deps *GaugeDeps) error {
	if err := validateGaugeInputs(deps); err != nil {
		return err
	}

	inputFormat, err := ParseInputFormat(deps.InputFormat)
	if err != nil {
		return err
	}

	loader := &ReportLoader{FS: deps.FS, Stdin: deps.Stdin, Config: deps.Config, Format: inputFormat}
	report, err := loader.LoadReport()
	if err != nil {
		return err
	}

	baselineReport, delta, err := LoadBaseline(baselineConfigFromGaugeDeps(deps), report.ScoreValue())
	if err != nil {
		return err
	}

	outputToStdout := deps.Output == "-"
	showVerbose := deps.Verbose && !deps.Quiet && !outputToStdout

	if showVerbose {
		fmt.Printf("Generating %s for %q (score: %d, threshold: %d)\n",
			deps.Format, report.Title, report.ScoreValue(), report.Threshold)
	}

	if err := writeGaugeOutput(deps, report, baselineReport, delta, outputToStdout); err != nil {
		return err
	}

	if deps.EmitJSON != "" {
		if err := emitJSONMetadata(deps, report, baselineReport, delta); err != nil {
			return err
		}
	}

	if showVerbose {
		fmt.Printf("Wrote %s to %s\n", deps.Format, deps.Output)
	}

	checkGaugeThresholds(deps, report, baselineReport, delta)

	return nil
}

// validateGaugeInputs checks that format and badge type are valid.
func validateGaugeInputs(deps *GaugeDeps) error {
	switch deps.Format {
	case "svg", "json", "text", "markdown", "github-comment":
		// valid
	default:
		return fmt.Errorf("invalid format %q: must be svg, json, text, markdown, or github-comment", deps.Format)
	}

	switch deps.BadgeType {
	case "gauge", "flat", "sparkline":
		// valid
	default:
		return fmt.Errorf("invalid badge-type %q: must be gauge, flat, or sparkline", deps.BadgeType)
	}

	return nil
}

// writeGaugeOutput writes the formatted output to the appropriate destination.
func writeGaugeOutput(deps *GaugeDeps, report, baselineReport *confidence.Report, delta int, outputToStdout bool) (err error) {
	var w io.Writer
	if outputToStdout {
		w = deps.Stdout
	} else {
		f, createErr := deps.FS.Create(deps.Output)
		if createErr != nil {
			return fmt.Errorf("creating output file: %w", createErr)
		}
		defer func() {
			if cerr := f.Close(); cerr != nil && err == nil {
				err = fmt.Errorf("closing output file: %w", cerr)
			}
		}()
		w = f
	}

	return writeFormatOutput(w, deps.Format, report, baselineReport, delta, deps)
}

// checkGaugeThresholds checks fail-under, regression, and per-factor thresholds.
func checkGaugeThresholds(deps *GaugeDeps, report *confidence.Report, baselineReport *confidence.Report, delta int) {
	result := CheckThresholds(report, baselineReport, delta, ThresholdConfig{
		FailUnder:        deps.FailUnder,
		FailOnRegression: deps.FailOnRegression,
		FactorThresholds: deps.FactorThresholds,
	})

	if !result.ScorePassed {
		if !deps.Quiet {
			_, _ = fmt.Fprintf(deps.Stderr, "Score %d is below threshold %d\n", report.ScoreValue(), deps.FailUnder)
		}
		deps.ExitFunc(1)
	}

	if !result.BaselinePassed {
		if !deps.Quiet {
			_, _ = fmt.Fprintf(deps.Stderr, "Score regressed from %d to %d (%d)\n", baselineReport.ScoreValue(), report.ScoreValue(), delta)
		}
		deps.ExitFunc(1)
	}

	if !result.FactorsPassed {
		if !deps.Quiet {
			for _, failure := range result.FactorFailures {
				_, _ = fmt.Fprintf(deps.Stderr, "Factor threshold failed: %s\n", failure)
			}
		}
		deps.ExitFunc(1)
	}
}

// baselineConfigFromGaugeDeps creates a BaselineConfig from GaugeDeps.
func baselineConfigFromGaugeDeps(deps *GaugeDeps) BaselineConfig {
	return BaselineConfig{
		CompareBaseline:      deps.CompareBaseline,
		Compare:              deps.Compare,
		BaselineRef:          deps.BaselineRef,
		BaselineFile:         deps.BaselineFile,
		FS:                   deps.FS,
		IsGitRepo:            deps.IsGitRepo,
		BaselineGitRefReader: deps.BaselineGitRefReader,
		BaselineFileReader:   deps.BaselineFileReader,
	}
}

// emitJSONMetadata writes JSON metadata to a separate file.
func emitJSONMetadata(deps *GaugeDeps, report *confidence.Report, baselineReport *confidence.Report, delta int) (err error) {
	f, err := deps.FS.Create(deps.EmitJSON)
	if err != nil {
		return fmt.Errorf("creating emit-json file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing emit-json file: %w", cerr)
		}
	}()
	return writeJSON(f, report, baselineReport, delta)
}

