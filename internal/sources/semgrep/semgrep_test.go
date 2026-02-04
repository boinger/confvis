package semgrep

import (
	"strings"
	"testing"

	"github.com/boinger/confvis/internal/sources"
)

func TestSeverityScore(t *testing.T) {
	tests := []struct {
		name    string
		count   int
		penalty int
		want    int
	}{
		{"no findings", 0, 20, 100},
		{"one error", 1, 20, 80},
		{"three errors", 3, 20, 40},
		{"five errors", 5, 20, 0},
		{"six errors (capped)", 6, 20, 0},
		{"one warning", 1, 10, 90},
		{"five warnings", 5, 10, 50},
		{"one info", 1, 2, 98},
		{"many info", 50, 2, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SeverityScore(tt.count, tt.penalty)
			if got != tt.want {
				t.Errorf("SeverityScore(%d, %d) = %d, want %d", tt.count, tt.penalty, got, tt.want)
			}
		})
	}
}

func TestCountFromResults(t *testing.T) {
	results := []Result{
		{Extra: Extra{Severity: "ERROR"}},
		{Extra: Extra{Severity: "ERROR"}},
		{Extra: Extra{Severity: "WARNING"}},
		{Extra: Extra{Severity: "WARNING"}},
		{Extra: Extra{Severity: "WARNING"}},
		{Extra: Extra{Severity: "INFO"}},
	}

	counts := CountFromResults(results)

	if counts.Error != 2 {
		t.Errorf("Error = %d, want 2", counts.Error)
	}
	if counts.Warning != 3 {
		t.Errorf("Warning = %d, want 3", counts.Warning)
	}
	if counts.Info != 1 {
		t.Errorf("Info = %d, want 1", counts.Info)
	}
}

func TestParseFromReader(t *testing.T) {
	jsonData := `{
		"results": [
			{"check_id": "rule1", "path": "test.py", "extra": {"severity": "ERROR", "message": "test"}},
			{"check_id": "rule2", "path": "test.py", "extra": {"severity": "WARNING", "message": "test"}}
		]
	}`

	report, err := ParseFromReader(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("ParseFromReader() error = %v", err)
	}

	if len(report.Results) != 2 {
		t.Errorf("len(Results) = %d, want 2", len(report.Results))
	}
}

func TestSource_BuildReport(t *testing.T) {
	tests := []struct {
		name      string
		results   []Result
		wantScore int
	}{
		{
			name:      "no findings",
			results:   []Result{},
			wantScore: 100,
		},
		{
			name: "one error",
			results: []Result{
				{Extra: Extra{Severity: "ERROR"}},
			},
			wantScore: 92, // (80*40 + 100*35 + 100*25) / 100 = 92
		},
		{
			name: "mixed findings",
			results: []Result{
				{Extra: Extra{Severity: "ERROR"}},
				{Extra: Extra{Severity: "WARNING"}},
				{Extra: Extra{Severity: "INFO"}},
			},
			wantScore: 88, // (80*40 + 90*35 + 98*25) / 100 = 88
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Source{}
			report := &Report{Results: tt.results}
			opts := sources.Options{Threshold: 75}

			result, err := s.buildReport(report, opts, ".")
			if err != nil {
				t.Fatalf("buildReport() error = %v", err)
			}

			if result.Score != tt.wantScore {
				t.Errorf("Score = %d, want %d", result.Score, tt.wantScore)
			}

			if result.Source != sourceName {
				t.Errorf("Source = %q, want %q", result.Source, sourceName)
			}

			if len(result.Factors) != 3 {
				t.Errorf("len(Factors) = %d, want 3", len(result.Factors))
			}
		})
	}
}

func TestSource_Name(t *testing.T) {
	s := &Source{}
	if s.Name() != sourceName {
		t.Errorf("Name() = %q, want %q", s.Name(), sourceName)
	}
}

func TestSource_FetchFromReader(t *testing.T) {
	jsonData := `{
		"results": [
			{"check_id": "rule1", "path": "test.py", "extra": {"severity": "ERROR", "message": "test"}},
			{"check_id": "rule2", "path": "test.py", "extra": {"severity": "WARNING", "message": "test"}}
		]
	}`

	s := &Source{}
	opts := sources.Options{Threshold: 75}

	result, err := s.fetchFromReader(strings.NewReader(jsonData), opts)
	if err != nil {
		t.Fatalf("fetchFromReader() error = %v", err)
	}

	// Should have 1 error, 1 warning
	// Error: 80*40 = 3200
	// Warning: 90*35 = 3150
	// Info: 100*25 = 2500
	// Total: 8850/100 = 89 (integer rounding)
	wantScore := 89
	if result.Score != wantScore {
		t.Errorf("Score = %d, want %d", result.Score, wantScore)
	}
}
