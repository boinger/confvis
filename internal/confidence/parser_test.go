package confidence

import (
	"strings"
	"testing"
)

func TestParse_ValidJSON(t *testing.T) {
	input := `{
		"title": "Test Report",
		"score": 85,
		"threshold": 75,
		"description": "A test report",
		"factors": [
			{"name": "Factor 1", "score": 90, "weight": 50},
			{"name": "Factor 2", "score": 80, "weight": 50}
		]
	}`

	report, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if report.Title != "Test Report" {
		t.Errorf("Title = %q, want %q", report.Title, "Test Report")
	}
	if report.Score != 85 {
		t.Errorf("Score = %d, want %d", report.Score, 85)
	}
	if report.Threshold != 75 {
		t.Errorf("Threshold = %d, want %d", report.Threshold, 75)
	}
	if len(report.Factors) != 2 {
		t.Errorf("len(Factors) = %d, want %d", len(report.Factors), 2)
	}
}

func TestParse_InvalidScore(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "score too high",
			input: `{"title": "Test", "score": 150, "threshold": 75}`,
			want:  "score must be between 0 and 100",
		},
		{
			name:  "score negative",
			input: `{"title": "Test", "score": -5, "threshold": 75}`,
			want:  "score must be between 0 and 100",
		},
		{
			name:  "threshold too high",
			input: `{"title": "Test", "score": 50, "threshold": 120}`,
			want:  "threshold must be between 0 and 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tt.input))
			if err == nil {
				t.Fatal("Parse() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.want)
			}
		})
	}
}

func TestParse_MissingFields(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "missing title",
			input: `{"score": 85, "threshold": 75}`,
			want:  "title is required",
		},
		{
			name:  "factor missing name",
			input: `{"title": "Test", "score": 85, "threshold": 75, "factors": [{"score": 50, "weight": 50}]}`,
			want:  "factor[0] name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tt.input))
			if err == nil {
				t.Fatal("Parse() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.want)
			}
		})
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	_, err := Parse(strings.NewReader("not json"))
	if err == nil {
		t.Fatal("Parse() expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "decoding JSON") {
		t.Errorf("error = %q, want to contain 'decoding JSON'", err.Error())
	}
}

func TestReport_Passed(t *testing.T) {
	tests := []struct {
		score     int
		threshold int
		want      bool
	}{
		{85, 75, true},
		{75, 75, true},
		{74, 75, false},
		{0, 0, true},
		{100, 100, true},
	}

	for _, tt := range tests {
		r := &Report{Score: tt.score, Threshold: tt.threshold}
		if got := r.Passed(); got != tt.want {
			t.Errorf("Report{Score: %d, Threshold: %d}.Passed() = %v, want %v",
				tt.score, tt.threshold, got, tt.want)
		}
	}
}

func TestParse_ColorThresholds_Valid(t *testing.T) {
	input := `{
		"title": "Test",
		"score": 85,
		"threshold": 75,
		"thresholds": {
			"greenAbove": 80,
			"yellowAbove": 50
		}
	}`

	report, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if report.Thresholds == nil {
		t.Fatal("Thresholds should not be nil")
	}
	if report.Thresholds.GreenAbove != 80 {
		t.Errorf("GreenAbove = %d, want 80", report.Thresholds.GreenAbove)
	}
	if report.Thresholds.YellowAbove != 50 {
		t.Errorf("YellowAbove = %d, want 50", report.Thresholds.YellowAbove)
	}
}

func TestParse_ColorThresholds_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "greenAbove too high",
			input: `{"title": "Test", "score": 85, "threshold": 75, "thresholds": {"greenAbove": 150, "yellowAbove": 50}}`,
			want:  "thresholds.greenAbove must be between 0 and 100",
		},
		{
			name:  "yellowAbove negative",
			input: `{"title": "Test", "score": 85, "threshold": 75, "thresholds": {"greenAbove": 75, "yellowAbove": -10}}`,
			want:  "thresholds.yellowAbove must be between 0 and 100",
		},
		{
			name:  "greenAbove less than yellowAbove",
			input: `{"title": "Test", "score": 85, "threshold": 75, "thresholds": {"greenAbove": 40, "yellowAbove": 60}}`,
			want:  "thresholds.greenAbove (40) must be >= thresholds.yellowAbove (60)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tt.input))
			if err == nil {
				t.Fatal("Parse() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.want)
			}
		})
	}
}

func TestReport_EffectiveColorThresholds(t *testing.T) {
	// Without thresholds, should return defaults
	r := &Report{Title: "Test", Score: 85, Threshold: 75}
	thresholds := r.EffectiveColorThresholds()
	if thresholds.GreenAbove != 75 || thresholds.YellowAbove != 50 {
		t.Errorf("Expected defaults (75, 50), got (%d, %d)", thresholds.GreenAbove, thresholds.YellowAbove)
	}

	// With thresholds, should return them
	r.Thresholds = &ColorThresholds{GreenAbove: 90, YellowAbove: 70}
	thresholds = r.EffectiveColorThresholds()
	if thresholds.GreenAbove != 90 || thresholds.YellowAbove != 70 {
		t.Errorf("Expected (90, 70), got (%d, %d)", thresholds.GreenAbove, thresholds.YellowAbove)
	}
}

func TestReport_CalculateScore(t *testing.T) {
	tests := []struct {
		name    string
		factors []Factor
		want    int
	}{
		{
			name:    "no factors",
			factors: nil,
			want:    0,
		},
		{
			name: "equal weights",
			factors: []Factor{
				{Name: "A", Score: 80, Weight: 50},
				{Name: "B", Score: 60, Weight: 50},
			},
			want: 70, // (80*50 + 60*50) / 100 = 70
		},
		{
			name: "different weights",
			factors: []Factor{
				{Name: "A", Score: 100, Weight: 75},
				{Name: "B", Score: 0, Weight: 25},
			},
			want: 75, // (100*75 + 0*25) / 100 = 75
		},
		{
			name: "single factor",
			factors: []Factor{
				{Name: "A", Score: 85, Weight: 100},
			},
			want: 85,
		},
		{
			name: "zero total weight",
			factors: []Factor{
				{Name: "A", Score: 50, Weight: 0},
			},
			want: 0,
		},
		{
			name: "rounding up",
			factors: []Factor{
				{Name: "A", Score: 100, Weight: 1},
				{Name: "B", Score: 0, Weight: 2},
			},
			want: 33, // (100*1 + 0*2) / 3 = 33.33... rounds to 33
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Report{Title: "Test", Threshold: 75, Factors: tt.factors}
			got := r.CalculateScore()
			if got != tt.want {
				t.Errorf("CalculateScore() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParse_AutoCalculateScore(t *testing.T) {
	// When score is omitted but factors exist, score should be auto-calculated
	input := `{
		"title": "Auto Test",
		"threshold": 75,
		"factors": [
			{"name": "A", "score": 80, "weight": 50},
			{"name": "B", "score": 60, "weight": 50}
		]
	}`

	report, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// (80*50 + 60*50) / 100 = 70
	if report.Score != 70 {
		t.Errorf("Auto-calculated score = %d, want 70", report.Score)
	}
}

func TestParse_ExplicitScoreNotOverridden(t *testing.T) {
	// When score is explicitly provided, it should not be overridden
	input := `{
		"title": "Explicit Test",
		"score": 50,
		"threshold": 75,
		"factors": [
			{"name": "A", "score": 100, "weight": 100}
		]
	}`

	report, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Explicit score should be kept (not overridden to 100 from factors)
	if report.Score != 50 {
		t.Errorf("Score = %d, want 50 (explicit value)", report.Score)
	}
}
