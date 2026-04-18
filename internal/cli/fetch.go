package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
	// Import sources to register them
	_ "github.com/boinger/confvis/internal/sources/codecov"
	_ "github.com/boinger/confvis/internal/sources/codeql"
	_ "github.com/boinger/confvis/internal/sources/coveralls"
	_ "github.com/boinger/confvis/internal/sources/dependabot"
	_ "github.com/boinger/confvis/internal/sources/ghactions"
	_ "github.com/boinger/confvis/internal/sources/gitleaks"
	_ "github.com/boinger/confvis/internal/sources/gosec"
	_ "github.com/boinger/confvis/internal/sources/grype"
	_ "github.com/boinger/confvis/internal/sources/semgrep"
	_ "github.com/boinger/confvis/internal/sources/snyk"
	_ "github.com/boinger/confvis/internal/sources/sonarqube"
	_ "github.com/boinger/confvis/internal/sources/trivy"
	_ "github.com/boinger/confvis/internal/sources/trufflehog"
)

var (
	fetchURL       string
	fetchProject   string
	fetchToken     string
	fetchBranch    string
	fetchTitle     string
	fetchThreshold int
	fetchTimeout   int
	fetchOutput    string
	// Source-specific flags
	fetchService     string // codecov: git provider
	fetchWorkflow    string // github-actions: workflow filter
	fetchEvent       string // github-actions: event filter
	fetchCount       int    // github-actions: run count
	fetchGrypeCmd    string // grype: command to run
	fetchOrg         string // snyk: organization ID
	fetchSemgrepCmd  string // semgrep: command to run
	fetchSemgrepConf string // semgrep: config/rules
	fetchFromStdin   bool   // semgrep: read from stdin
	fetchTrivyCmd    string // trivy: command to run
	fetchToolName    string // codeql: tool name filter
	fetchGitleaksCmd   string // gitleaks: command to run
	fetchTrufflehogCmd string // trufflehog: command to run
	fetchScanMode      string // trufflehog: scan mode (filesystem or git)
	fetchGosecCmd      string // gosec: command to run
)

var fetchCmd = &cobra.Command{
	Use:   "fetch <source>",
	Short: "Fetch metrics from an external source",
	Long: `Fetch metrics from an external source and output a confidence report.

Available sources:
  sonarqube      Code quality metrics from SonarQube
  codecov        Coverage metrics from Codecov
  codeql         Code scanning alerts from GitHub CodeQL
  coveralls      Coverage metrics from Coveralls
  dependabot     Vulnerability alerts from GitHub Dependabot
  github-actions CI/CD workflow metrics from GitHub Actions
  gitleaks       Secret detection with GitLeaks
  gosec          Go security analysis with Gosec
  grype          Security vulnerability scanning with Grype
  semgrep        Static analysis findings from Semgrep
  snyk           Vulnerability metrics from Snyk
  trivy          Security vulnerability scanning with Trivy
  trufflehog     Secret detection with TruffleHog

Examples:
  # Fetch from SonarQube
  confvis fetch sonarqube --url https://sonar.example.com --project myapp -o confidence.json

  # Fetch from Codecov (project is owner/repo)
  export CODECOV_TOKEN=xxx
  confvis fetch codecov -p myorg/myrepo -o confidence.json

  # Fetch from CodeQL (GitHub code scanning)
  export GITHUB_TOKEN=xxx
  confvis fetch codeql -p owner/repo -o codeql.json
  confvis fetch codeql -p owner/repo --tool CodeQL -o codeql.json

  # Fetch from Coveralls (coverage metrics)
  confvis fetch coveralls -p myorg/myrepo -o confidence.json
  export COVERALLS_TOKEN=xxx  # optional, for private repos
  confvis fetch coveralls -p myorg/myrepo --service gitlab -o confidence.json

  # Fetch from Dependabot (GitHub vulnerability alerts)
  export GITHUB_TOKEN=xxx
  confvis fetch dependabot -p owner/repo -o dependabot.json

  # Fetch from GitHub Actions
  export GITHUB_TOKEN=xxx
  confvis fetch github-actions -p myorg/myrepo --workflow ci.yml --count 20 -o confidence.json

  # Fetch from GitLeaks (secret detection)
  confvis fetch gitleaks -p . -o gitleaks.json
  confvis fetch gitleaks -p /path/to/repo --gitleaks-cmd "docker run zricethezav/gitleaks" -o gitleaks.json

  # Fetch from Gosec (Go security analysis)
  confvis fetch gosec -p ./... -o gosec.json
  confvis fetch gosec -p ./cmd/... -o gosec.json

  # Fetch from Grype (container/filesystem scan)
  confvis fetch grype -p . -o grype.json
  confvis fetch grype -p alpine:latest -o grype.json

  # Fetch from Semgrep (static analysis)
  confvis fetch semgrep -p . -o semgrep.json
  semgrep --json . | confvis fetch semgrep --from-stdin -o semgrep.json

  # Fetch from Snyk
  export SNYK_TOKEN=xxx
  confvis fetch snyk --org my-org-id -p my-project-id -o confidence.json

  # Fetch from Trivy (local security scan)
  confvis fetch trivy -p . -o security.json
  confvis fetch trivy -p . --trivy-cmd "docker run aquasec/trivy" -o security.json

  # Fetch from TruffleHog (secret detection with verification)
  confvis fetch trufflehog -p . -o trufflehog.json
  confvis fetch trufflehog -p https://github.com/org/repo.git --scan-mode git -o trufflehog.json

  # Fetch and pipe directly to gauge
  confvis fetch sonarqube -p myproject -o - | confvis gauge -c - -o badge.svg`,
	Args: cobra.ExactArgs(1),
	RunE: runFetch,
}

func init() {
	// Common flags
	// Note: defaults set to 0/"" here; actual defaults come from config.go getters
	fetchCmd.Flags().StringVarP(&fetchURL, "url", "u", "", "source server URL (or use environment variable)")
	fetchCmd.Flags().StringVarP(&fetchProject, "project", "p", "", "project key/identifier (required)")
	fetchCmd.Flags().StringVarP(&fetchToken, "token", "t", "", "API token (or use environment variable)")
	fetchCmd.Flags().StringVarP(&fetchBranch, "branch", "b", "", "branch to query")
	fetchCmd.Flags().StringVar(&fetchTitle, "title", "", "report title (defaults to project name)")
	fetchCmd.Flags().IntVar(&fetchThreshold, "threshold", 0, "pass/fail threshold")
	fetchCmd.Flags().IntVar(&fetchTimeout, "timeout", 0, "HTTP timeout in seconds")
	fetchCmd.Flags().StringVarP(&fetchOutput, "output", "o", "", "output file path, or - for stdout (required)")

	// Source-specific flags
	fetchCmd.Flags().StringVar(&fetchService, "service", "", "codecov: git provider (github, gitlab, bitbucket)")
	fetchCmd.Flags().StringVar(&fetchWorkflow, "workflow", "", "github-actions: workflow file or ID to filter")
	fetchCmd.Flags().StringVar(&fetchEvent, "event", "", "github-actions: trigger event to filter (push, pull_request)")
	fetchCmd.Flags().IntVar(&fetchCount, "count", 20, "github-actions: number of recent runs to analyze")
	fetchCmd.Flags().StringVar(&fetchGrypeCmd, "grype-cmd", "", "grype: command to run (default: grype)")
	fetchCmd.Flags().StringVar(&fetchOrg, "org", "", "snyk: organization ID")
	fetchCmd.Flags().StringVar(&fetchSemgrepCmd, "semgrep-cmd", "", "semgrep: command to run (default: semgrep)")
	fetchCmd.Flags().StringVar(&fetchSemgrepConf, "semgrep-config", "", "semgrep: config/rules to use (default: auto)")
	fetchCmd.Flags().BoolVar(&fetchFromStdin, "from-stdin", false, "semgrep: read JSON output from stdin")
	fetchCmd.Flags().StringVar(&fetchTrivyCmd, "trivy-cmd", "", "trivy: command to run (default: trivy)")
	fetchCmd.Flags().StringVar(&fetchToolName, "tool", "", "codeql: filter alerts by tool name")
	fetchCmd.Flags().StringVar(&fetchGitleaksCmd, "gitleaks-cmd", "", "gitleaks: command to run (default: gitleaks)")
	fetchCmd.Flags().StringVar(&fetchTrufflehogCmd, "trufflehog-cmd", "", "trufflehog: command to run (default: trufflehog)")
	fetchCmd.Flags().StringVar(&fetchScanMode, "scan-mode", "", "trufflehog: scan mode (filesystem or git)")
	fetchCmd.Flags().StringVar(&fetchGosecCmd, "gosec-cmd", "", "gosec: command to run (default: gosec)")

	// Bind flags to viper for config file support
	bindFetchFlags(fetchCmd)

	if err := fetchCmd.MarkFlagRequired("project"); err != nil {
		panic(err)
	}
	if err := fetchCmd.MarkFlagRequired("output"); err != nil {
		panic(err)
	}

	rootCmd.AddCommand(fetchCmd)
}

// FetchDeps contains dependencies for the fetch command.
type FetchDeps struct {
	FS           FileSystem
	Stdout       io.Writer
	Stderr       io.Writer
	Verbose      bool
	Quiet        bool
	SourceGetter func(string) sources.Source
	SourceName   string
	URL          string
	Project      string
	Token        string
	Branch       string
	Title        string
	Threshold    int
	Timeout      int
	Output       string
	Extra        map[string]string
}

func runFetch(_ *cobra.Command, args []string) error {
	sourceName := args[0]

	// Get config values with proper precedence (config < env < flag)
	timeout := getFetchTimeout()
	threshold := getFetchThreshold()

	// Get source-specific URL from config if not provided via flag
	url := fetchURL
	if url == "" {
		url = getSourceURL(sourceName)
	}

	// Get source-specific org from config if not provided via flag
	org := fetchOrg
	if org == "" {
		org = getSourceOrg(sourceName)
	}

	// Get source-specific service from config if not provided via flag
	service := fetchService
	if service == "" {
		service = getSourceService(sourceName)
	}
	if service == "" {
		service = "github" // default for codecov
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	return fetchImpl(ctx, &FetchDeps{
		FS:           DefaultFileSystem,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
		Verbose:      verbose,
		Quiet:        quiet,
		SourceGetter: sources.Get,
		SourceName:   sourceName,
		URL:          url,
		Project:      fetchProject,
		Token:        fetchToken,
		Branch:       fetchBranch,
		Title:        fetchTitle,
		Threshold:    threshold,
		Timeout:      timeout,
		Output:       fetchOutput,
		Extra: map[string]string{
			"service":     service,
			"workflow":    fetchWorkflow,
			"event":       fetchEvent,
			"count":       strconv.Itoa(fetchCount),
			"grype-cmd":   fetchGrypeCmd,
			"org":         org,
			"semgrep-cmd": fetchSemgrepCmd,
			"config":      fetchSemgrepConf,
			"from-stdin":  strconv.FormatBool(fetchFromStdin),
			"trivy-cmd":   fetchTrivyCmd,
			"tool":         fetchToolName,
			"gitleaks-cmd":   fetchGitleaksCmd,
			"trufflehog-cmd": fetchTrufflehogCmd,
			"mode":           fetchScanMode,
			"gosec-cmd":     fetchGosecCmd,
		},
	})
}

func fetchImpl(ctx context.Context, deps *FetchDeps) error {
	source := deps.SourceGetter(deps.SourceName)
	if source == nil {
		available := sources.Names()
		sort.Strings(available)
		return fmt.Errorf("unknown source %q, available sources: %s", deps.SourceName, strings.Join(available, ", "))
	}

	opts := sources.Options{
		URL:       deps.URL,
		Project:   deps.Project,
		Token:     deps.Token,
		Branch:    deps.Branch,
		Title:     deps.Title,
		Threshold: deps.Threshold,
		Timeout:   deps.Timeout,
		Extra:     deps.Extra,
	}

	outputToStdout := deps.Output == "-"
	showVerbose := deps.Verbose && !deps.Quiet && !outputToStdout

	if showVerbose {
		_, _ = fmt.Fprintf(deps.Stderr, "Fetching metrics from %s for project %q\n", deps.SourceName, deps.Project)
	}

	report, err := source.Fetch(ctx, opts)
	if err != nil {
		return fmt.Errorf("fetching from %s: %w", deps.SourceName, err)
	}

	if err := writeFetchOutput(deps, report, outputToStdout); err != nil {
		return err
	}

	if showVerbose {
		printFetchVerbose(deps, report, outputToStdout)
	}

	return nil
}

// writeFetchOutput writes the report as JSON to the configured output destination.
func writeFetchOutput(deps *FetchDeps, report *confidence.Report, outputToStdout bool) (err error) {
	var out io.WriteCloser
	if outputToStdout {
		out = nopWriteCloser{deps.Stdout}
	} else {
		out, err = deps.FS.Create(deps.Output)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer func() {
			if cerr := out.Close(); cerr != nil && err == nil {
				err = fmt.Errorf("closing output file: %w", cerr)
			}
		}()
	}

	if err := encodeJSONIndented(out, report); err != nil {
		return err
	}

	return nil
}

func printFetchVerbose(deps *FetchDeps, report *confidence.Report, outputToStdout bool) {
	status := "PASS"
	if !report.IsPass() {
		status = "FAIL"
	}
	_, _ = fmt.Fprintf(deps.Stderr, "Score: %d/%d (%s)\n", report.ScoreValue(), report.Threshold, status)
	if !outputToStdout {
		_, _ = fmt.Fprintf(deps.Stderr, "Wrote report to %s\n", deps.Output)
	}
}

// nopWriteCloser wraps an io.Writer and provides a no-op Close method.
type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }
