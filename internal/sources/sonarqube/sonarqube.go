package sonarqube

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
)

const sourceName = "sonarqube"

// Environment variable names for configuration.
const (
	EnvURL   = "SONARQUBE_URL"
	EnvToken = "SONARQUBE_TOKEN"
)

// Source implements the sources.Source interface for SonarQube.
type Source struct{}

func init() {
	sources.Register(&Source{})
}

// Name returns the source identifier.
func (s *Source) Name() string {
	return sourceName
}

// Fetch retrieves metrics from SonarQube and converts them to a confidence report.
func (s *Source) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
	// Resolve URL from option or environment
	serverURL := opts.URL
	if serverURL == "" {
		serverURL = os.Getenv(EnvURL)
	}
	if serverURL == "" {
		return nil, fmt.Errorf("SonarQube URL required: use --url flag or set %s", EnvURL)
	}

	// Resolve token from option or environment
	token := opts.Token
	if token == "" {
		token = os.Getenv(EnvToken)
	}
	// Token is optional for public projects

	// Create client with timeout
	timeout := time.Duration(opts.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := NewClient(serverURL, token, timeout)

	// Fetch measures
	measures, err := client.FetchMeasures(ctx, opts.Project, opts.Branch)
	if err != nil {
		return nil, err
	}

	// Convert measures to factors
	factors := s.measuresToFactors(measures, client, opts.Project, opts.Branch)

	// Determine title
	title := opts.Title
	if title == "" {
		title = measures.Component.Name
	}
	if title == "" {
		title = opts.Project
	}

	// Build report
	report := &confidence.Report{
		Title:       title,
		Threshold:   opts.Threshold,
		Source:      sourceName,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Factors:     factors,
	}

	// Calculate weighted score
	report.Score = report.CalculateScore()

	return report, nil
}

// measuresToFactors converts SonarQube measures to confidence factors.
func (s *Source) measuresToFactors(measures *MeasuresResponse, client *Client, project, branch string) []confidence.Factor {
	// Create a map for quick lookup
	measureMap := make(map[string]string)
	for _, m := range measures.Component.Measures {
		measureMap[m.Metric] = m.Value
	}

	var factors []confidence.Factor

	// Test Coverage (direct percentage)
	if val, ok := measureMap[MetricCoverage]; ok {
		coverage, err := strconv.ParseFloat(val, 64)
		if err == nil {
			factors = append(factors, confidence.Factor{
				Name:   "Test Coverage",
				Score:  int(coverage),
				Weight: 25,
				URL:    client.MeasureURL(project, MetricCoverage, branch),
			})
		}
	}

	// Reliability Rating (A-E converted to score)
	if val, ok := measureMap[MetricReliabilityRating]; ok {
		rating, err := strconv.ParseFloat(val, 64)
		if err == nil {
			factors = append(factors, confidence.Factor{
				Name:   "Reliability",
				Score:  RatingToScore(rating),
				Weight: 25,
				URL:    client.MeasureURL(project, MetricReliabilityRating, branch),
			})
		}
	}

	// Security Rating (A-E converted to score)
	if val, ok := measureMap[MetricSecurityRating]; ok {
		rating, err := strconv.ParseFloat(val, 64)
		if err == nil {
			factors = append(factors, confidence.Factor{
				Name:   "Security",
				Score:  RatingToScore(rating),
				Weight: 25,
				URL:    client.MeasureURL(project, MetricSecurityRating, branch),
			})
		}
	}

	// Maintainability Rating (A-E converted to score)
	if val, ok := measureMap[MetricSqaleRating]; ok {
		rating, err := strconv.ParseFloat(val, 64)
		if err == nil {
			factors = append(factors, confidence.Factor{
				Name:   "Maintainability",
				Score:  RatingToScore(rating),
				Weight: 25,
				URL:    client.MeasureURL(project, MetricSqaleRating, branch),
			})
		}
	}

	return factors
}
