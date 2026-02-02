package snyk

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
)

const sourceName = "snyk"

// Environment variable names for configuration.
const (
	EnvToken  = "SNYK_TOKEN"
	EnvOrgID  = "SNYK_ORG_ID"
	EnvAPIURL = "SNYK_API_URL"
)

// Severity penalties (points deducted per issue).
const (
	PenaltyCritical = 33
	PenaltyHigh     = 20
	PenaltyMedium   = 10
	PenaltyLow      = 5
)

// Factor weights.
const (
	WeightCritical = 40
	WeightHigh     = 30
	WeightMedium   = 20
	WeightLow      = 10
)

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
	// Resolve token from option or environment
	token := opts.Token
	if token == "" {
		token = os.Getenv(EnvToken)
	}
	if token == "" {
		return nil, fmt.Errorf("snyk token required: use --token flag or set %s", EnvToken)
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

	// Resolve API URL from option or environment
	apiURL := opts.URL
	if apiURL == "" {
		apiURL = os.Getenv(EnvAPIURL)
	}

	// Project ID is required
	projectID := opts.Project
	if projectID == "" {
		return nil, fmt.Errorf("snyk project ID required: use --project flag")
	}

	// Create client with timeout
	timeout := time.Duration(opts.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := NewClient(apiURL, token, timeout)

	// Fetch project with issue counts
	project, err := client.FetchProject(ctx, orgID, projectID)
	if err != nil {
		return nil, err
	}

	// Get issue counts (default to zero if not present)
	var counts IssueCounts
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
	factors := []confidence.Factor{
		{
			Name:        "Critical Vulnerabilities",
			Score:       SeverityScore(counts.Critical, PenaltyCritical),
			Weight:      WeightCritical,
			Description: fmt.Sprintf("%d critical", counts.Critical),
			URL:         client.ProjectURL(orgID, projectID),
		},
		{
			Name:        "High Vulnerabilities",
			Score:       SeverityScore(counts.High, PenaltyHigh),
			Weight:      WeightHigh,
			Description: fmt.Sprintf("%d high", counts.High),
			URL:         client.ProjectURL(orgID, projectID),
		},
		{
			Name:        "Medium Vulnerabilities",
			Score:       SeverityScore(counts.Medium, PenaltyMedium),
			Weight:      WeightMedium,
			Description: fmt.Sprintf("%d medium", counts.Medium),
			URL:         client.ProjectURL(orgID, projectID),
		},
		{
			Name:        "Low Vulnerabilities",
			Score:       SeverityScore(counts.Low, PenaltyLow),
			Weight:      WeightLow,
			Description: fmt.Sprintf("%d low", counts.Low),
			URL:         client.ProjectURL(orgID, projectID),
		},
	}

	// Build report
	result := &confidence.Report{
		Title:       title,
		Threshold:   opts.Threshold,
		Source:      sourceName,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Factors:     factors,
	}

	// Calculate weighted score
	result.Score = result.CalculateScore()

	return result, nil
}
