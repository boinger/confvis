// Package confidence provides types and parsing for confidence report JSON.
package confidence

// Factor represents a single confidence factor contributing to the overall score.
type Factor struct {
	Name        string `json:"name"`
	Score       int    `json:"score"`
	Weight      int    `json:"weight"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

// ColorThresholds defines score boundaries for color coding.
// GreenAbove is the minimum score for green (success) color.
// YellowAbove is the minimum score for yellow (warning) color.
// Scores below YellowAbove are shown in red (danger).
type ColorThresholds struct {
	GreenAbove  int `json:"greenAbove"`
	YellowAbove int `json:"yellowAbove"`
}

// DefaultColorThresholds returns the default color thresholds.
func DefaultColorThresholds() ColorThresholds {
	return ColorThresholds{
		GreenAbove:  75,
		YellowAbove: 50,
	}
}

// Report represents a complete confidence report with overall score and breakdown.
type Report struct {
	Title       string           `json:"title"`
	Score       int              `json:"score"`
	Threshold   int              `json:"threshold"`
	Description string           `json:"description,omitempty"`
	Factors     []Factor         `json:"factors,omitempty"`
	Thresholds  *ColorThresholds `json:"thresholds,omitempty"`

	// Metadata fields
	Version     string `json:"version,omitempty"`
	GeneratedAt string `json:"generatedAt,omitempty"`
	Source      string `json:"source,omitempty"`

	// Custom labels
	PassLabel string `json:"passLabel,omitempty"`
	FailLabel string `json:"failLabel,omitempty"`
}

// Passed returns true if the score meets or exceeds the threshold.
func (r *Report) Passed() bool {
	return r.Score >= r.Threshold
}

// EffectivePassLabel returns the custom pass label or "PASS" if not specified.
func (r *Report) EffectivePassLabel() string {
	if r.PassLabel != "" {
		return r.PassLabel
	}
	return "PASS"
}

// EffectiveFailLabel returns the custom fail label or "FAIL" if not specified.
func (r *Report) EffectiveFailLabel() string {
	if r.FailLabel != "" {
		return r.FailLabel
	}
	return "FAIL"
}

// EffectiveColorThresholds returns the report's thresholds or defaults if not specified.
func (r *Report) EffectiveColorThresholds() ColorThresholds {
	if r.Thresholds != nil {
		return *r.Thresholds
	}
	return DefaultColorThresholds()
}

// CalculateScore computes the weighted average score from factors.
// Returns 0 if no factors exist or total weight is zero.
func (r *Report) CalculateScore() int {
	if len(r.Factors) == 0 {
		return 0
	}

	var totalWeight int
	var weightedSum int

	for _, f := range r.Factors {
		totalWeight += f.Weight
		weightedSum += f.Score * f.Weight
	}

	if totalWeight == 0 {
		return 0
	}

	// Round to nearest integer
	return (weightedSum + totalWeight/2) / totalWeight
}
