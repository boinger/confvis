package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/boinger/confvis/internal/baseline"
	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/gitutil"
)

var (
	gateConfig           string
	gateFailUnder        int
	gateFailOnRegression bool
	gateFactorThresholds []string
	gateCompare          string
	gateCompareBaseline  bool
	gateBaselineRef      string
	gateBaselineFile     string
	gateInputFormat      string
)

var gateCmd = &cobra.Command{
	Use:   "gate",
	Short: "CI gate: check thresholds and exit non-zero on failure",
	Long: `Check confidence report thresholds for CI gating.

Exits 0 if all threshold checks pass, 1 if any fail.
At least one threshold flag is required (--fail-under, --fail-on-regression, or --factor-threshold).

Unlike gauge, this command produces no badge output — it is purpose-built for CI pass/fail gating.`,
	RunE: runGate,
}

func init() {
	gateCmd.Flags().StringVarP(&gateConfig, "config", "c", "", "path to confidence report (JSON/YAML), or - for stdin (required)")
	gateCmd.Flags().StringVar(&gateInputFormat, "input-format", "auto", "input format: auto, json, or yaml (auto-detects from extension)")
	gateCmd.Flags().IntVar(&gateFailUnder, "fail-under", 0, "exit non-zero if score is below this value")
	gateCmd.Flags().BoolVar(&gateFailOnRegression, "fail-on-regression", false, "exit non-zero if score decreased from baseline")
	gateCmd.Flags().StringSliceVar(&gateFactorThresholds, "factor-threshold", nil, "per-factor threshold in 'Name:threshold' format (can be repeated)")
	gateCmd.Flags().StringVar(&gateCompare, "compare", "", "path to baseline report JSON for comparison")
	gateCmd.Flags().BoolVar(&gateCompareBaseline, "compare-baseline", false, "auto-fetch baseline from ref/file and compare")
	gateCmd.Flags().StringVar(&gateBaselineRef, "baseline-ref", "", "git ref for baseline storage (default: refs/confvis/baseline)")
	gateCmd.Flags().StringVar(&gateBaselineFile, "baseline-file", "", "file path for baseline")

	if err := gateCmd.MarkFlagRequired("config"); err != nil {
		panic(err)
	}

	rootCmd.AddCommand(gateCmd)
}

// GateDeps contains dependencies for the gate command.
type GateDeps struct {
	FS               FileSystem
	Stdin            io.Reader
	Stdout           io.Writer
	Stderr           io.Writer
	Verbose          bool
	Quiet            bool
	ExitFunc         func(int)
	Config           string
	InputFormat      string
	FailUnder        int
	FailOnRegression bool
	FactorThresholds map[string]int
	Baseline         BaselineConfig
}

func runGate(_ *cobra.Command, _ []string) error {
	factorThresholds, err := parseFactorThresholds(gateFactorThresholds, getGaugeFactorThresholds())
	if err != nil {
		return err
	}

	return gateImpl(&GateDeps{
		FS:               DefaultFileSystem,
		Stdin:            os.Stdin,
		Stdout:           os.Stdout,
		Stderr:           os.Stderr,
		Verbose:          verbose,
		Quiet:            quiet,
		ExitFunc:         os.Exit,
		Config:           gateConfig,
		InputFormat:      gateInputFormat,
		FailUnder:        getGateFailUnder(),
		FailOnRegression: gateFailOnRegression,
		FactorThresholds: factorThresholds,
		Baseline: BaselineConfig{
			CompareBaseline:      getGateCompareBaseline(),
			Compare:              gateCompare,
			BaselineRef:          getGateBaselineRef(),
			BaselineFile:         getGateBaselineFile(),
			FS:                   DefaultFileSystem,
			IsGitRepo:            gitutil.IsGitRepo,
			BaselineGitRefReader: baseline.ReadFromGitRef,
			BaselineFileReader:   baseline.ReadFromFile,
		},
	})
}

func gateImpl(deps *GateDeps) error {
	// Validate: at least one threshold flag is required
	if deps.FailUnder == 0 && !deps.FailOnRegression && len(deps.FactorThresholds) == 0 {
		return fmt.Errorf("at least one threshold flag is required (--fail-under, --fail-on-regression, or --factor-threshold)")
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

	deps.Baseline.FS = deps.FS
	baselineReport, delta, err := LoadBaseline(deps.Baseline, report.ScoreValue())
	if err != nil {
		return err
	}

	result := CheckThresholds(report, baselineReport, delta, ThresholdConfig{
		FailUnder:        deps.FailUnder,
		FailOnRegression: deps.FailOnRegression,
		FactorThresholds: deps.FactorThresholds,
	})

	if !deps.Quiet {
		writeGateSummary(deps.Stdout, report, baselineReport, delta, deps, result)
	}

	if !result.Passed() {
		deps.ExitFunc(1)
	}

	return nil
}

// writeGateSummary writes the gate command output.
func writeGateSummary(w io.Writer, report *confidence.Report, baselineReport *confidence.Report, delta int, deps *GateDeps, result ThresholdResult) {
	_, _ = fmt.Fprintf(w, "Score: %d/100\n", report.ScoreValue())

	if deps.Verbose {
		for _, f := range report.Factors {
			_, _ = fmt.Fprintf(w, "  %-12s %d (weight %d)\n", f.Name+":", f.Score, f.Weight)
		}
	}

	if deps.FailUnder > 0 {
		mark := passMarkFor(result.ScorePassed)
		_, _ = fmt.Fprintf(w, "Threshold: %d %s\n", deps.FailUnder, mark)
	}

	if baselineReport != nil {
		mark := passMarkFor(result.BaselinePassed)
		sign := "+"
		if delta < 0 {
			sign = ""
		}
		_, _ = fmt.Fprintf(w, "Baseline: %d → %d (%s%d) %s\n",
			baselineReport.ScoreValue(), report.ScoreValue(), sign, delta, mark)
	}

	if !result.FactorsPassed {
		for _, failure := range result.FactorFailures {
			_, _ = fmt.Fprintf(w, "Factor: %s ✗\n", failure)
		}
	}
}

func passMarkFor(passed bool) string {
	if passed {
		return "✓"
	}
	return "✗"
}
