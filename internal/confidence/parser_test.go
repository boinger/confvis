package confidence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseWithFormat_ValidJSON(t *testing.T) {
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

	report, err := ParseWithFormat(strings.NewReader(input), FormatJSON)
	if err != nil {
		t.Fatalf("ParseWithFormat() error = %v", err)
	}

	if report.Title != "Test Report" {
		t.Errorf("Title = %q, want %q", report.Title, "Test Report")
	}
	if report.ScoreValue() != 85 {
		t.Errorf("Score = %d, want %d", report.ScoreValue(), 85)
	}
	if report.Threshold != 75 {
		t.Errorf("Threshold = %d, want %d", report.Threshold, 75)
	}
	if len(report.Factors) != 2 {
		t.Errorf("len(Factors) = %d, want %d", len(report.Factors), 2)
	}
}

func TestParseWithFormat_InvalidScore(t *testing.T) {
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
		{
			name:  "factor threshold too high",
			input: `{"title": "Test", "score": 50, "threshold": 75, "factors": [{"name": "F1", "score": 50, "weight": 50, "threshold": 999}]}`,
			want:  `factor "F1" threshold must be between 0 and 100`,
		},
		{
			name:  "factor threshold negative",
			input: `{"title": "Test", "score": 50, "threshold": 75, "factors": [{"name": "F1", "score": 50, "weight": 50, "threshold": -5}]}`,
			want:  `factor "F1" threshold must be between 0 and 100`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseWithFormat(strings.NewReader(tt.input), FormatJSON)
			if err == nil {
				t.Fatal("ParseWithFormat() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.want)
			}
		})
	}
}

func TestParseWithFormat_MissingFields(t *testing.T) {
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
			_, err := ParseWithFormat(strings.NewReader(tt.input), FormatJSON)
			if err == nil {
				t.Fatal("ParseWithFormat() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.want)
			}
		})
	}
}

func TestParseWithFormat_InvalidJSON(t *testing.T) {
	_, err := ParseWithFormat(strings.NewReader("not json"), FormatJSON)
	if err == nil {
		t.Fatal("ParseWithFormat() expected error for invalid JSON")
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
		score := tt.score
		r := &Report{Score: &score, Threshold: tt.threshold}
		if got := r.IsPass(); got != tt.want {
			t.Errorf("Report{Score: %d, Threshold: %d}.IsPass() = %v, want %v",
				tt.score, tt.threshold, got, tt.want)
		}
	}
}

func TestParseWithFormat_ColorThresholds_Valid(t *testing.T) {
	input := `{
		"title": "Test",
		"score": 85,
		"threshold": 75,
		"thresholds": {
			"greenAbove": 80,
			"yellowAbove": 50
		}
	}`

	report, err := ParseWithFormat(strings.NewReader(input), FormatJSON)
	if err != nil {
		t.Fatalf("ParseWithFormat() error = %v", err)
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

func TestParseWithFormat_ColorThresholds_Invalid(t *testing.T) {
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
			_, err := ParseWithFormat(strings.NewReader(tt.input), FormatJSON)
			if err == nil {
				t.Fatal("ParseWithFormat() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.want)
			}
		})
	}
}

func TestReport_EffectiveColorThresholds(t *testing.T) {
	// Without thresholds, should return defaults
	score := 85
	r := &Report{Title: "Test", Score: &score, Threshold: 75}
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

func TestParseWithFormat_AutoCalculateScore(t *testing.T) {
	// When score is omitted but factors exist, score should be auto-calculated
	input := `{
		"title": "Auto Test",
		"threshold": 75,
		"factors": [
			{"name": "A", "score": 80, "weight": 50},
			{"name": "B", "score": 60, "weight": 50}
		]
	}`

	report, err := ParseWithFormat(strings.NewReader(input), FormatJSON)
	if err != nil {
		t.Fatalf("ParseWithFormat() error = %v", err)
	}

	// (80*50 + 60*50) / 100 = 70
	if report.ScoreValue() != 70 {
		t.Errorf("Auto-calculated score = %d, want 70", report.Score)
	}
}

func TestParseWithFormat_ExplicitScoreNotOverridden(t *testing.T) {
	// When score is explicitly provided, it should not be overridden
	input := `{
		"title": "Explicit Test",
		"score": 50,
		"threshold": 75,
		"factors": [
			{"name": "A", "score": 100, "weight": 100}
		]
	}`

	report, err := ParseWithFormat(strings.NewReader(input), FormatJSON)
	if err != nil {
		t.Fatalf("ParseWithFormat() error = %v", err)
	}

	// Explicit score should be kept (not overridden to 100 from factors)
	if report.ScoreValue() != 50 {
		t.Errorf("Score = %d, want 50 (explicit value)", report.Score)
	}
}

func TestParseWithFormat_MetadataFields(t *testing.T) {
	input := `{
		"title": "Test",
		"score": 85,
		"threshold": 75,
		"version": "1.0",
		"generatedAt": "2024-01-15T10:00:00Z",
		"source": "ci-pipeline"
	}`

	report, err := ParseWithFormat(strings.NewReader(input), FormatJSON)
	if err != nil {
		t.Fatalf("ParseWithFormat() error = %v", err)
	}

	if report.Version != "1.0" {
		t.Errorf("Version = %q, want %q", report.Version, "1.0")
	}
	if report.GeneratedAt != "2024-01-15T10:00:00Z" {
		t.Errorf("GeneratedAt = %q, want %q", report.GeneratedAt, "2024-01-15T10:00:00Z")
	}
	if report.Source != "ci-pipeline" {
		t.Errorf("Source = %q, want %q", report.Source, "ci-pipeline")
	}
}

func TestParseWithFormat_FactorURL(t *testing.T) {
	input := `{
		"title": "Test",
		"score": 85,
		"threshold": 75,
		"factors": [
			{"name": "Coverage", "score": 90, "weight": 100, "url": "https://codecov.io/report"}
		]
	}`

	report, err := ParseWithFormat(strings.NewReader(input), FormatJSON)
	if err != nil {
		t.Fatalf("ParseWithFormat() error = %v", err)
	}

	if len(report.Factors) != 1 {
		t.Fatalf("len(Factors) = %d, want 1", len(report.Factors))
	}
	if report.Factors[0].URL != "https://codecov.io/report" {
		t.Errorf("Factor URL = %q, want %q", report.Factors[0].URL, "https://codecov.io/report")
	}
}

func TestParseWithFormat_CustomLabels(t *testing.T) {
	input := `{
		"title": "Test",
		"score": 85,
		"threshold": 75,
		"passLabel": "OK",
		"failLabel": "NEEDS WORK"
	}`

	report, err := ParseWithFormat(strings.NewReader(input), FormatJSON)
	if err != nil {
		t.Fatalf("ParseWithFormat() error = %v", err)
	}

	if report.PassLabel != "OK" {
		t.Errorf("PassLabel = %q, want %q", report.PassLabel, "OK")
	}
	if report.FailLabel != "NEEDS WORK" {
		t.Errorf("FailLabel = %q, want %q", report.FailLabel, "NEEDS WORK")
	}
}

func TestReport_EffectiveLabels(t *testing.T) {
	// Default labels
	score := 85
	r := &Report{Title: "Test", Score: &score, Threshold: 75}
	if r.EffectivePassLabel() != "PASS" {
		t.Errorf("EffectivePassLabel() = %q, want PASS", r.EffectivePassLabel())
	}
	if r.EffectiveFailLabel() != "FAIL" {
		t.Errorf("EffectiveFailLabel() = %q, want FAIL", r.EffectiveFailLabel())
	}

	// Custom labels
	r.PassLabel = "OK"
	r.FailLabel = "NOT OK"
	if r.EffectivePassLabel() != "OK" {
		t.Errorf("EffectivePassLabel() = %q, want OK", r.EffectivePassLabel())
	}
	if r.EffectiveFailLabel() != "NOT OK" {
		t.Errorf("EffectiveFailLabel() = %q, want NOT OK", r.EffectiveFailLabel())
	}
}

func TestParseWithFormat_YAML(t *testing.T) {
	input := `title: Test Report
score: 85
threshold: 75
description: A test report
factors:
  - name: Factor 1
    score: 90
    weight: 50
  - name: Factor 2
    score: 80
    weight: 50
`

	report, err := ParseWithFormat(strings.NewReader(input), FormatYAML)
	if err != nil {
		t.Fatalf("ParseWithFormat(YAML) error = %v", err)
	}

	if report.Title != "Test Report" {
		t.Errorf("Title = %q, want %q", report.Title, "Test Report")
	}
	if report.ScoreValue() != 85 {
		t.Errorf("Score = %d, want %d", report.ScoreValue(), 85)
	}
	if report.Threshold != 75 {
		t.Errorf("Threshold = %d, want %d", report.Threshold, 75)
	}
	if len(report.Factors) != 2 {
		t.Errorf("len(Factors) = %d, want %d", len(report.Factors), 2)
	}
}

func TestParseWithFormat_YAML_Invalid(t *testing.T) {
	_, err := ParseWithFormat(strings.NewReader("not: [valid: yaml"), FormatYAML)
	if err == nil {
		t.Fatal("ParseWithFormat(YAML) expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "decoding YAML") {
		t.Errorf("error = %q, want to contain 'decoding YAML'", err.Error())
	}
}

func TestParseWithFormat_YAML_AutoCalculateScore(t *testing.T) {
	input := `title: Auto Test
threshold: 75
factors:
  - name: A
    score: 80
    weight: 50
  - name: B
    score: 60
    weight: 50
`

	report, err := ParseWithFormat(strings.NewReader(input), FormatYAML)
	if err != nil {
		t.Fatalf("ParseWithFormat(YAML) error = %v", err)
	}

	// (80*50 + 60*50) / 100 = 70
	if report.ScoreValue() != 70 {
		t.Errorf("Auto-calculated score = %d, want 70", report.Score)
	}
}

func TestIsKnownFormatExtension(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"config.json", true},
		{"config.yaml", true},
		{"config.yml", true},
		{"config.JSON", true},
		{"config.YAML", true},
		{"config.YML", true},
		{"config.txt", false},
		{"config", false},
		{"config.toml", false},
		{"config.xml", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsKnownFormatExtension(tt.path)
			if got != tt.want {
				t.Errorf("IsKnownFormatExtension(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		path string
		want Format
	}{
		{"config.json", FormatJSON},
		{"config.yaml", FormatYAML},
		{"config.yml", FormatYAML},
		{"config.YAML", FormatYAML},
		{"config.YML", FormatYAML},
		{"config.txt", FormatJSON}, // default to JSON
		{"config", FormatJSON},     // no extension defaults to JSON
	}

	for _, tt := range tests {
		got := DetectFormat(tt.path)
		if got != tt.want {
			t.Errorf("DetectFormat(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

// Tests for ParseFile and ParseFileWithFormat

func TestParseFileWithFormat_JSON(t *testing.T) {
	// Use existing testdata file (sample.json has title "API Service")
	report, err := ParseFileWithFormat("../../testdata/sample.json", FormatAuto)
	if err != nil {
		t.Fatalf("ParseFileWithFormat() error = %v", err)
	}

	if report.Title != "API Service" {
		t.Errorf("Title = %q, want %q", report.Title, "API Service")
	}
	if report.ScoreValue() != 85 {
		t.Errorf("Score = %d, want %d", report.ScoreValue(), 85)
	}
}

func TestParseFileWithFormat_YAML(t *testing.T) {
	// Use existing testdata file
	report, err := ParseFileWithFormat("../../testdata/sample.yaml", FormatAuto)
	if err != nil {
		t.Fatalf("ParseFileWithFormat() error = %v", err)
	}

	if report.Title != "Code Quality Report" {
		t.Errorf("Title = %q, want %q", report.Title, "Code Quality Report")
	}
	if report.ScoreValue() != 85 {
		t.Errorf("Score = %d, want %d", report.ScoreValue(), 85)
	}
}

func TestParseFileWithFormat_NonExistent(t *testing.T) {
	_, err := ParseFileWithFormat("nonexistent.json", FormatAuto)
	if err == nil {
		t.Error("ParseFileWithFormat() expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "opening file") {
		t.Errorf("error = %q, want to contain 'opening file'", err.Error())
	}
}

func TestParseFileWithFormat_ExplicitJSON(t *testing.T) {
	report, err := ParseFileWithFormat("../../testdata/sample.json", FormatJSON)
	if err != nil {
		t.Fatalf("ParseFileWithFormat() error = %v", err)
	}

	if report.ScoreValue() != 85 {
		t.Errorf("Score = %d, want %d", report.ScoreValue(), 85)
	}
}

func TestParseFileWithFormat_ExplicitYAML(t *testing.T) {
	report, err := ParseFileWithFormat("../../testdata/sample.yaml", FormatYAML)
	if err != nil {
		t.Fatalf("ParseFileWithFormat() error = %v", err)
	}

	if report.ScoreValue() != 85 {
		t.Errorf("Score = %d, want %d", report.ScoreValue(), 85)
	}
}

func TestParseFileWithFormat_AutoDetect(t *testing.T) {
	// Auto-detect from .json extension
	report, err := ParseFileWithFormat("../../testdata/sample.json", FormatAuto)
	if err != nil {
		t.Fatalf("ParseFileWithFormat(Auto) error = %v", err)
	}

	if report.ScoreValue() != 85 {
		t.Errorf("Score = %d, want %d", report.ScoreValue(), 85)
	}
}

func TestParseFileWithFormat_UnknownExtensionError(t *testing.T) {
	// Files with unrecognized extensions that fail to parse should mention the extension
	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "report.txt")
	if err := os.WriteFile(txtPath, []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	_, err := ParseFileWithFormat(txtPath, FormatAuto)
	if err == nil {
		t.Fatal("expected error for invalid content with unknown extension")
	}
	if !strings.Contains(err.Error(), "unrecognized extension") {
		t.Errorf("error = %q, want to contain 'unrecognized extension'", err.Error())
	}
	if !strings.Contains(err.Error(), ".json, .yaml, or .yml") {
		t.Errorf("error = %q, want to list recognized extensions", err.Error())
	}
}

func TestParseFileWithFormat_KnownExtensionError(t *testing.T) {
	// Files with known extensions that fail to parse should NOT mention extension
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(jsonPath, []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	_, err := ParseFileWithFormat(jsonPath, FormatAuto)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if strings.Contains(err.Error(), "unrecognized extension") {
		t.Errorf("error for known .json extension should not mention 'unrecognized extension': %q", err.Error())
	}
}

func TestParseFileWithFormat_InvalidContent(t *testing.T) {
	// Create temp file with invalid JSON
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	_, err := ParseFileWithFormat(invalidPath, FormatJSON)
	if err == nil {
		t.Error("ParseFileWithFormat() expected error for invalid JSON")
	}
}

func TestParseFileWithFormat_ValidationError(t *testing.T) {
	// Create temp file with JSON that fails validation (missing title)
	tmpDir := t.TempDir()
	noTitlePath := filepath.Join(tmpDir, "notitle.json")
	content := `{"score": 85, "threshold": 75}`
	if err := os.WriteFile(noTitlePath, []byte(content), 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	_, err := ParseFileWithFormat(noTitlePath, FormatJSON)
	if err == nil {
		t.Error("ParseFileWithFormat() expected validation error")
	}
	if !strings.Contains(err.Error(), "title is required") {
		t.Errorf("error = %q, want to contain 'title is required'", err.Error())
	}
}

func TestParseFileWithFormat_FactorValidation(t *testing.T) {
	// Create temp file with invalid factor score
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid_factor.json")
	content := `{
		"title": "Test",
		"score": 85,
		"threshold": 75,
		"factors": [{"name": "F1", "score": 150, "weight": 50}]
	}`
	if err := os.WriteFile(invalidPath, []byte(content), 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	_, err := ParseFileWithFormat(invalidPath, FormatAuto)
	if err == nil {
		t.Error("ParseFileWithFormat() expected validation error for invalid factor score")
	}
	if !strings.Contains(err.Error(), "factor[0] score must be between") {
		t.Errorf("error = %q, want to contain factor validation error", err.Error())
	}
}

func TestParseFileWithFormat_FactorWeightValidation(t *testing.T) {
	// Create temp file with invalid factor weight
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid_weight.json")
	content := `{
		"title": "Test",
		"score": 85,
		"threshold": 75,
		"factors": [{"name": "F1", "score": 50, "weight": 150}]
	}`
	if err := os.WriteFile(invalidPath, []byte(content), 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	_, err := ParseFileWithFormat(invalidPath, FormatAuto)
	if err == nil {
		t.Error("ParseFileWithFormat() expected validation error for invalid factor weight")
	}
	if !strings.Contains(err.Error(), "factor[0] weight must be between") {
		t.Errorf("error = %q, want to contain factor weight validation error", err.Error())
	}
}

func TestParseFileWithFormat_NegativeThreshold(t *testing.T) {
	// Create temp file with negative threshold
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "negative_threshold.json")
	content := `{"title": "Test", "score": 85, "threshold": -10}`
	if err := os.WriteFile(invalidPath, []byte(content), 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	_, err := ParseFileWithFormat(invalidPath, FormatAuto)
	if err == nil {
		t.Error("ParseFileWithFormat() expected validation error for negative threshold")
	}
}

func TestReport_ScoreValue(t *testing.T) {
	intPtr := func(i int) *int { return &i }

	tests := []struct {
		name  string
		score *int
		want  int
	}{
		{
			name:  "nil score returns 0",
			score: nil,
			want:  0,
		},
		{
			name:  "zero score returns 0",
			score: intPtr(0),
			want:  0,
		},
		{
			name:  "positive score returns value",
			score: intPtr(85),
			want:  85,
		},
		{
			name:  "max score returns 100",
			score: intPtr(100),
			want:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &Report{Score: tt.score}
			if got := report.ScoreValue(); got != tt.want {
				t.Errorf("ScoreValue() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseWithFormat_UnknownFieldJSON(t *testing.T) {
	// A typo in "threshold" should surface as a decode error rather than
	// silently parse to Threshold: 0 (which would make the gate a no-op).
	input := `{"title": "Test", "score": 85, "thresholdd": 75}`

	_, err := ParseWithFormat(strings.NewReader(input), FormatJSON)
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
	if !strings.Contains(err.Error(), "thresholdd") {
		t.Errorf("error should mention the unknown field name; got: %v", err)
	}
}

func TestParseWithFormat_UnknownFieldYAML(t *testing.T) {
	input := "title: Test\nscore: 85\nthresholdd: 75\n"

	_, err := ParseWithFormat(strings.NewReader(input), FormatYAML)
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
	if !strings.Contains(err.Error(), "thresholdd") {
		t.Errorf("error should mention the unknown field name; got: %v", err)
	}
}

func TestParseWithFormat_UnknownFactorField(t *testing.T) {
	// Unknown fields inside nested factor objects are also rejected.
	input := `{
		"title": "Test",
		"score": 85,
		"threshold": 75,
		"factors": [{"name": "F1", "score": 90, "weight": 50, "weigth": 60}]
	}`

	_, err := ParseWithFormat(strings.NewReader(input), FormatJSON)
	if err == nil {
		t.Fatal("expected error for unknown factor field, got nil")
	}
	if !strings.Contains(err.Error(), "weigth") {
		t.Errorf("error should mention the unknown field name; got: %v", err)
	}
}
