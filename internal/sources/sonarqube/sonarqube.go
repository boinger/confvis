package sonarqube

import (
	"context"
	"fmt"
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

// metricKind indicates how to convert a SonarQube metric value to a score.
type metricKind int

const (
	metricKindPercentage  metricKind = iota // Direct float → int percentage
	metricKindRating                        // A-E rating float → score via RatingToScore
	metricKindCount                         // Integer count → score via CountToScore
	metricKindDuplication                   // Float percentage → inverted score via DuplicationToScore
)

// metricMapping defines how a SonarQube metric maps to a confidence factor.
type metricMapping struct {
	Key    string
	Name   string
	Weight int
	Kind   metricKind
}

// metricMappings defines the ordered list of SonarQube metrics to extract.
var metricMappings = []metricMapping{
	// High priority (weight 20)
	{MetricCoverage, "Test Coverage", 20, metricKindPercentage},
	{MetricReliabilityRating, "Reliability", 20, metricKindRating},
	{MetricSecurityRating, "Security", 20, metricKindRating},
	{MetricSqaleRating, "Maintainability", 20, metricKindRating},
	// Medium priority (weight 10)
	{MetricVulnerabilities, "Vulnerabilities", 10, metricKindCount},
	{MetricBugs, "Bugs", 10, metricKindCount},
	// Low priority (weight 5)
	{MetricCodeSmells, "Code Smells", 5, metricKindCount},
	{MetricDuplicatedLinesDensity, "Duplication", 5, metricKindDuplication},
}

// measuresToFactors converts SonarQube measures to confidence factors.
func (s *Source) measuresToFactors(measures *MeasuresResponse, client *Client, project, branch string) []confidence.Factor {
	measureMap := make(map[string]string)
	for _, m := range measures.Component.Measures {
		measureMap[m.Metric] = m.Value
	}

	var factors []confidence.Factor
	for _, m := range metricMappings {
		val, ok := measureMap[m.Key]
		if !ok {
			continue
		}
		score, err := convertMetricValue(val, m.Kind)
		if err != nil {
			continue
		}
		factors = append(factors, confidence.Factor{
			Name:   m.Name,
			Score:  score,
			Weight: m.Weight,
			URL:    client.MeasureURL(project, m.Key, branch),
		})
	}

	return factors
}

// convertMetricValue converts a raw SonarQube metric string to a score.
func convertMetricValue(val string, kind metricKind) (int, error) {
	switch kind {
	case metricKindPercentage:
		f, err := strconv.ParseFloat(val, 64)
		return int(f), err
	case metricKindRating:
		f, err := strconv.ParseFloat(val, 64)
		return RatingToScore(f), err
	case metricKindCount:
		n, err := strconv.Atoi(val)
		return CountToScore(n), err
	case metricKindDuplication:
		f, err := strconv.ParseFloat(val, 64)
		return DuplicationToScore(f), err
	default:
		return 0, fmt.Errorf("unknown metric kind: %d", kind)
	}
}
