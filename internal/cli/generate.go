package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/dashboard"
	"github.com/boinger/confvis/internal/gauge"
)

var (
	genConfig    string
	genOutput    string
	genDark      bool
	genFailUnder int
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate badge and dashboard",
	Long: `Generate both an SVG badge and HTML dashboard from a confidence report.

Creates:
  <output>/badge.svg         - SVG gauge badge
  <output>/dashboard/index.html - HTML dashboard`,
	RunE: runGenerate,
}

func init() {
	generateCmd.Flags().StringVarP(&genConfig, "config", "c", "", "path to confidence report JSON, or - for stdin (required)")
	generateCmd.Flags().StringVarP(&genOutput, "output", "o", "", "output directory (required)")
	generateCmd.Flags().BoolVar(&genDark, "dark", false, "use dark mode colors")
	generateCmd.Flags().IntVar(&genFailUnder, "fail-under", 0, "exit non-zero if score is below this value")

	if err := generateCmd.MarkFlagRequired("config"); err != nil {
		panic(err)
	}
	if err := generateCmd.MarkFlagRequired("output"); err != nil {
		panic(err)
	}

	rootCmd.AddCommand(generateCmd)
}

func runGenerate(_ *cobra.Command, _ []string) error {
	var report *confidence.Report
	var err error

	if genConfig == "-" {
		report, err = confidence.Parse(os.Stdin)
	} else {
		report, err = confidence.ParseFile(genConfig)
	}
	if err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	showVerbose := verbose && !quiet

	if showVerbose {
		fmt.Printf("Generating output for %q (score: %d, threshold: %d)\n",
			report.Title, report.Score, report.Threshold)
	}

	// Create output directories
	dashboardDir := filepath.Join(genOutput, "dashboard")
	if err := os.MkdirAll(dashboardDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Generate badge.svg
	badgePath := filepath.Join(genOutput, "badge.svg")
	if err := generateBadge(badgePath, report, genDark, showVerbose); err != nil {
		return err
	}

	// Generate dashboard/index.html
	dashboardPath := filepath.Join(dashboardDir, "index.html")
	if err := generateDashboard(dashboardPath, report, genDark, showVerbose); err != nil {
		return err
	}

	if genFailUnder > 0 && report.Score < genFailUnder {
		if !quiet {
			fmt.Fprintf(os.Stderr, "Score %d is below threshold %d\n", report.Score, genFailUnder)
		}
		os.Exit(1)
	}

	return nil
}

func generateBadge(path string, report *confidence.Report, dark, verbose bool) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating badge file: %w", err)
	}

	opts := gauge.Options{
		Width:    200,
		Height:   120,
		DarkMode: dark,
	}
	if err := gauge.Generate(f, report, opts); err != nil {
		_ = f.Close()
		return fmt.Errorf("generating badge: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("closing badge file: %w", err)
	}

	if verbose {
		fmt.Printf("Wrote badge to %s\n", path)
	}
	return nil
}

func generateDashboard(path string, report *confidence.Report, dark, verbose bool) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating dashboard file: %w", err)
	}

	opts := dashboard.Options{
		DarkMode: dark,
	}
	if err := dashboard.Generate(f, report, opts); err != nil {
		_ = f.Close()
		return fmt.Errorf("generating dashboard: %w", err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("closing dashboard file: %w", err)
	}

	if verbose {
		fmt.Printf("Wrote dashboard to %s\n", path)
	}
	return nil
}
