package cli

import (
	"bytes"
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

const errCreatingComment = "creating comment: %w"

var (
	commentGitHubConfig     string
	commentGitHubOwner      string
	commentGitHubRepo       string
	commentGitHubPR         int
	commentGitHubToken      string
	commentGitHubAPIURL     string
	commentGitHubMode       string
	commentGitHubAutoDetect bool
	commentGitHubDryRun     bool
)

var commentGitHubCmd = &cobra.Command{
	Use:   "github",
	Short: "Post confidence report as GitHub PR comment",
	Long: `Post confidence report as a comment on a GitHub pull request.

This creates or updates a PR comment showing the confidence score,
pass/fail status, and factor breakdown. Comments are identified by
a hidden marker, allowing updates to the same comment.

In GitHub Actions, most options can be auto-detected from
environment variables.`,
	Example: `  # Auto-detect from GitHub Actions environment
  confvis comment github -c confidence.json

  # Explicit options
  confvis comment github -c confidence.json \
    --repo owner/repo --pr 123 \
    --token $GITHUB_TOKEN

  # Always create new comment
  confvis comment github -c confidence.json --mode create

  # Update existing or create new (default)
  confvis comment github -c confidence.json --mode update

  # Delete all previous confvis comments, then create new
  confvis comment github -c confidence.json --mode replace

  # Preview without posting
  confvis comment github -c confidence.json --dry-run`,
	RunE: runCommentGitHub,
}

func init() {
	commentGitHubCmd.Flags().StringVarP(&commentGitHubConfig, "config", "c", "", "path to confidence report (JSON/YAML) (required)")
	commentGitHubCmd.Flags().StringVar(&commentGitHubOwner, "owner", "", "repository owner (auto-detected in GitHub Actions)")
	commentGitHubCmd.Flags().StringVar(&commentGitHubRepo, "repo", "", "repository name (auto-detected in GitHub Actions)")
	commentGitHubCmd.Flags().IntVar(&commentGitHubPR, "pr", 0, "pull request number (auto-detected in GitHub Actions)")
	commentGitHubCmd.Flags().StringVar(&commentGitHubToken, "token", "", "GitHub token (or GITHUB_TOKEN env var)")
	commentGitHubCmd.Flags().StringVar(&commentGitHubAPIURL, "api-url", "", "GitHub API URL (auto-detected in GitHub Actions)")
	commentGitHubCmd.Flags().StringVar(&commentGitHubMode, "mode", "update", "comment mode: create, update, replace")
	commentGitHubCmd.Flags().BoolVar(&commentGitHubAutoDetect, "auto-detect", true, "auto-detect values from GitHub Actions environment")
	commentGitHubCmd.Flags().BoolVar(&commentGitHubDryRun, "dry-run", false, "preview comment without posting")

	if err := commentGitHubCmd.MarkFlagRequired("config"); err != nil {
		panic(err)
	}

	bindCommentGitHubFlags(commentGitHubCmd)

	commentCmd.AddCommand(commentGitHubCmd)
}

func bindCommentGitHubFlags(cmd *cobra.Command) {
	_ = viper.BindPFlag("comment.github.owner", cmd.Flags().Lookup("owner"))
	_ = viper.BindPFlag("comment.github.repo", cmd.Flags().Lookup("repo"))
	_ = viper.BindPFlag("comment.github.mode", cmd.Flags().Lookup("mode"))
	_ = viper.BindPFlag("comment.github.api_url", cmd.Flags().Lookup("api-url"))
}

// CommentGitHubDeps contains dependencies for the comment github command.
type CommentGitHubDeps struct {
	FS         FileSystem
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	Verbose    bool
	Quiet      bool
	Config     string
	Owner      string
	Repo       string
	PR         int
	Token      string
	APIURL     string
	Mode       string
	AutoDetect bool
	DryRun     bool
	Timeout    time.Duration

	// Functions for testability
	FindComment       func(ctx context.Context, client *checks.GitHubClient, opts checks.CommentOptions) (*checks.CommentResponse, error)
	PostComment       func(ctx context.Context, client *checks.GitHubClient, opts checks.CommentOptions, body string) (*checks.CommentResponse, error)
	UpdateComment     func(ctx context.Context, client *checks.GitHubClient, opts checks.CommentOptions, commentID int64, body string) (*checks.CommentResponse, error)
	DeleteComment     func(ctx context.Context, client *checks.GitHubClient, opts checks.CommentOptions, commentID int64) error
	FindAllComments   func(ctx context.Context, client *checks.GitHubClient, opts checks.CommentOptions) ([]checks.CommentResponse, error)
	LoadGitHubEnvPR   func() (*checks.GitHubEnv, int, error)
}

func runCommentGitHub(_ *cobra.Command, _ []string) error {
	return commentGitHubImpl(&CommentGitHubDeps{
		FS:              DefaultFileSystem,
		Stdin:           os.Stdin,
		Stdout:          os.Stdout,
		Stderr:          os.Stderr,
		Verbose:         verbose,
		Quiet:           quiet,
		Config:          commentGitHubConfig,
		Owner:           getCommentGitHubOwner(),
		Repo:            getCommentGitHubRepo(),
		PR:              commentGitHubPR,
		Token:           commentGitHubToken,
		APIURL:          getCommentGitHubAPIURL(),
		Mode:            getCommentGitHubMode(),
		AutoDetect:      commentGitHubAutoDetect,
		DryRun:          commentGitHubDryRun,
		Timeout:         30 * time.Second,
		FindComment:     defaultFindComment,
		PostComment:     defaultPostComment,
		UpdateComment:   defaultUpdateComment,
		DeleteComment:   defaultDeleteComment,
		FindAllComments: defaultFindAllComments,
		LoadGitHubEnvPR: checks.LoadGitHubEnvWithPR,
	})
}

func defaultFindComment(ctx context.Context, client *checks.GitHubClient, opts checks.CommentOptions) (*checks.CommentResponse, error) {
	return client.FindComment(ctx, opts)
}

func defaultPostComment(ctx context.Context, client *checks.GitHubClient, opts checks.CommentOptions, body string) (*checks.CommentResponse, error) {
	return client.PostComment(ctx, opts, body)
}

func defaultUpdateComment(ctx context.Context, client *checks.GitHubClient, opts checks.CommentOptions, commentID int64, body string) (*checks.CommentResponse, error) {
	return client.UpdateComment(ctx, opts, commentID, body)
}

func defaultDeleteComment(ctx context.Context, client *checks.GitHubClient, opts checks.CommentOptions, commentID int64) error {
	return client.DeleteComment(ctx, opts, commentID)
}

func defaultFindAllComments(ctx context.Context, client *checks.GitHubClient, opts checks.CommentOptions) ([]checks.CommentResponse, error) {
	return client.FindAllConfvisComments(ctx, opts)
}

func commentGitHubImpl(deps *CommentGitHubDeps) error {
	// Validate mode
	switch deps.Mode {
	case "create", "update", "replace":
		// valid
	default:
		return fmt.Errorf("invalid mode %q: must be create, update, or replace", deps.Mode)
	}

	// Resolve options from environment
	opts, client, err := resolveCommentOptions(deps)
	if err != nil {
		return err
	}

	// Read the confidence report
	report, err := parseCommentConfig(deps)
	if err != nil {
		return err
	}

	// Generate comment body
	commentBody := generateCommentBody(report)

	// Dry-run mode: show what would happen without posting
	if deps.DryRun {
		outputCommentDryRun(deps, opts, report, commentBody)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), deps.Timeout)
	defer cancel()

	// Execute based on mode
	resp, err := executeCommentMode(ctx, deps, client, opts, commentBody)
	if err != nil {
		return err
	}

	if !deps.Quiet && resp != nil {
		status := "passed"
		if !report.Passed() {
			status = "failed"
		}
		_, _ = fmt.Fprintf(deps.Stdout, "Posted comment: %s (%s)\n", resp.HTMLURL, status)
	}

	return nil
}

func resolveCommentOptions(deps *CommentGitHubDeps) (checks.CommentOptions, *checks.GitHubClient, error) {
	opts := checks.CommentOptions{
		Owner: deps.Owner,
		Repo:  deps.Repo,
		PR:    deps.PR,
	}

	token := deps.Token
	apiURL := deps.APIURL

	// Auto-detect from GitHub Actions environment
	if deps.AutoDetect {
		var err error
		opts, token, apiURL, err = applyCommentGitHubEnv(deps, opts, token, apiURL)
		if err != nil {
			return checks.CommentOptions{}, nil, err
		}
	}

	// Validate required fields
	if token == "" {
		return checks.CommentOptions{}, nil, fmt.Errorf("GitHub token required: use --token or GITHUB_TOKEN env var")
	}
	if opts.Owner == "" {
		return checks.CommentOptions{}, nil, fmt.Errorf("owner required: use --owner or GITHUB_REPOSITORY env var")
	}
	if opts.Repo == "" {
		return checks.CommentOptions{}, nil, fmt.Errorf("repo required: use --repo or GITHUB_REPOSITORY env var")
	}
	if opts.PR == 0 {
		return checks.CommentOptions{}, nil, fmt.Errorf("PR number required: use --pr or run in a pull_request event")
	}

	client := checks.NewGitHubClient(checks.GitHubClientConfig{
		BaseURL: apiURL,
		Token:   token,
		Timeout: deps.Timeout,
	})

	return opts, client, nil
}

func applyCommentGitHubEnv(deps *CommentGitHubDeps, opts checks.CommentOptions, token, apiURL string) (checks.CommentOptions, string, string, error) {
	env, prNumber, err := deps.LoadGitHubEnvPR()
	if err != nil && token == "" {
		return opts, token, apiURL, fmt.Errorf("loading GitHub environment: %w", err)
	}

	if env == nil {
		return opts, token, apiURL, nil
	}

	if token == "" {
		token = env.Token
	}
	if apiURL == "" && env.APIURL != "" {
		apiURL = env.APIURL
	}
	if opts.PR == 0 && prNumber > 0 {
		opts.PR = prNumber
	}

	opts = applyCommentRepoFromEnv(opts, env)

	return opts, token, apiURL, nil
}

func applyCommentRepoFromEnv(opts checks.CommentOptions, env *checks.GitHubEnv) checks.CommentOptions {
	opts.Owner, opts.Repo = ParseRepoFromEnv(opts.Owner, opts.Repo, env)
	return opts
}

func parseCommentConfig(deps *CommentGitHubDeps) (*confidence.Report, error) {
	loader := &ReportLoader{
		FS:     deps.FS,
		Stdin:  deps.Stdin,
		Config: deps.Config,
	}
	return loader.LoadReport()
}

func generateCommentBody(report *confidence.Report) string {
	var buf bytes.Buffer

	// Write marker first (hidden)
	buf.WriteString(checks.CommentMarker)
	buf.WriteString("\n")

	// Use the same format as github-comment in gauge
	// Errors are ignored because bytes.Buffer.Write never fails
	_ = writeGitHubCommentHeader(&buf, report)
	_ = writeGitHubCommentFactors(&buf, report.Factors)
	_ = writeGitHubCommentFooter(&buf, report.Version)

	return buf.String()
}

func outputCommentDryRun(deps *CommentGitHubDeps, opts checks.CommentOptions, report *confidence.Report, body string) {
	status := "passed"
	if !report.Passed() {
		status = "failed"
	}

	_, _ = fmt.Fprintln(deps.Stdout, "DRY RUN: Would post comment to PR")
	_, _ = fmt.Fprintln(deps.Stdout)
	_, _ = fmt.Fprintf(deps.Stdout, "Repository: %s/%s\n", opts.Owner, opts.Repo)
	_, _ = fmt.Fprintf(deps.Stdout, "PR Number:  %d\n", opts.PR)
	_, _ = fmt.Fprintf(deps.Stdout, "Mode:       %s\n", deps.Mode)
	_, _ = fmt.Fprintf(deps.Stdout, "Status:     %s\n", status)
	_, _ = fmt.Fprintln(deps.Stdout)
	_, _ = fmt.Fprintln(deps.Stdout, "Comment content:")
	_, _ = fmt.Fprintln(deps.Stdout, "---")
	_, _ = fmt.Fprint(deps.Stdout, body)
	_, _ = fmt.Fprintln(deps.Stdout, "---")
	_, _ = fmt.Fprintln(deps.Stdout)
	_, _ = fmt.Fprintln(deps.Stdout, "No changes made.")
}

// Config getters for comment github command

func getCommentGitHubOwner() string {
	return viper.GetString("comment.github.owner")
}

func getCommentGitHubRepo() string {
	return viper.GetString("comment.github.repo")
}

func getCommentGitHubMode() string {
	if v := viper.GetString("comment.github.mode"); v != "" {
		return v
	}
	return "update"
}

func getCommentGitHubAPIURL() string {
	return viper.GetString("comment.github.api_url")
}

// executeCommentMode dispatches to the appropriate comment mode handler.
func executeCommentMode(ctx context.Context, deps *CommentGitHubDeps, client *checks.GitHubClient, opts checks.CommentOptions, body string) (*checks.CommentResponse, error) {
	switch deps.Mode {
	case "create":
		return commentModeCreate(ctx, deps, client, opts, body)
	case "update":
		return commentModeUpdate(ctx, deps, client, opts, body)
	default: // "replace" — mode is validated before reaching here
		return commentModeReplace(ctx, deps, client, opts, body)
	}
}

func commentModeCreate(ctx context.Context, deps *CommentGitHubDeps, client *checks.GitHubClient, opts checks.CommentOptions, body string) (*checks.CommentResponse, error) {
	resp, err := deps.PostComment(ctx, client, opts, body)
	if err != nil {
		return nil, fmt.Errorf(errCreatingComment, err)
	}
	return resp, nil
}

func commentModeUpdate(ctx context.Context, deps *CommentGitHubDeps, client *checks.GitHubClient, opts checks.CommentOptions, body string) (*checks.CommentResponse, error) {
	existing, err := deps.FindComment(ctx, client, opts)
	if err != nil {
		return nil, fmt.Errorf("finding existing comment: %w", err)
	}
	if existing != nil {
		resp, err := deps.UpdateComment(ctx, client, opts, existing.ID, body)
		if err != nil {
			return nil, fmt.Errorf("updating comment: %w", err)
		}
		return resp, nil
	}
	return commentModeCreate(ctx, deps, client, opts, body)
}

func commentModeReplace(ctx context.Context, deps *CommentGitHubDeps, client *checks.GitHubClient, opts checks.CommentOptions, body string) (*checks.CommentResponse, error) {
	existing, err := deps.FindAllComments(ctx, client, opts)
	if err != nil {
		return nil, fmt.Errorf("finding existing comments: %w", err)
	}
	for _, comment := range existing {
		if delErr := deps.DeleteComment(ctx, client, opts, comment.ID); delErr != nil {
			return nil, fmt.Errorf("deleting comment %d: %w", comment.ID, delErr)
		}
	}
	return commentModeCreate(ctx, deps, client, opts, body)
}
