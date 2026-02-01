package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/gauge"
)

var (
	gaugeConfig      string
	gaugeOutput      string
	gaugeWidth       int
	gaugeHeight      int
	gaugeDark        bool
	gaugeFailUnder   int
	gaugeFormat      string
	gaugeStyle       string
	gaugeGreenAbove  int
	gaugeYellowAbove int
)

var gaugeCmd = &cobra.Command{
	Use:   "gauge",
	Short: "Generate an SVG gauge badge",
	Long:  `Generate an SVG gauge badge from a confidence report JSON file.`,
	RunE:  runGauge,
}

func init() {
	gaugeCmd.Flags().StringVarP(&gaugeConfig, "config", "c", "", "path to confidence report JSON, or - for stdin (required)")
	gaugeCmd.Flags().StringVarP(&gaugeOutput, "output", "o", "", "output file path, or - for stdout (required)")
	gaugeCmd.Flags().StringVarP(&gaugeFormat, "format", "f", "svg", "output format: svg, json, or text")
	gaugeCmd.Flags().IntVar(&gaugeWidth, "width", 200, "gauge width in pixels (svg only)")
	gaugeCmd.Flags().IntVar(&gaugeHeight, "height", 120, "gauge height in pixels (svg only)")
	gaugeCmd.Flags().StringVar(&gaugeStyle, "style", "github", "color scheme: github, minimal, corporate, high-contrast (svg only)")
	gaugeCmd.Flags().BoolVar(&gaugeDark, "dark", false, "use dark mode colors (svg only)")
	gaugeCmd.Flags().IntVar(&gaugeFailUnder, "fail-under", 0, "exit non-zero if score is below this value")
	gaugeCmd.Flags().IntVar(&gaugeGreenAbove, "green-above", 0, "score threshold for green color (overrides JSON config)")
	gaugeCmd.Flags().IntVar(&gaugeYellowAbove, "yellow-above", 0, "score threshold for yellow color (overrides JSON config)")

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
	case "svg", "json", "text":
		// valid
	default:
		return fmt.Errorf("invalid format %q: must be svg, json, or text", gaugeFormat)
	}

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

	case "json":
		output := struct {
			Title       string `json:"title"`
			Score       int    `json:"score"`
			Threshold   int    `json:"threshold"`
			Passed      bool   `json:"passed"`
			Version     string `json:"version,omitempty"`
			GeneratedAt string `json:"generatedAt,omitempty"`
			Source      string `json:"source,omitempty"`
		}{
			Title:       report.Title,
			Score:       report.Score,
			Threshold:   report.Threshold,
			Passed:      report.Passed(),
			Version:     report.Version,
			GeneratedAt: report.GeneratedAt,
			Source:      report.Source,
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(output); err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}

	case "text":
		if _, err := fmt.Fprintf(w, "%d\n", report.Score); err != nil {
			return fmt.Errorf("writing text output: %w", err)
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

	return nil
}
