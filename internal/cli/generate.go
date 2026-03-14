package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/dashboard"
	"github.com/boinger/confvis/internal/gauge"
)

var (
	genConfig      string
	genOutput      string
	genDark        bool
	genFailUnder   int
	genInputFormat string
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate badge and dashboard",
	Long: `Generate both an SVG badge and HTML dashboard from a confidence report (JSON or YAML).

Creates:
  <output>/badge.svg         - SVG gauge badge
  <output>/dashboard/index.html - HTML dashboard`,
	RunE: runGenerate,
}

func init() {
	generateCmd.Flags().StringVarP(&genConfig, "config", "c", "", "path to confidence report (JSON/YAML), or - for stdin (required)")
	generateCmd.Flags().StringVar(&genInputFormat, "input-format", "auto", "input format: auto, json, or yaml (auto-detects from extension)")
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

// GenerateDeps contains dependencies for the generate command.
type GenerateDeps struct {
	FS          FileSystem
	Stdin       io.Reader
	Stderr      io.Writer
	Verbose     bool
	Quiet       bool
	ExitFunc    func(int)
	Config      string
	Output      string
	InputFormat string
	Dark        bool
	FailUnder   int
}

func runGenerate(_ *cobra.Command, _ []string) error {
	return generateImpl(&GenerateDeps{
		FS:          DefaultFileSystem,
		Stdin:       os.Stdin,
		Stderr:      os.Stderr,
		Verbose:     verbose,
		Quiet:       quiet,
		ExitFunc:    os.Exit,
		Config:      genConfig,
		Output:      genOutput,
		InputFormat: genInputFormat,
		Dark:        genDark,
		FailUnder:   genFailUnder,
	})
}

func generateImpl(deps *GenerateDeps) error {
	// Validate and convert input format
	inputFormat, err := ParseInputFormat(deps.InputFormat)
	if err != nil {
		return err
	}

	loader := &ReportLoader{FS: deps.FS, Stdin: deps.Stdin, Config: deps.Config, Format: inputFormat}
	report, err := loader.LoadReport()
	if err != nil {
		return err
	}

	showVerbose := deps.Verbose && !deps.Quiet

	if showVerbose {
		fmt.Printf("Generating output for %q (score: %d, threshold: %d)\n",
			report.Title, report.ScoreValue(), report.Threshold)
	}

	// Create output directories
	dashboardDir := filepath.Join(deps.Output, "dashboard")
	if err := deps.FS.MkdirAll(dashboardDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Generate badge.svg
	badgePath := filepath.Join(deps.Output, "badge.svg")
	if err := generateBadgeWithFS(deps.FS, badgePath, report, deps.Dark, showVerbose); err != nil {
		return err
	}

	// Generate dashboard/index.html
	dashboardPath := filepath.Join(dashboardDir, "index.html")
	if err := generateDashboardWithFS(deps.FS, dashboardPath, report, deps.Dark, showVerbose); err != nil {
		return err
	}

	if deps.FailUnder > 0 && report.ScoreValue() < deps.FailUnder {
		if !deps.Quiet {
			_, _ = fmt.Fprintf(deps.Stderr, "Score %d is below threshold %d\n", report.ScoreValue(), deps.FailUnder)
		}
		deps.ExitFunc(1)
	}

	return nil
}

// openConfigFile opens a config file using the provided FileSystem and determines its format.
func openConfigFile(fs FileSystem, path string, format confidence.Format) (io.Reader, confidence.Format, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, format, fmt.Errorf("opening file: %w", err)
	}

	// Read all content into memory so we can return it as a reader
	// The file handle will be closed but the content is available
	content, err := io.ReadAll(f)
	if cerr := f.Close(); cerr != nil && err == nil {
		err = cerr
	}
	if err != nil {
		return nil, format, fmt.Errorf("reading file: %w", err)
	}

	// Auto-detect format from extension
	if format == confidence.FormatAuto {
		format = confidence.DetectFormat(path)
	}

	return bytes.NewReader(content), format, nil
}


// writeToFileWithFS creates a file, calls the generator function, closes the file,
// and optionally prints a verbose message. It handles error cleanup on the file handle.
func writeToFileWithFS(fs FileSystem, path string, verbose bool, label string, fn func(io.Writer) error) error {
	f, err := fs.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s file: %w", label, err)
	}
	if err := fn(f); err != nil {
		_ = f.Close()
		return fmt.Errorf("generating %s: %w", label, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing %s file: %w", label, err)
	}
	if verbose {
		fmt.Printf("Wrote %s to %s\n", label, path)
	}
	return nil
}

// encodeJSONIndented writes v as pretty-printed JSON to w.
func encodeJSONIndented(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	return nil
}

func generateBadge(path string, report *confidence.Report, dark, verbose bool) error {
	return generateBadgeWithFS(DefaultFileSystem, path, report, dark, verbose)
}

func generateBadgeWithFS(fs FileSystem, path string, report *confidence.Report, dark, verbose bool) error {
	return writeToFileWithFS(fs, path, verbose, "badge", func(w io.Writer) error {
		return gauge.Generate(w, report, gauge.Options{
			ColorOptions: gauge.ColorOptions{DarkMode: dark},
			Width:        200,
			Height:       120,
		})
	})
}

func generateDashboard(path string, report *confidence.Report, dark, verbose bool) error {
	return generateDashboardWithFS(DefaultFileSystem, path, report, dark, verbose)
}

func generateDashboardWithFS(fs FileSystem, path string, report *confidence.Report, dark, verbose bool) error {
	return writeToFileWithFS(fs, path, verbose, "dashboard", func(w io.Writer) error {
		return dashboard.Generate(w, report, dashboard.Options{DarkMode: dark})
	})
}
