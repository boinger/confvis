package cli

import (
	"fmt"
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

func runAggregate(_ *cobra.Command, _ []string) error {
	// Parse all configs and expand globs
	reports, err := parseConfigsWithWeights(aggConfigs)
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

	showVerbose := verbose && !quiet

	if showVerbose {
		fmt.Printf("Aggregating %d reports (aggregate score: %d)\n", len(reports), aggregateScore)
		for _, r := range reports {
			fmt.Printf("  - %s: %d (weight: %d)\n", r.Report.Title, r.Report.Score, r.Weight)
		}
	}

	// Create output directories
	dashboardDir := filepath.Join(aggOutput, "dashboard")
	if err := os.MkdirAll(dashboardDir, 0o755); err != nil {
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
	badgePath := filepath.Join(aggOutput, "badge.svg")
	if err := generateAggregateBadge(badgePath, aggregateReport, aggDark, showVerbose); err != nil {
		return err
	}

	// Generate multi-dashboard
	dashboardPath := filepath.Join(dashboardDir, "index.html")
	if err := generateMultiDashboard(dashboardPath, reports, aggregateReport, aggDark, showVerbose); err != nil {
		return err
	}

	// Generate individual badges
	for i, r := range reports {
		safeName := sanitizeFilename(r.Report.Title)
		if safeName == "" {
			safeName = fmt.Sprintf("report_%d", i)
		}
		individualBadgePath := filepath.Join(aggOutput, fmt.Sprintf("%s.svg", safeName))
		if err := generateAggregateBadge(individualBadgePath, r.Report, aggDark, showVerbose); err != nil {
			return err
		}
	}

	if aggFailUnder > 0 && aggregateScore < aggFailUnder {
		if !quiet {
			fmt.Fprintf(os.Stderr, "Aggregate score %d is below threshold %d\n", aggregateScore, aggFailUnder)
		}
		os.Exit(1)
	}

	return nil
}

// parseConfigsWithWeights parses config flags, expands globs, and extracts weights.
func parseConfigsWithWeights(configs []string) ([]reportWithWeight, error) {
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
		matches, err := filepath.Glob(path)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", path, err)
		}

		// If no glob matches, treat as literal path
		if len(matches) == 0 {
			matches = []string{path}
		}

		for _, match := range matches {
			report, err := confidence.ParseFile(match)
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

func generateAggregateBadge(path string, report *confidence.Report, dark, verbose bool) error {
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

func generateMultiDashboard(path string, reports []reportWithWeight, aggregate *confidence.Report, dark, verbose bool) error {
	f, err := os.Create(path)
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
