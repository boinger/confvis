package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/boinger/confvis/internal/sources"
	// Import sources to register them
	_ "github.com/boinger/confvis/internal/sources/sonarqube"
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
)

var fetchCmd = &cobra.Command{
	Use:   "fetch <source>",
	Short: "Fetch metrics from an external source",
	Long: `Fetch metrics from an external source and output a confidence report.

Available sources:
  sonarqube    Fetch code quality metrics from SonarQube

Examples:
  # Fetch from SonarQube and save to file
  confvis fetch sonarqube --url https://sonar.example.com --project myapp -o confidence.json

  # Fetch and pipe directly to gauge
  confvis fetch sonarqube -p myproject -o - | confvis gauge -c - -o badge.svg

  # Use environment variables for authentication
  export SONARQUBE_URL=https://sonar.example.com
  export SONARQUBE_TOKEN=squ_xxx
  confvis fetch sonarqube -p myproject -o confidence.json`,
	Args: cobra.ExactArgs(1),
	RunE: runFetch,
}

func init() {
	fetchCmd.Flags().StringVarP(&fetchURL, "url", "u", "", "source server URL (or use environment variable)")
	fetchCmd.Flags().StringVarP(&fetchProject, "project", "p", "", "project key/identifier (required)")
	fetchCmd.Flags().StringVarP(&fetchToken, "token", "t", "", "API token (or use environment variable)")
	fetchCmd.Flags().StringVarP(&fetchBranch, "branch", "b", "", "branch to query")
	fetchCmd.Flags().StringVar(&fetchTitle, "title", "", "report title (defaults to project name)")
	fetchCmd.Flags().IntVar(&fetchThreshold, "threshold", 75, "pass/fail threshold")
	fetchCmd.Flags().IntVar(&fetchTimeout, "timeout", 30, "HTTP timeout in seconds")
	fetchCmd.Flags().StringVarP(&fetchOutput, "output", "o", "", "output file path, or - for stdout (required)")

	if err := fetchCmd.MarkFlagRequired("project"); err != nil {
		panic(err)
	}
	if err := fetchCmd.MarkFlagRequired("output"); err != nil {
		panic(err)
	}

	rootCmd.AddCommand(fetchCmd)
}

func runFetch(_ *cobra.Command, args []string) error {
	sourceName := args[0]

	source := sources.Get(sourceName)
	if source == nil {
		available := sources.Names()
		sort.Strings(available)
		return fmt.Errorf("unknown source %q, available sources: %s", sourceName, strings.Join(available, ", "))
	}

	// Build options, allowing environment variable fallbacks
	opts := sources.Options{
		URL:       fetchURL,
		Project:   fetchProject,
		Token:     fetchToken,
		Branch:    fetchBranch,
		Title:     fetchTitle,
		Threshold: fetchThreshold,
		Timeout:   fetchTimeout,
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(fetchTimeout)*time.Second)
	defer cancel()

	// Suppress verbose output when writing to stdout
	outputToStdout := fetchOutput == "-"
	showVerbose := verbose && !quiet && !outputToStdout

	if showVerbose {
		fmt.Printf("Fetching metrics from %s for project %q\n", sourceName, fetchProject)
	}

	report, err := source.Fetch(ctx, opts)
	if err != nil {
		return fmt.Errorf("fetching from %s: %w", sourceName, err)
	}

	// Write output
	var out *os.File
	if outputToStdout {
		out = os.Stdout
	} else {
		out, err = os.Create(fetchOutput)
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
		fmt.Printf("Score: %d/%d (%s)\n", report.Score, report.Threshold, status)
		if !outputToStdout {
			fmt.Printf("Wrote report to %s\n", fetchOutput)
		}
	}

	return nil
}
