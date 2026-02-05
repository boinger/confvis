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

// Metrics we fetch from SonarQube.
const (
	metricCoverage              = "coverage"
	metricReliabilityRating     = "reliability_rating"
	metricSecurityRating        = "security_rating"
	metricSqaleRating           = "sqale_rating" // Maintainability
	metricVulnerabilities       = "vulnerabilities"
	metricBugs                  = "bugs"
	metricCodeSmells            = "code_smells"
	metricDuplicatedLinesDensity = "duplicated_lines_density"
)

// ratingToScore converts a SonarQube rating (1.0-5.0 for A-E) to a score (0-100).
// A=100, B=75, C=50, D=25, E=0
func ratingToScore(rating float64) int {
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

// countToScore converts an issue count (vulnerabilities, bugs, code_smells) to a score.
// 0 issues = 100, then diminishing returns as count increases.
func countToScore(count int) int {
	switch {
	case count == 0:
		return 100
	case count <= 5:
		return 80
	case count <= 10:
		return 60
	case count <= 25:
		return 40
	case count <= 50:
		return 20
	default:
		return 0
	}
}

// duplicationToScore converts duplicated lines density (0-100%) to a score.
// 0% duplication = 100, 100% duplication = 0 (linear inverse).
func duplicationToScore(pct float64) int {
	score := 100 - int(pct)
	if score < 0 {
		return 0
	}
	return score
}
