package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/gauge"
)

var (
	gaugeConfig    string
	gaugeOutput    string
	gaugeWidth     int
	gaugeHeight    int
	gaugeDark      bool
	gaugeFailUnder int
)

var gaugeCmd = &cobra.Command{
	Use:   "gauge",
	Short: "Generate an SVG gauge badge",
	Long:  `Generate an SVG gauge badge from a confidence report JSON file.`,
	RunE:  runGauge,
}

func init() {
	gaugeCmd.Flags().StringVarP(&gaugeConfig, "config", "c", "", "path to confidence report JSON, or - for stdin (required)")
	gaugeCmd.Flags().StringVarP(&gaugeOutput, "output", "o", "", "output SVG file path, or - for stdout (required)")
	gaugeCmd.Flags().IntVar(&gaugeWidth, "width", 200, "gauge width in pixels")
	gaugeCmd.Flags().IntVar(&gaugeHeight, "height", 120, "gauge height in pixels")
	gaugeCmd.Flags().BoolVar(&gaugeDark, "dark", false, "use dark mode colors")
	gaugeCmd.Flags().IntVar(&gaugeFailUnder, "fail-under", 0, "exit non-zero if score is below this value")

	if err := gaugeCmd.MarkFlagRequired("config"); err != nil {
		panic(err)
	}
	if err := gaugeCmd.MarkFlagRequired("output"); err != nil {
		panic(err)
	}

	rootCmd.AddCommand(gaugeCmd)
}

func runGauge(_ *cobra.Command, _ []string) error {
	var report *confidence.Report
	var err error

	if gaugeConfig == "-" {
		report, err = confidence.Parse(os.Stdin)
	} else {
		report, err = confidence.ParseFile(gaugeConfig)
	}
	if err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	// Suppress verbose output when writing to stdout
	outputToStdout := gaugeOutput == "-"
	showVerbose := verbose && !quiet && !outputToStdout

	if showVerbose {
		fmt.Printf("Generating gauge for %q (score: %d, threshold: %d)\n",
			report.Title, report.Score, report.Threshold)
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

	opts := gauge.Options{
		Width:    gaugeWidth,
		Height:   gaugeHeight,
		DarkMode: gaugeDark,
	}

	if err := gauge.Generate(w, report, opts); err != nil {
		return fmt.Errorf("generating gauge: %w", err)
	}

	if showVerbose {
		fmt.Printf("Wrote gauge to %s\n", gaugeOutput)
	}

	if gaugeFailUnder > 0 && report.Score < gaugeFailUnder {
		if !quiet {
			fmt.Fprintf(os.Stderr, "Score %d is below threshold %d\n", report.Score, gaugeFailUnder)
		}
		os.Exit(1)
	}

	return nil
}
