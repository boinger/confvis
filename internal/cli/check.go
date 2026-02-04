package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/boinger/confvis/internal/checks"
	"github.com/boinger/confvis/internal/confidence"
)

var (
	checkConfig     string
	checkOwner      string
	checkRepo       string
	checkSHA        string
	checkName       string
	checkToken      string
	checkAPIURL     string
	checkAutoDetect bool
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Create check runs on CI platforms",
	Long: `Create check runs on CI platforms.

Confvis can create check runs with your confidence score on
supported CI platforms, providing inline status feedback.`,
}

var checkGitHubCmd = &cobra.Command{
	Use:   "github",
	Short: "Create a GitHub Check Run",
	Long: `Create a GitHub Check Run for your confidence score.

This creates a check run on the specified commit showing the
confidence score, pass/fail status, and factor breakdown.

In GitHub Actions, most options can be auto-detected from
environment variables.`,
	Example: `  # Auto-detect from GitHub Actions environment
  confvis check github -c confidence.json

  # Explicit options
  confvis check github -c confidence.json \
    --owner myorg --repo myrepo --sha abc123 \
    --token $GITHUB_TOKEN

  # Custom check name
  confvis check github -c confidence.json --name "Code Quality"`,
	RunE: runCheckGitHub,
}

func init() {
	// GitHub check flags
	checkGitHubCmd.Flags().StringVarP(&checkConfig, "config", "c", "", "path to confidence report (JSON/YAML) (required)")
	checkGitHubCmd.Flags().StringVar(&checkOwner, "owner", "", "repository owner (auto-detected in GitHub Actions)")
	checkGitHubCmd.Flags().StringVar(&checkRepo, "repo", "", "repository name (auto-detected in GitHub Actions)")
	checkGitHubCmd.Flags().StringVar(&checkSHA, "sha", "", "commit SHA (auto-detected in GitHub Actions)")
	checkGitHubCmd.Flags().StringVar(&checkName, "name", "Confidence Score", "check run name")
	checkGitHubCmd.Flags().StringVar(&checkToken, "token", "", "GitHub token (or GITHUB_TOKEN env var)")
	checkGitHubCmd.Flags().StringVar(&checkAPIURL, "api-url", "", "GitHub API URL (auto-detected in GitHub Actions)")
	checkGitHubCmd.Flags().BoolVar(&checkAutoDetect, "auto-detect", true, "auto-detect values from GitHub Actions environment")

	if err := checkGitHubCmd.MarkFlagRequired("config"); err != nil {
		panic(err)
	}

	// Bind flags to viper for config file support
	bindCheckFlags(checkGitHubCmd)

	checkCmd.AddCommand(checkGitHubCmd)
	rootCmd.AddCommand(checkCmd)
}

// bindCheckFlags binds check command flags to viper configuration keys.
func bindCheckFlags(cmd *cobra.Command) {
	_ = viper.BindPFlag("check.github.owner", cmd.Flags().Lookup("owner"))
	_ = viper.BindPFlag("check.github.repo", cmd.Flags().Lookup("repo"))
	_ = viper.BindPFlag("check.github.name", cmd.Flags().Lookup("name"))
	_ = viper.BindPFlag("check.github.api_url", cmd.Flags().Lookup("api-url"))
}

// CheckGitHubDeps contains dependencies for the check github command.
type CheckGitHubDeps struct {
	FS         FileSystem
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	Verbose    bool
	Quiet      bool
	Config     string
	Owner      string
	Repo       string
	SHA        string
	Name       string
	Token      string
	APIURL     string
	AutoDetect bool
	Timeout    time.Duration

	// Functions for testability
	CreateCheck func(ctx context.Context, client *checks.GitHubClient, report *confidence.Report,
		opts checks.CreateCheckOptions) (*checks.CheckRunResponse, error)
	LoadGitHubEnv func() (*checks.GitHubEnv, error)
}

func runCheckGitHub(_ *cobra.Command, _ []string) error {
	return checkGitHubImpl(&CheckGitHubDeps{
		FS:            DefaultFileSystem,
		Stdin:         os.Stdin,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		Verbose:       verbose,
		Quiet:         quiet,
		Config:        checkConfig,
		Owner:         getCheckOwner(),
		Repo:          getCheckRepo(),
		SHA:           checkSHA,
		Name:          getCheckName(),
		Token:         checkToken,
		APIURL:        getCheckAPIURL(),
		AutoDetect:    checkAutoDetect,
		Timeout:       30 * time.Second,
		CreateCheck:   defaultCreateCheck,
		LoadGitHubEnv: checks.LoadGitHubEnv,
	})
}

func defaultCreateCheck(ctx context.Context, client *checks.GitHubClient, report *confidence.Report,
	opts checks.CreateCheckOptions) (*checks.CheckRunResponse, error) {
	return client.CreateCheck(ctx, report, opts)
}

func checkGitHubImpl(deps *CheckGitHubDeps) error {
	// Load GitHub Actions environment if auto-detect is enabled
	opts, client, err := resolveCheckOptions(deps)
	if err != nil {
		return err
	}

	// Read the confidence report
	report, err := parseCheckConfig(deps)
	if err != nil {
		return err
	}

	// Create the check run
	ctx, cancel := context.WithTimeout(context.Background(), deps.Timeout)
	defer cancel()

	resp, err := deps.CreateCheck(ctx, client, report, opts)
	if err != nil {
		return fmt.Errorf("creating check run: %w", err)
	}

	if !deps.Quiet {
		status := "passed"
		if !report.Passed() {
			status = "failed"
		}
		_, _ = fmt.Fprintf(deps.Stdout, "Created check run: %s (%s)\n", resp.HTMLURL, status)
	}

	return nil
}

func resolveCheckOptions(deps *CheckGitHubDeps) (checks.CreateCheckOptions, *checks.GitHubClient, error) {
	opts := checks.CreateCheckOptions{
		Owner: deps.Owner,
		Repo:  deps.Repo,
		SHA:   deps.SHA,
		Name:  deps.Name,
	}

	token := deps.Token
	apiURL := deps.APIURL

	// Auto-detect from GitHub Actions environment
	if deps.AutoDetect {
		var err error
		opts, token, apiURL, err = applyGitHubEnv(deps, opts, token, apiURL)
		if err != nil {
			return checks.CreateCheckOptions{}, nil, err
		}
	}

	// Validate we have everything we need
	if err := validateCheckOptions(opts, token); err != nil {
		return checks.CreateCheckOptions{}, nil, err
	}

	// Create the client
	client := checks.NewGitHubClient(checks.GitHubClientConfig{
		BaseURL: apiURL,
		Token:   token,
		Timeout: deps.Timeout,
	})

	return opts, client, nil
}

func applyGitHubEnv(deps *CheckGitHubDeps, opts checks.CreateCheckOptions,
	token, apiURL string) (checks.CreateCheckOptions, string, string, error) {
	env, err := deps.LoadGitHubEnv()
	if err != nil && token == "" {
		return opts, token, apiURL, fmt.Errorf("loading GitHub environment: %w", err)
	}

	if env == nil {
		return opts, token, apiURL, nil
	}

	// Fill in missing values from environment
	if token == "" {
		token = env.Token
	}
	if apiURL == "" && env.APIURL != "" {
		apiURL = env.APIURL
	}
	if opts.SHA == "" {
		opts.SHA = env.SHA
	}
	opts = applyRepoFromEnv(opts, env)

	return opts, token, apiURL, nil
}

func applyRepoFromEnv(opts checks.CreateCheckOptions, env *checks.GitHubEnv) checks.CreateCheckOptions {
	if opts.Owner != "" && opts.Repo != "" {
		return opts
	}
	if env.Repository == "" {
		return opts
	}

	owner, repo, err := checks.ParseRepository(env.Repository)
	if err != nil {
		return opts
	}

	if opts.Owner == "" {
		opts.Owner = owner
	}
	if opts.Repo == "" {
		opts.Repo = repo
	}
	return opts
}

func validateCheckOptions(opts checks.CreateCheckOptions, token string) error {
	if token == "" {
		return fmt.Errorf("GitHub token required: use --token or GITHUB_TOKEN env var")
	}
	if opts.Owner == "" {
		return fmt.Errorf("owner required: use --owner or GITHUB_REPOSITORY env var")
	}
	if opts.Repo == "" {
		return fmt.Errorf("repo required: use --repo or GITHUB_REPOSITORY env var")
	}
	if opts.SHA == "" {
		return fmt.Errorf("SHA required: use --sha or GITHUB_SHA env var")
	}
	return nil
}

func parseCheckConfig(deps *CheckGitHubDeps) (*confidence.Report, error) {
	inputFormat := confidence.FormatAuto
	if deps.Config == "-" {
		inputFormat = confidence.FormatJSON
	}

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

// Config getters for check command

func getCheckOwner() string {
	return viper.GetString("check.github.owner")
}

func getCheckRepo() string {
	return viper.GetString("check.github.repo")
}

func getCheckName() string {
	if v := viper.GetString("check.github.name"); v != "" {
		return v
	}
	return "Confidence Score"
}

func getCheckAPIURL() string {
	return viper.GetString("check.github.api_url")
}
