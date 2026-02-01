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
