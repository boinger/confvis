package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/gauge"
)

var (
	gaugeConfig string
	gaugeOutput string
	gaugeWidth  int
	gaugeHeight int
	gaugeDark   bool
)

var gaugeCmd = &cobra.Command{
	Use:   "gauge",
	Short: "Generate an SVG gauge badge",
	Long:  `Generate an SVG gauge badge from a confidence report JSON file.`,
	RunE:  runGauge,
}

func init() {
	gaugeCmd.Flags().StringVarP(&gaugeConfig, "config", "c", "", "path to confidence report JSON (required)")
	gaugeCmd.Flags().StringVarP(&gaugeOutput, "output", "o", "", "output SVG file path (required)")
	gaugeCmd.Flags().IntVar(&gaugeWidth, "width", 200, "gauge width in pixels")
	gaugeCmd.Flags().IntVar(&gaugeHeight, "height", 120, "gauge height in pixels")
	gaugeCmd.Flags().BoolVar(&gaugeDark, "dark", false, "use dark mode colors")

	if err := gaugeCmd.MarkFlagRequired("config"); err != nil {
		panic(err)
	}
	if err := gaugeCmd.MarkFlagRequired("output"); err != nil {
		panic(err)
	}

	rootCmd.AddCommand(gaugeCmd)
}

func runGauge(_ *cobra.Command, _ []string) error {
	report, err := confidence.ParseFile(gaugeConfig)
	if err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	if verbose {
		fmt.Printf("Generating gauge for %q (score: %d, threshold: %d)\n",
			report.Title, report.Score, report.Threshold)
	}

	f, err := os.Create(gaugeOutput)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}

	opts := gauge.Options{
		Width:    gaugeWidth,
		Height:   gaugeHeight,
		DarkMode: gaugeDark,
	}

	if err := gauge.Generate(f, report, opts); err != nil {
		_ = f.Close()
		return fmt.Errorf("generating gauge: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("closing output file: %w", err)
	}

	if verbose {
		fmt.Printf("Wrote gauge to %s\n", gaugeOutput)
	}

	return nil
}
