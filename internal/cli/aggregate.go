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
	aggIcon      string
	aggLabel     string
	aggEmitJSON  string
	aggFragment  bool
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
	aggregateCmd.Flags().StringVar(&aggIcon, "icon", "", "SVG path data for flat badge icon")
	aggregateCmd.Flags().StringVar(&aggLabel, "label", "", "custom label for flat badge (defaults to 'Aggregate')")
	aggregateCmd.Flags().StringVar(&aggEmitJSON, "emit-json", "", "write aggregate report JSON to file")
	aggregateCmd.Flags().BoolVar(&aggFragment, "fragment", false, "output HTML fragment without DOCTYPE/html wrapper (for embedding)")

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
	Icon      string
	Label     string
	EmitJSON  string
	Fragment  bool
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
		Icon:      aggIcon,
		Label:     aggLabel,
		EmitJSON:  aggEmitJSON,
		Fragment:  aggFragment,
	})
}

func aggregateImpl(deps *AggregateDeps) error {
	reports, err := parseConfigsWithWeightsFS(deps.FS, deps.Configs)
	if err != nil {
		return err
	}

	if len(reports) == 0 {
		return fmt.Errorf("no reports found")
	}

	aggregateReport := computeAggregateReport(reports)
	showVerbose := deps.Verbose && !deps.Quiet

	if showVerbose {
		printAggregateVerbose(reports, aggregateReport.ScoreValue())
	}

	if err := writeAggregateOutputs(deps, reports, aggregateReport, showVerbose); err != nil {
		return err
	}

	if deps.FailUnder > 0 && aggregateReport.ScoreValue() < deps.FailUnder {
		if !deps.Quiet {
			_, _ = fmt.Fprintf(deps.Stderr, "Aggregate score %d is below threshold %d\n", aggregateReport.ScoreValue(), deps.FailUnder)
		}
		deps.ExitFunc(1)
	}

	return nil
}

// computeAggregateReport calculates the weighted average score and determines the threshold.
func computeAggregateReport(reports []reportWithWeight) *confidence.Report {
	var totalWeight int
	var weightedSum int
	for _, r := range reports {
		totalWeight += r.Weight
		weightedSum += r.Report.ScoreValue() * r.Weight
	}

	aggregateScore := 0
	if totalWeight > 0 {
		aggregateScore = (weightedSum + totalWeight/2) / totalWeight
	}

	report := &confidence.Report{
		Title:     "Aggregate",
		Score:     &aggregateScore,
		Threshold: 75,
	}

	for _, r := range reports {
		if r.Report.Threshold > 0 && (report.Threshold == 75 || r.Report.Threshold < report.Threshold) {
			report.Threshold = r.Report.Threshold
		}
	}

	return report
}

func printAggregateVerbose(reports []reportWithWeight, aggregateScore int) {
	fmt.Printf("Aggregating %d reports (aggregate score: %d)\n", len(reports), aggregateScore)
	for _, r := range reports {
		fmt.Printf("  - %s: %d (weight: %d)\n", r.Report.Title, r.Report.ScoreValue(), r.Weight)
	}
}

// writeAggregateOutputs generates all output files: JSON, badges, and dashboard.
func writeAggregateOutputs(deps *AggregateDeps, reports []reportWithWeight, aggregateReport *confidence.Report, verbose bool) error {
	dashboardDir := filepath.Join(deps.Output, "dashboard")
	if err := deps.FS.MkdirAll(dashboardDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	if deps.EmitJSON != "" {
		if err := writeAggregateJSON(deps.FS, deps.EmitJSON, aggregateReport, verbose); err != nil {
			return err
		}
	}

	badgePath := filepath.Join(deps.Output, "badge.svg")
	if err := generateAggregateBadgeWithFS(deps.FS, badgePath, aggregateReport, deps.Dark, deps.BadgeType, deps.Icon, deps.Label, verbose); err != nil {
		return err
	}

	dashboardPath := filepath.Join(dashboardDir, "index.html")
	if err := generateMultiDashboardWithFS(deps.FS, dashboardPath, reports, aggregateReport, deps.Dark, deps.Fragment, verbose); err != nil {
		return err
	}

	return generateIndividualBadges(deps.FS, deps.Output, reports, deps.Dark, deps.BadgeType, verbose)
}

func generateIndividualBadges(fs FileSystem, outputDir string, reports []reportWithWeight, dark bool, badgeType string, verbose bool) error {
	for i, r := range reports {
		safeName := sanitizeFilename(r.Report.Title)
		if safeName == "" {
			safeName = fmt.Sprintf("report_%d", i)
		}
		path := filepath.Join(outputDir, fmt.Sprintf("%s.svg", safeName))
		if err := generateAggregateBadgeWithFS(fs, path, r.Report, dark, badgeType, "", "", verbose); err != nil {
			return err
		}
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
			suffix := cfg[idx+1:]
			if suffix != "" {
				w, err := strconv.Atoi(suffix)
				if err != nil {
					return nil, fmt.Errorf("invalid weight %q in config %q: must be an integer", suffix, cfg)
				}
				path = cfg[:idx]
				weight = w
			}
		}

		// Expand glob pattern
		matches, err := fs.Glob(path)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", path, err)
		}

		// If no glob matches, treat as literal path — but error if the path
		// contains glob metacharacters, since that means the pattern matched nothing.
		if len(matches) == 0 {
			if strings.ContainsAny(path, "*?[") {
				return nil, fmt.Errorf("no files matched glob pattern %q", path)
			}
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

func generateAggregateBadge(path string, report *confidence.Report, dark bool, badgeType, icon, label string, verbose bool) error {
	return generateAggregateBadgeWithFS(DefaultFileSystem, path, report, dark, badgeType, icon, label, verbose)
}

func generateAggregateBadgeWithFS(fs FileSystem, path string, report *confidence.Report, dark bool, badgeType, icon, label string, verbose bool) error {
	return writeToFileWithFS(fs, path, verbose, "badge", func(w io.Writer) error {
		if badgeType == "flat" {
			return gauge.GenerateFlat(w, report, gauge.FlatOptions{
				ColorOptions: gauge.ColorOptions{DarkMode: dark},
				Icon:         icon,
				Label:        label,
			})
		}
		return gauge.Generate(w, report, gauge.Options{
			ColorOptions: gauge.ColorOptions{DarkMode: dark},
			Width:        200,
			Height:       120,
		})
	})
}

func generateMultiDashboard(path string, reports []reportWithWeight, aggregate *confidence.Report, dark, fragment, verbose bool) error {
	return generateMultiDashboardWithFS(DefaultFileSystem, path, reports, aggregate, dark, fragment, verbose)
}

func generateMultiDashboardWithFS(fs FileSystem, path string, reports []reportWithWeight, aggregate *confidence.Report, dark, fragment, verbose bool) error {
	// Convert to format expected by dashboard
	dashReports := make([]dashboard.ReportSummary, 0, len(reports))
	for _, r := range reports {
		dashReports = append(dashReports, dashboard.ReportSummary{
			Report: r.Report,
			Weight: r.Weight,
			Path:   r.Path,
		})
	}

	return writeToFileWithFS(fs, path, verbose, "dashboard", func(w io.Writer) error {
		return dashboard.GenerateMulti(w, dashReports, aggregate, dashboard.MultiOptions{
			DarkMode: dark, Fragment: fragment,
		})
	})
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

// writeAggregateJSON writes the aggregate report to a JSON file.
func writeAggregateJSON(fs FileSystem, path string, report *confidence.Report, verbose bool) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := fs.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating directory for emit-json: %w", err)
		}
	}

	output := struct {
		Title     string `json:"title"`
		Score     int    `json:"score"`
		Threshold int    `json:"threshold"`
		Passed    bool   `json:"passed"`
	}{
		Title:     report.Title,
		Score:     report.ScoreValue(),
		Threshold: report.Threshold,
		Passed:    report.Passed(),
	}

	return writeToFileWithFS(fs, path, verbose, "JSON", func(w io.Writer) error {
		return encodeJSONIndented(w, output)
	})
}


