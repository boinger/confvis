package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/dashboard"
	"github.com/boinger/confvis/internal/gauge"
)

var (
	aggConfigs   []string
	aggOutput    string
	aggDark      bool
	aggFailUnder int
	aggBadgeType string
)

var aggregateCmd = &cobra.Command{
	Use:   "aggregate",
	Short: "Aggregate multiple reports into a single dashboard",
	Long: `Aggregate multiple confidence reports into a single dashboard.

Each report becomes a component in the aggregate view, with an overall
score calculated as a weighted average.

Config format: path[:weight]
  - path: Path to confidence report (JSON/YAML), or glob pattern
  - weight: Optional weight (default 100)

Examples:
  confvis aggregate -c api/confidence.json -c web/confidence.json -o ./dashboard
  confvis aggregate -c "services/*/confidence.json" -o ./dashboard
  confvis aggregate -c api/confidence.json:60 -c web/confidence.json:40 -o ./dashboard`,
	RunE: runAggregate,
}

func init() {
	aggregateCmd.Flags().StringArrayVarP(&aggConfigs, "config", "c", nil, "config file path[:weight] or glob pattern (can be repeated)")
	aggregateCmd.Flags().StringVarP(&aggOutput, "output", "o", "", "output directory (required)")
	aggregateCmd.Flags().BoolVar(&aggDark, "dark", false, "use dark mode colors")
	aggregateCmd.Flags().IntVar(&aggFailUnder, "fail-under", 0, "exit non-zero if aggregate score is below this value")
	aggregateCmd.Flags().StringVar(&aggBadgeType, "badge-type", "gauge", "badge type: gauge or flat")

	if err := aggregateCmd.MarkFlagRequired("config"); err != nil {
		panic(err)
	}
	if err := aggregateCmd.MarkFlagRequired("output"); err != nil {
		panic(err)
	}

	rootCmd.AddCommand(aggregateCmd)
}

// reportWithWeight holds a parsed report and its weight for aggregation.
type reportWithWeight struct {
	Report *confidence.Report
	Weight int
	Path   string
}

// AggregateDeps contains dependencies for the aggregate command.
type AggregateDeps struct {
	FS        FileSystem
	Stderr    io.Writer
	Verbose   bool
	Quiet     bool
	ExitFunc  func(int)
	Configs   []string
	Output    string
	Dark      bool
	FailUnder int
	BadgeType string
}

func runAggregate(_ *cobra.Command, _ []string) error {
	return aggregateImpl(&AggregateDeps{
		FS:        DefaultFileSystem,
		Stderr:    os.Stderr,
		Verbose:   verbose,
		Quiet:     quiet,
		ExitFunc:  os.Exit,
		Configs:   aggConfigs,
		Output:    aggOutput,
		Dark:      aggDark,
		FailUnder: aggFailUnder,
		BadgeType: aggBadgeType,
	})
}

func aggregateImpl(deps *AggregateDeps) error {
	// Parse all configs and expand globs
	reports, err := parseConfigsWithWeightsFS(deps.FS, deps.Configs)
	if err != nil {
		return err
	}

	if len(reports) == 0 {
		return fmt.Errorf("no reports found")
	}

	// Calculate aggregate score
	var totalWeight int
	var weightedSum int
	for _, r := range reports {
		totalWeight += r.Weight
		weightedSum += r.Report.Score * r.Weight
	}

	aggregateScore := 0
	if totalWeight > 0 {
		aggregateScore = (weightedSum + totalWeight/2) / totalWeight
	}

	showVerbose := deps.Verbose && !deps.Quiet

	if showVerbose {
		fmt.Printf("Aggregating %d reports (aggregate score: %d)\n", len(reports), aggregateScore)
		for _, r := range reports {
			fmt.Printf("  - %s: %d (weight: %d)\n", r.Report.Title, r.Report.Score, r.Weight)
		}
	}

	// Create output directories
	dashboardDir := filepath.Join(deps.Output, "dashboard")
	if err := deps.FS.MkdirAll(dashboardDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	// Build aggregate report for badge generation
	aggregateReport := &confidence.Report{
		Title:     "Aggregate",
		Score:     aggregateScore,
		Threshold: 75, // Default threshold
	}

	// Use lowest threshold from any report
	for _, r := range reports {
		if r.Report.Threshold > 0 && (aggregateReport.Threshold == 75 || r.Report.Threshold < aggregateReport.Threshold) {
			aggregateReport.Threshold = r.Report.Threshold
		}
	}

	// Generate aggregate badge
	badgePath := filepath.Join(deps.Output, "badge.svg")
	if err := generateAggregateBadgeWithFS(deps.FS, badgePath, aggregateReport, deps.Dark, deps.BadgeType, showVerbose); err != nil {
		return err
	}

	// Generate multi-dashboard
	dashboardPath := filepath.Join(dashboardDir, "index.html")
	if err := generateMultiDashboardWithFS(deps.FS, dashboardPath, reports, aggregateReport, deps.Dark, showVerbose); err != nil {
		return err
	}

	// Generate individual badges
	for i, r := range reports {
		safeName := sanitizeFilename(r.Report.Title)
		if safeName == "" {
			safeName = fmt.Sprintf("report_%d", i)
		}
		individualBadgePath := filepath.Join(deps.Output, fmt.Sprintf("%s.svg", safeName))
		if err := generateAggregateBadgeWithFS(deps.FS, individualBadgePath, r.Report, deps.Dark, deps.BadgeType, showVerbose); err != nil {
			return err
		}
	}

	if deps.FailUnder > 0 && aggregateScore < deps.FailUnder {
		if !deps.Quiet {
			_, _ = fmt.Fprintf(deps.Stderr, "Aggregate score %d is below threshold %d\n", aggregateScore, deps.FailUnder)
		}
		deps.ExitFunc(1)
	}

	return nil
}

// parseConfigsWithWeights parses config flags, expands globs, and extracts weights.
// Uses the default file system.
func parseConfigsWithWeights(configs []string) ([]reportWithWeight, error) {
	return parseConfigsWithWeightsFS(DefaultFileSystem, configs)
}

// parseConfigsWithWeightsFS parses config flags using the provided FileSystem.
func parseConfigsWithWeightsFS(fs FileSystem, configs []string) ([]reportWithWeight, error) {
	var results []reportWithWeight

	for _, cfg := range configs {
		// Parse path:weight format
		path := cfg
		weight := 100

		if idx := strings.LastIndex(cfg, ":"); idx > 0 {
			// Check if it's a weight suffix (number) not a Windows drive letter
			suffix := cfg[idx+1:]
			if w, err := strconv.Atoi(suffix); err == nil {
				path = cfg[:idx]
				weight = w
			}
		}

		// Expand glob pattern
		matches, err := fs.Glob(path)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", path, err)
		}

		// If no glob matches, treat as literal path
		if len(matches) == 0 {
			matches = []string{path}
		}

		for _, match := range matches {
			// Read file using the filesystem
			reader, format, err := openConfigFile(fs, match, confidence.FormatAuto)
			if err != nil {
				return nil, fmt.Errorf("parsing %q: %w", match, err)
			}

			report, err := confidence.ParseWithFormat(reader, format)
			if err != nil {
				return nil, fmt.Errorf("parsing %q: %w", match, err)
			}

			results = append(results, reportWithWeight{
				Report: report,
				Weight: weight,
				Path:   match,
			})
		}
	}

	return results, nil
}

func generateAggregateBadge(path string, report *confidence.Report, dark bool, badgeType string, verbose bool) error {
	return generateAggregateBadgeWithFS(DefaultFileSystem, path, report, dark, badgeType, verbose)
}

func generateAggregateBadgeWithFS(fs FileSystem, path string, report *confidence.Report, dark bool, badgeType string, verbose bool) error {
	f, err := fs.Create(path)
	if err != nil {
		return fmt.Errorf("creating badge file: %w", err)
	}

	var genErr error
	if badgeType == "flat" {
		flatOpts := gauge.FlatOptions{
			DarkMode: dark,
		}
		genErr = gauge.GenerateFlat(f, report, flatOpts)
	} else {
		opts := gauge.Options{
			Width:    200,
			Height:   120,
			DarkMode: dark,
		}
		genErr = gauge.Generate(f, report, opts)
	}

	if genErr != nil {
		_ = f.Close()
		return fmt.Errorf("generating badge: %w", genErr)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("closing badge file: %w", err)
	}

	if verbose {
		fmt.Printf("Wrote badge to %s\n", path)
	}
	return nil
}

func generateMultiDashboard(path string, reports []reportWithWeight, aggregate *confidence.Report, dark, verbose bool) error {
	return generateMultiDashboardWithFS(DefaultFileSystem, path, reports, aggregate, dark, verbose)
}

func generateMultiDashboardWithFS(fs FileSystem, path string, reports []reportWithWeight, aggregate *confidence.Report, dark, verbose bool) error {
	f, err := fs.Create(path)
	if err != nil {
		return fmt.Errorf("creating dashboard file: %w", err)
	}

	// Convert to format expected by dashboard
	var dashReports []dashboard.ReportSummary
	for _, r := range reports {
		dashReports = append(dashReports, dashboard.ReportSummary{
			Report: r.Report,
			Weight: r.Weight,
			Path:   r.Path,
		})
	}

	opts := dashboard.MultiOptions{
		DarkMode: dark,
	}
	if err := dashboard.GenerateMulti(f, dashReports, aggregate, opts); err != nil {
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

// sanitizeFilename creates a safe filename from a string.
func sanitizeFilename(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// openConfigFile is defined in generate.go, but we need to ensure it uses
// the format detection logic. Since it's already defined there, we'll reuse it.
// If the function uses a different filesystem interface, we need to ensure compatibility.

// detectConfigFormat is defined in generate.go
// If not, we define a local version here for aggregate-specific use
func init() {
	// Ensure openConfigFile and detectConfigFormat are available
	// They are defined in generate.go
}

