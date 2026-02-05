package snyk

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/scoring"
)

const sourceName = "snyk"

// Fetcher defines the interface for fetching Snyk data.
type Fetcher interface {
	FetchProject(ctx context.Context, orgID, projectID string) (*ProjectResponse, error)
	ProjectURL(orgID, projectID string) string
}

// Environment variable names for configuration.
const (
	EnvToken  = "SNYK_TOKEN"
	EnvOrgID  = "SNYK_ORG_ID"
	EnvAPIURL = "SNYK_API_URL"
)

var configResolver = &sources.ConfigResolver{
	SourceName:     sourceName,
	TokenEnvVar:    EnvToken,
	URLEnvVar:      EnvAPIURL,
	TokenRequired:  true,
	URLRequired:    false, // Has default
	DefaultTimeout: 30 * time.Second,
}

// Penalty and weight constants are defined in the scoring package.
// snyk uses the default strict penalties shared with grype and trivy.

// Source implements the sources.Source interface for Snyk.
type Source struct{}

func init() {
	sources.Register(&Source{})
}

// Name returns the source identifier.
func (s *Source) Name() string {
	return sourceName
}

// Fetch retrieves vulnerability metrics from Snyk and converts them to a confidence report.
func (s *Source) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
	cfg, err := configResolver.Resolve(opts)
	if err != nil {
		return nil, err
	}

	// Resolve org ID from Extra options or environment
	orgID := ""
	if opts.Extra != nil {
		orgID = opts.Extra["org"]
	}
	if orgID == "" {
		orgID = os.Getenv(EnvOrgID)
	}
	if orgID == "" {
		return nil, fmt.Errorf("snyk organization ID required: use --org flag or set %s", EnvOrgID)
	}

	// Project ID is required
	projectID := opts.Project
	if projectID == "" {
		return nil, fmt.Errorf("snyk project ID required: use --project flag")
	}

	client := NewClient(cfg.URL, cfg.Token, cfg.Timeout)

	return s.FetchWithClient(ctx, client, opts, orgID, projectID)
}

// FetchWithClient retrieves vulnerability metrics using the provided Fetcher.
// This allows injecting mock clients for testing.
func (s *Source) FetchWithClient(ctx context.Context, fetcher Fetcher, opts sources.Options, orgID, projectID string) (*confidence.Report, error) {
	// Fetch project with issue counts
	project, err := fetcher.FetchProject(ctx, orgID, projectID)
	if err != nil {
		return nil, err
	}

	// Get issue counts (default to zero if not present)
	var counts scoring.SeverityCounts
	if project.Data.Meta.LatestIssueCounts != nil {
		counts = *project.Data.Meta.LatestIssueCounts
	}

	// Determine title
	title := opts.Title
	if title == "" {
		title = project.Data.Attributes.Name
	}
	if title == "" {
		title = projectID
	}

	// Build factors with severity-based scoring
	factors := scoring.BuildVulnFactors(
		counts,
		scoring.DefaultPenalties(),
		scoring.DefaultWeights(),
		fetcher.ProjectURL(orgID, projectID),
	)

	return scoring.BuildReport(title, sourceName, opts.Threshold, factors), nil
}
