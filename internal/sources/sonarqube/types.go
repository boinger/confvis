// Package sonarqube provides a source for fetching metrics from SonarQube.
package sonarqube

// MeasuresResponse represents the response from /api/measures/component.
type MeasuresResponse struct {
	Component ComponentMeasures `json:"component"`
}

// ComponentMeasures holds the measures for a component.
type ComponentMeasures struct {
	Key      string    `json:"key"`
	Name     string    `json:"name"`
	Measures []Measure `json:"measures"`
}

// Measure represents a single metric measurement.
type Measure struct {
	Metric string `json:"metric"`
	Value  string `json:"value"`
}

// QualityGateResponse represents the response from /api/qualitygates/project_status.
type QualityGateResponse struct {
	ProjectStatus ProjectStatus `json:"projectStatus"`
}

// ProjectStatus holds the quality gate status.
type ProjectStatus struct {
	Status     string      `json:"status"`
	Conditions []Condition `json:"conditions"`
}

// Condition represents a quality gate condition.
type Condition struct {
	Status         string `json:"status"`
	MetricKey      string `json:"metricKey"`
	Comparator     string `json:"comparator"`
	ErrorThreshold string `json:"errorThreshold"`
	ActualValue    string `json:"actualValue"`
}

// Metrics we fetch from SonarQube.
const (
	MetricCoverage         = "coverage"
	MetricReliabilityRating = "reliability_rating"
	MetricSecurityRating   = "security_rating"
	MetricSqaleRating      = "sqale_rating" // Maintainability
)

// AllMetrics lists all metrics to fetch from SonarQube.
var AllMetrics = []string{
	MetricCoverage,
	MetricReliabilityRating,
	MetricSecurityRating,
	MetricSqaleRating,
}

// RatingToScore converts a SonarQube rating (1.0-5.0 for A-E) to a score (0-100).
// A=100, B=75, C=50, D=25, E=0
func RatingToScore(rating float64) int {
	// SonarQube ratings: 1.0=A, 2.0=B, 3.0=C, 4.0=D, 5.0=E
	switch {
	case rating <= 1.0:
		return 100
	case rating <= 2.0:
		return 75
	case rating <= 3.0:
		return 50
	case rating <= 4.0:
		return 25
	default:
		return 0
	}
}
