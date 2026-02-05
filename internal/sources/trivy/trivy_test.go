package trivy

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/boinger/confvis/internal/sources"
	"github.com/boinger/confvis/internal/sources/scoring"
)

func TestSource_Name(t *testing.T) {
	s := &Source{}
	if got := s.Name(); got != "trivy" {
		t.Errorf("Name() = %q, want %q", got, "trivy")
	}
}

func TestSource_Registration(t *testing.T) {
	s := sources.Get("trivy")
	if s == nil {
		t.Error("trivy source not registered")
	}
}

func TestCountFromResults(t *testing.T) {
	tests := []struct {
		name    string
		results []Result
		want    IssueCounts
	}{
		{
			name:    "empty results",
			results: nil,
			want:    IssueCounts{},
		},
		{
			name: "no vulnerabilities",
			results: []Result{
				{
					Target:          "go.mod",
					Vulnerabilities: nil,
				},
			},
			want: IssueCounts{},
		},
		{
			name: "mixed severities",
			results: []Result{
				{
					Target: "go.mod",
					Vulnerabilities: []Vulnerability{
						{Severity: "CRITICAL"},
						{Severity: "HIGH"},
						{Severity: "HIGH"},
						{Severity: "MEDIUM"},
						{Severity: "MEDIUM"},
						{Severity: "MEDIUM"},
						{Severity: "LOW"},
					},
				},
			},
			want: IssueCounts{Critical: 1, High: 2, Medium: 3, Low: 1},
		},
		{
			name: "multiple results",
			results: []Result{
				{
					Target: "go.mod",
					Vulnerabilities: []Vulnerability{
						{Severity: "CRITICAL"},
					},
				},
				{
					Target: "package.json",
					Vulnerabilities: []Vulnerability{
						{Severity: "HIGH"},
						{Severity: "MEDIUM"},
					},
				},
			},
			want: IssueCounts{Critical: 1, High: 1, Medium: 1},
		},
		{
			name: "unknown severity",
			results: []Result{
				{
					Target: "file",
					Vulnerabilities: []Vulnerability{
						{Severity: "UNKNOWN"},
						{Severity: ""},
						{Severity: "OTHER"},
					},
				},
			},
			want: IssueCounts{Unknown: 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountFromResults(tt.results)
			if got != tt.want {
				t.Errorf("CountFromResults() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestClient_NewClient(t *testing.T) {
	// Default command
	c := NewClient("")
	if c.command != DefaultCommand {
		t.Errorf("NewClient(\"\").command = %q, want %q", c.command, DefaultCommand)
	}

	// Custom command
	c = NewClient("docker run aquasec/trivy")
	if c.command != "docker run aquasec/trivy" {
		t.Errorf("NewClient(custom).command = %q, want custom", c.command)
	}
}

func TestSource_Fetch_WithMockScript(t *testing.T) {
	// Create a mock trivy script that outputs valid JSON
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-trivy")

	// Create mock script
	scriptContent := `#!/bin/bash
echo '{"Results":[{"Target":"go.mod","Vulnerabilities":[{"VulnerabilityID":"CVE-2024-1234","Severity":"HIGH"},{"VulnerabilityID":"CVE-2024-5678","Severity":"MEDIUM"}]}]}'
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	s := &Source{}
	opts := sources.Options{
		Project:   tmpDir,
		Threshold: 75,
		Extra:     map[string]string{"trivy-cmd": mockScript},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Verify report structure
	if report.Source != "trivy" {
		t.Errorf("Source = %q, want %q", report.Source, "trivy")
	}

	if len(report.Factors) != 4 {
		t.Errorf("len(Factors) = %d, want 4", len(report.Factors))
	}

	// Verify severity counts in factor descriptions
	// 0 critical, 1 high, 1 medium, 0 low
	expectedDescs := map[string]string{
		"Critical Vulnerabilities": "0 critical",
		"High Vulnerabilities":     "1 high",
		"Medium Vulnerabilities":   "1 medium",
		"Low Vulnerabilities":      "0 low",
	}

	for _, f := range report.Factors {
		if expected, ok := expectedDescs[f.Name]; ok {
			if f.Description != expected {
				t.Errorf("Factor %q description = %q, want %q", f.Name, f.Description, expected)
			}
		}
	}
}

func TestSource_Fetch_CleanScan(t *testing.T) {
	// Create a mock trivy script that outputs no vulnerabilities
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-trivy")

	scriptContent := `#!/bin/bash
echo '{"Results":[]}'
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	s := &Source{}
	opts := sources.Options{
		Project:   tmpDir,
		Threshold: 75,
		Extra:     map[string]string{"trivy-cmd": mockScript},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Clean scan should have score of 100
	if report.ScoreValue() != 100 {
		t.Errorf("Score = %d, want 100 for clean scan", report.Score)
	}
}

func TestSource_Fetch_EnvVarFallback(t *testing.T) {
	// Create a mock trivy script
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-trivy")

	scriptContent := `#!/bin/bash
echo '{"Results":[]}'
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	// Set env var
	t.Setenv(EnvCommand, mockScript)

	s := &Source{}
	opts := sources.Options{
		Project:   tmpDir,
		Threshold: 75,
		// No trivy-cmd in Extra - should use env var
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if report.ScoreValue() != 100 {
		t.Errorf("Score = %d, want 100", report.Score)
	}
}

func TestSource_Fetch_CustomTitle(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-trivy")

	scriptContent := `#!/bin/bash
echo '{"Results":[]}'
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	s := &Source{}
	opts := sources.Options{
		Project:   tmpDir,
		Title:     "My Custom Title",
		Threshold: 75,
		Extra:     map[string]string{"trivy-cmd": mockScript},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if report.Title != "My Custom Title" {
		t.Errorf("Title = %q, want %q", report.Title, "My Custom Title")
	}
}

func TestSource_Fetch_DefaultPath(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-trivy")

	scriptContent := `#!/bin/bash
echo '{"Results":[]}'
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	s := &Source{}
	opts := sources.Options{
		// No Project - should default to "."
		Threshold: 75,
		Extra:     map[string]string{"trivy-cmd": mockScript},
	}

	// This will scan the current directory, which should work
	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if report.Source != "trivy" {
		t.Errorf("Source = %q, want %q", report.Source, "trivy")
	}
}

func TestSource_Fetch_TrivyError(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-trivy")

	// Script that exits with error
	scriptContent := `#!/bin/bash
echo "Error: something went wrong" >&2
exit 1
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	s := &Source{}
	opts := sources.Options{
		Project: tmpDir,
		Extra:   map[string]string{"trivy-cmd": mockScript},
	}

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error for trivy failure")
	}
}

func TestSource_Fetch_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-trivy")

	// Script that outputs invalid JSON
	scriptContent := `#!/bin/bash
echo 'not valid json'
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	s := &Source{}
	opts := sources.Options{
		Project: tmpDir,
		Extra:   map[string]string{"trivy-cmd": mockScript},
	}

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestClient_Scan_EmptyCommand(t *testing.T) {
	// Create client with technically non-empty but all-whitespace command
	// This tests the edge case in the Scan method
	c := &Client{command: ""}

	_, err := c.Scan(context.Background(), ".")
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestWeightedScore(t *testing.T) {
	// Test that the weighted score calculation matches expectations from the plan
	// Using the example: Critical: 0, High: 2, Medium: 5, Low: 9
	// Scores: 100, 60, 50, 55
	// Weights: 40, 30, 20, 10
	// Score = (100*40 + 60*30 + 50*20 + 55*10) / 100 = (4000 + 1800 + 1000 + 550) / 100 = 73.5 → 74

	critScore := scoring.SeverityScore(0, scoring.DefaultPenaltyCritical)  // 100
	highScore := scoring.SeverityScore(2, scoring.DefaultPenaltyHigh)      // 60
	medScore := scoring.SeverityScore(5, scoring.DefaultPenaltyMedium)     // 50
	lowScore := scoring.SeverityScore(9, scoring.DefaultPenaltyLow)        // 55

	if critScore != 100 {
		t.Errorf("Critical score = %d, want 100", critScore)
	}
	if highScore != 60 {
		t.Errorf("High score = %d, want 60", highScore)
	}
	if medScore != 50 {
		t.Errorf("Medium score = %d, want 50", medScore)
	}
	if lowScore != 55 {
		t.Errorf("Low score = %d, want 55", lowScore)
	}

	weightedSum := critScore*scoring.DefaultWeightCritical + highScore*scoring.DefaultWeightHigh + medScore*scoring.DefaultWeightMedium + lowScore*scoring.DefaultWeightLow
	totalWeight := scoring.DefaultWeightCritical + scoring.DefaultWeightHigh + scoring.DefaultWeightMedium + scoring.DefaultWeightLow
	expectedScore := (weightedSum + totalWeight/2) / totalWeight // Round to nearest

	// From the plan example: score should be around 74-75
	if expectedScore < 73 || expectedScore > 76 {
		t.Errorf("weighted score = %d, expected around 74-75", expectedScore)
	}
}

func TestAllCleanScore(t *testing.T) {
	// Project with no vulnerabilities should score 100
	critScore := scoring.SeverityScore(0, scoring.DefaultPenaltyCritical)
	highScore := scoring.SeverityScore(0, scoring.DefaultPenaltyHigh)
	medScore := scoring.SeverityScore(0, scoring.DefaultPenaltyMedium)
	lowScore := scoring.SeverityScore(0, scoring.DefaultPenaltyLow)

	// All should be 100
	if critScore != 100 || highScore != 100 || medScore != 100 || lowScore != 100 {
		t.Errorf("clean scores: crit=%d, high=%d, med=%d, low=%d - all should be 100",
			critScore, highScore, medScore, lowScore)
	}

	// Weighted average of all 100s is 100
	weightedSum := critScore*scoring.DefaultWeightCritical + highScore*scoring.DefaultWeightHigh + medScore*scoring.DefaultWeightMedium + lowScore*scoring.DefaultWeightLow
	totalWeight := scoring.DefaultWeightCritical + scoring.DefaultWeightHigh + scoring.DefaultWeightMedium + scoring.DefaultWeightLow
	score := (weightedSum + totalWeight/2) / totalWeight

	if score != 100 {
		t.Errorf("all-clean score = %d, want 100", score)
	}
}
