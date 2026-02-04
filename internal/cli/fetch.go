package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/boinger/confvis/internal/sources"
	// Import sources to register them
	_ "github.com/boinger/confvis/internal/sources/codecov"
	_ "github.com/boinger/confvis/internal/sources/dependabot"
	_ "github.com/boinger/confvis/internal/sources/ghactions"
	_ "github.com/boinger/confvis/internal/sources/snyk"
	_ "github.com/boinger/confvis/internal/sources/sonarqube"
	_ "github.com/boinger/confvis/internal/sources/trivy"
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
	fetchService  string // codecov: git provider
	fetchWorkflow string // github-actions: workflow filter
	fetchEvent    string // github-actions: event filter
	fetchCount    int    // github-actions: run count
	fetchOrg      string // snyk: organization ID
	fetchTrivyCmd string // trivy: command to run
)

var fetchCmd = &cobra.Command{
	Use:   "fetch <source>",
	Short: "Fetch metrics from an external source",
	Long: `Fetch metrics from an external source and output a confidence report.

Available sources:
  sonarqube      Code quality metrics from SonarQube
  codecov        Coverage metrics from Codecov
  dependabot     Vulnerability alerts from GitHub Dependabot
  github-actions CI/CD workflow metrics from GitHub Actions
  snyk           Vulnerability metrics from Snyk
  trivy          Security vulnerability scanning with Trivy

Examples:
  # Fetch from SonarQube
  confvis fetch sonarqube --url https://sonar.example.com --project myapp -o confidence.json

  # Fetch from Codecov (project is owner/repo)
  export CODECOV_TOKEN=xxx
  confvis fetch codecov -p myorg/myrepo -o confidence.json

  # Fetch from Dependabot (GitHub vulnerability alerts)
  export GITHUB_TOKEN=xxx
  confvis fetch dependabot -p owner/repo -o dependabot.json

  # Fetch from GitHub Actions
  export GITHUB_TOKEN=xxx
  confvis fetch github-actions -p myorg/myrepo --workflow ci.yml --count 20 -o confidence.json

  # Fetch from Snyk
  export SNYK_TOKEN=xxx
  confvis fetch snyk --org my-org-id -p my-project-id -o confidence.json

  # Fetch from Trivy (local security scan)
  confvis fetch trivy -p . -o security.json
  confvis fetch trivy -p . --trivy-cmd "docker run aquasec/trivy" -o security.json

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
	fetchCmd.Flags().StringVar(&fetchOrg, "org", "", "snyk: organization ID")
	fetchCmd.Flags().StringVar(&fetchTrivyCmd, "trivy-cmd", "", "trivy: command to run (default: trivy)")

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
			"service":   service,
			"workflow":  fetchWorkflow,
			"event":     fetchEvent,
			"count":     strconv.Itoa(fetchCount),
			"org":       org,
			"trivy-cmd": fetchTrivyCmd,
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

	// Build options, allowing environment variable fallbacks
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

	// Suppress verbose output when writing to stdout
	outputToStdout := deps.Output == "-"
	showVerbose := deps.Verbose && !deps.Quiet && !outputToStdout

	if showVerbose {
		_, _ = fmt.Fprintf(deps.Stderr, "Fetching metrics from %s for project %q\n", deps.SourceName, deps.Project)
	}

	report, err := source.Fetch(ctx, opts)
	if err != nil {
		return fmt.Errorf("fetching from %s: %w", deps.SourceName, err)
	}

	// Write output
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

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}

	if showVerbose {
		status := "PASS"
		if !report.Passed() {
			status = "FAIL"
		}
		_, _ = fmt.Fprintf(deps.Stderr, "Score: %d/%d (%s)\n", report.Score, report.Threshold, status)
		if !outputToStdout {
			_, _ = fmt.Fprintf(deps.Stderr, "Wrote report to %s\n", deps.Output)
		}
	}

	return nil
}

// nopWriteCloser wraps an io.Writer and provides a no-op Close method.
type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }
