package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/boinger/confvis/internal/baseline"
	"github.com/boinger/confvis/internal/confidence"
)

var (
	baselineConfig string
	baselineRef    string
	baselineFile   string
	baselineFormat string
	baselineDryRun bool
)

var baselineCmd = &cobra.Command{
	Use:   "baseline",
	Short: "Manage confidence baselines for regression detection",
	Long: `Manage confidence baselines for regression detection.

Baselines are saved confidence scores that can be used to detect
regressions in CI/CD pipelines. They can be stored in git refs
(default) or files.`,
}

var baselineSaveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save current confidence report as baseline",
	Long: `Save current confidence report as baseline.

The baseline is saved with metadata including the timestamp,
current git commit, and branch. By default, baselines are stored
in a git ref (refs/confvis/baseline), which keeps them out of your
working tree but accessible across branches.`,
	Example: `  # Save baseline to default git ref
  confvis baseline save -c confidence.json

  # Save to custom git ref
  confvis baseline save -c confidence.json --ref refs/confvis/prod-baseline

  # Save to file instead of git ref
  confvis baseline save -c confidence.json --file baseline.json`,
	RunE: runBaselineSave,
}

var baselineShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display the current baseline",
	Long: `Display the current baseline.

Shows the saved baseline score along with metadata like when it
was saved and from which commit/branch.`,
	Example: `  # Show baseline from default git ref
  confvis baseline show

  # Show baseline from custom git ref
  confvis baseline show --ref refs/confvis/prod-baseline

  # Show baseline from file
  confvis baseline show --file baseline.json

  # Output as JSON
  confvis baseline show --format json`,
	RunE: runBaselineShow,
}

func init() {
	// Save command flags
	baselineSaveCmd.Flags().StringVarP(&baselineConfig, "config", "c", "", "path to confidence report (JSON/YAML) (required)")
	baselineSaveCmd.Flags().StringVar(&baselineRef, "ref", "", "git ref for storage (default: refs/confvis/baseline)")
	baselineSaveCmd.Flags().StringVar(&baselineFile, "file", "", "file path alternative to git ref")
	baselineSaveCmd.Flags().BoolVar(&baselineDryRun, "dry-run", false, "preview what would be saved without writing")

	if err := baselineSaveCmd.MarkFlagRequired("config"); err != nil {
		panic(err)
	}

	// Show command flags
	baselineShowCmd.Flags().StringVar(&baselineRef, "ref", "", "git ref for storage (default: refs/confvis/baseline)")
	baselineShowCmd.Flags().StringVar(&baselineFile, "file", "", "file path alternative to git ref")
	baselineShowCmd.Flags().StringVar(&baselineFormat, "format", "text", "output format: text or json")

	// Bind flags to viper for config file support
	bindBaselineFlags(baselineSaveCmd)
	bindBaselineFlags(baselineShowCmd)

	baselineCmd.AddCommand(baselineSaveCmd)
	baselineCmd.AddCommand(baselineShowCmd)
	rootCmd.AddCommand(baselineCmd)
}

// bindBaselineFlags binds baseline command flags to viper configuration keys.
func bindBaselineFlags(cmd *cobra.Command) {
	if cmd.Flags().Lookup("ref") != nil {
		_ = viper.BindPFlag("baseline.ref", cmd.Flags().Lookup("ref"))
	}
	if cmd.Flags().Lookup("file") != nil {
		_ = viper.BindPFlag("baseline.file", cmd.Flags().Lookup("file"))
	}
}

// BaselineDeps contains dependencies for baseline commands.
type BaselineDeps struct {
	FS            FileSystem
	Stdin         io.Reader
	Stdout        io.Writer
	Stderr        io.Writer
	Verbose       bool
	Quiet         bool
	Config        string
	Ref           string
	File          string
	Format        string
	DryRun        bool
	IsGitRepo     func() bool
	GitRefReader  func(string) (*baseline.Baseline, error)
	GitRefWriter  func(string, *baseline.Baseline) error
	FileReader    func(string) (*baseline.Baseline, error)
	FileWriter    func(string, *baseline.Baseline) error
}

func runBaselineSave(_ *cobra.Command, _ []string) error {
	return baselineSaveImpl(&BaselineDeps{
		FS:           DefaultFileSystem,
		Stdin:        os.Stdin,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
		Verbose:      verbose,
		Quiet:        quiet,
		Config:       baselineConfig,
		Ref:          getBaselineRef(),
		File:         getBaselineFile(),
		DryRun:       baselineDryRun,
		IsGitRepo:    baseline.IsGitRepo,
		GitRefReader: baseline.ReadFromGitRef,
		GitRefWriter: baseline.WriteToGitRef,
		FileReader:   baseline.ReadFromFile,
		FileWriter:   baseline.WriteToFile,
	})
}

func runBaselineShow(_ *cobra.Command, _ []string) error {
	return baselineShowImpl(&BaselineDeps{
		FS:           DefaultFileSystem,
		Stdin:        os.Stdin,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
		Verbose:      verbose,
		Quiet:        quiet,
		Ref:          getBaselineRef(),
		File:         getBaselineFile(),
		Format:       baselineFormat,
		IsGitRepo:    baseline.IsGitRepo,
		GitRefReader: baseline.ReadFromGitRef,
		GitRefWriter: baseline.WriteToGitRef,
		FileReader:   baseline.ReadFromFile,
		FileWriter:   baseline.WriteToFile,
	})
}

func baselineSaveImpl(deps *BaselineDeps) error {
	// Determine input format from filename
	inputFormat := confidence.FormatAuto
	if deps.Config == "-" {
		inputFormat = confidence.FormatJSON
	}

	// Read the confidence report
	report, err := parseBaselineConfig(deps, inputFormat)
	if err != nil {
		return err
	}

	// Create baseline from report
	b := baseline.NewBaseline(report)

	// Determine storage mode: file takes precedence, otherwise use git ref
	useFile := deps.File != ""
	useGitRef := !useFile && (deps.IsGitRepo == nil || deps.IsGitRepo())

	if !useFile && !useGitRef {
		return fmt.Errorf("not in a git repository and no --file specified")
	}

	return saveBaseline(deps, b, useFile)
}

func parseBaselineConfig(deps *BaselineDeps, inputFormat confidence.Format) (*confidence.Report, error) {
	if deps.Config == "-" {
		report, err := confidence.ParseWithFormat(deps.Stdin, inputFormat)
		if err != nil {
			return nil, fmt.Errorf("parsing config: %w", err)
		}
		return report, nil
	}

	reader, format, err := openConfigFile(deps.FS, deps.Config, inputFormat)
	if err != nil {
		return nil, fmt.Errorf("opening config: %w", err)
	}
	report, err := confidence.ParseWithFormat(reader, format)
	if err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return report, nil
}

func saveBaseline(deps *BaselineDeps, b *baseline.Baseline, useFile bool) error {
	// Dry-run mode: show what would be saved without writing
	if deps.DryRun {
		outputBaselineDryRun(deps, b, useFile)
		return nil
	}

	if useFile {
		if err := deps.FileWriter(deps.File, b); err != nil {
			return fmt.Errorf("saving baseline to file: %w", err)
		}
		if deps.Verbose && !deps.Quiet {
			_, _ = fmt.Fprintf(deps.Stdout, "Saved baseline to %s (score: %d, commit: %s)\n",
				deps.File, b.Score, truncateCommit(b.Commit))
		}
		return nil
	}

	if err := deps.GitRefWriter(deps.Ref, b); err != nil {
		return fmt.Errorf("saving baseline to git ref: %w", err)
	}
	if deps.Verbose && !deps.Quiet {
		_, _ = fmt.Fprintf(deps.Stdout, "Saved baseline to %s (score: %d, commit: %s)\n",
			deps.Ref, b.Score, truncateCommit(b.Commit))
	}
	return nil
}

func outputBaselineDryRun(deps *BaselineDeps, b *baseline.Baseline, useFile bool) {
	destination := deps.Ref
	destType := "git ref"
	if useFile {
		destination = deps.File
		destType = "file"
	}

	_, _ = fmt.Fprintln(deps.Stdout, "DRY RUN: Would save baseline")
	_, _ = fmt.Fprintln(deps.Stdout)
	_, _ = fmt.Fprintf(deps.Stdout, "Destination: %s (%s)\n", destination, destType)
	_, _ = fmt.Fprintln(deps.Stdout)
	_, _ = fmt.Fprintln(deps.Stdout, "Baseline content:")
	_, _ = fmt.Fprintf(deps.Stdout, "  Score:   %d\n", b.Score)
	_, _ = fmt.Fprintf(deps.Stdout, "  Title:   %s\n", b.Title)
	if b.Commit != "" {
		_, _ = fmt.Fprintf(deps.Stdout, "  Commit:  %s\n", truncateCommit(b.Commit))
	}
	if b.Branch != "" {
		_, _ = fmt.Fprintf(deps.Stdout, "  Branch:  %s\n", b.Branch)
	}
	_, _ = fmt.Fprintf(deps.Stdout, "  SavedAt: %s\n", b.SavedAt)
	_, _ = fmt.Fprintln(deps.Stdout)
	_, _ = fmt.Fprintln(deps.Stdout, "No changes made.")
}

func baselineShowImpl(deps *BaselineDeps) error {
	// Validate format
	switch deps.Format {
	case "text", "json":
		// valid
	default:
		return fmt.Errorf("invalid format %q: must be text or json", deps.Format)
	}

	b, source, err := loadBaseline(deps)
	if err != nil {
		return err
	}

	if b == nil {
		return fmt.Errorf("no baseline found at %s", source)
	}

	return outputBaseline(deps, b)
}

func loadBaseline(deps *BaselineDeps) (*baseline.Baseline, string, error) {
	useFile := deps.File != ""
	useGitRef := !useFile && (deps.IsGitRepo == nil || deps.IsGitRepo())

	if !useFile && !useGitRef {
		return nil, "", fmt.Errorf("not in a git repository and no --file specified")
	}

	if useFile {
		b, err := deps.FileReader(deps.File)
		if err != nil {
			return nil, "", fmt.Errorf("reading baseline from file: %w", err)
		}
		return b, deps.File, nil
	}

	b, err := deps.GitRefReader(deps.Ref)
	if err != nil {
		return nil, "", fmt.Errorf("reading baseline from git ref: %w", err)
	}
	return b, deps.Ref, nil
}

func outputBaseline(deps *BaselineDeps, b *baseline.Baseline) error {
	switch deps.Format {
	case "json":
		enc := json.NewEncoder(deps.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(b); err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
	case "text":
		_, _ = fmt.Fprintf(deps.Stdout, "Baseline: %d%%", b.Score)
		if b.SavedAt != "" {
			_, _ = fmt.Fprintf(deps.Stdout, " (saved %s", b.SavedAt)
			if b.Commit != "" {
				_, _ = fmt.Fprintf(deps.Stdout, ", commit %s", truncateCommit(b.Commit))
			}
			if b.Branch != "" {
				_, _ = fmt.Fprintf(deps.Stdout, ", branch %s", b.Branch)
			}
			_, _ = fmt.Fprint(deps.Stdout, ")")
		}
		_, _ = fmt.Fprintln(deps.Stdout)
	}
	return nil
}

// truncateCommit returns the first 7 characters of a commit hash.
func truncateCommit(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}

// getBaselineRef returns the baseline ref from config/env/flag with defaults.
func getBaselineRef() string {
	if v := viper.GetString("baseline.ref"); v != "" {
		return v
	}
	return baseline.DefaultBaselineRef
}

// getBaselineFile returns the baseline file from config/env/flag.
func getBaselineFile() string {
	return viper.GetString("baseline.file")
}
