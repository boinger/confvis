package sonarqube

import (
	"context"
	"strconv"
	"time"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/scoring"
)

const sourceName = "sonarqube"

// Environment variable names for configuration.
const (
	EnvURL   = "SONARQUBE_URL"
	EnvToken = "SONARQUBE_TOKEN"
)

var configResolver = &sources.ConfigResolver{
	SourceName:     "SonarQube",
	TokenEnvVar:    EnvToken,
	URLEnvVar:      EnvURL,
	TokenRequired:  false, // Optional for public projects
	URLRequired:    true,
	DefaultTimeout: 30 * time.Second,
}

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
	cfg, err := configResolver.Resolve(opts)
	if err != nil {
		return nil, err
	}

	client := NewClient(cfg.URL, cfg.Token, cfg.Timeout)

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

	return scoring.BuildReport(title, sourceName, opts.Threshold, factors), nil
}

// measuresToFactors converts SonarQube measures to confidence factors.
func (s *Source) measuresToFactors(measures *MeasuresResponse, client *Client, project, branch string) []confidence.Factor {
	// Create a map for quick lookup
	measureMap := make(map[string]string)
	for _, m := range measures.Component.Measures {
		measureMap[m.Metric] = m.Value
	}

	var factors []confidence.Factor

	// High priority metrics (weight 20 each)

	// Test Coverage (direct percentage)
	if val, ok := measureMap[MetricCoverage]; ok {
		coverage, err := strconv.ParseFloat(val, 64)
		if err == nil {
			factors = append(factors, confidence.Factor{
				Name:   "Test Coverage",
				Score:  int(coverage),
				Weight: 20,
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
				Weight: 20,
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
				Weight: 20,
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
				Weight: 20,
				URL:    client.MeasureURL(project, MetricSqaleRating, branch),
			})
		}
	}

	// Medium priority metrics (weight 10 each)

	// Vulnerabilities (count converted to score)
	if val, ok := measureMap[MetricVulnerabilities]; ok {
		count, err := strconv.Atoi(val)
		if err == nil {
			factors = append(factors, confidence.Factor{
				Name:   "Vulnerabilities",
				Score:  CountToScore(count),
				Weight: 10,
				URL:    client.MeasureURL(project, MetricVulnerabilities, branch),
			})
		}
	}

	// Bugs (count converted to score)
	if val, ok := measureMap[MetricBugs]; ok {
		count, err := strconv.Atoi(val)
		if err == nil {
			factors = append(factors, confidence.Factor{
				Name:   "Bugs",
				Score:  CountToScore(count),
				Weight: 10,
				URL:    client.MeasureURL(project, MetricBugs, branch),
			})
		}
	}

	// Low priority metrics (weight 5 each)

	// Code Smells (count converted to score)
	if val, ok := measureMap[MetricCodeSmells]; ok {
		count, err := strconv.Atoi(val)
		if err == nil {
			factors = append(factors, confidence.Factor{
				Name:   "Code Smells",
				Score:  CountToScore(count),
				Weight: 5,
				URL:    client.MeasureURL(project, MetricCodeSmells, branch),
			})
		}
	}

	// Duplicated Lines Density (percentage inverted to score)
	if val, ok := measureMap[MetricDuplicatedLinesDensity]; ok {
		pct, err := strconv.ParseFloat(val, 64)
		if err == nil {
			factors = append(factors, confidence.Factor{
				Name:   "Duplication",
				Score:  DuplicationToScore(pct),
				Weight: 5,
				URL:    client.MeasureURL(project, MetricDuplicatedLinesDensity, branch),
			})
		}
	}

	return factors
}
