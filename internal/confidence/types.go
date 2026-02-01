// Package confidence provides types and parsing for confidence report JSON.
package confidence

// Factor represents a single confidence factor contributing to the overall score.
type Factor struct {
	Name        string `json:"name"`
	Score       int    `json:"score"`
	Weight      int    `json:"weight"`
	Description string `json:"description,omitempty"`
}

// Report represents a complete confidence report with overall score and breakdown.
type Report struct {
	Title       string   `json:"title"`
	Score       int      `json:"score"`
	Threshold   int      `json:"threshold"`
	Description string   `json:"description,omitempty"`
	Factors     []Factor `json:"factors,omitempty"`
}

// Passed returns true if the score meets or exceeds the threshold.
func (r *Report) Passed() bool {
	return r.Score >= r.Threshold
}
