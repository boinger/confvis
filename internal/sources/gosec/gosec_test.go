package gosec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/boinger/confvis/internal/sources"
)

// mockFetcher implements Fetcher for testing.
type mockFetcher struct {
	report  *Report
	scanErr error
}

func (m *mockFetcher) Scan(_ context.Context, _ string) (*Report, error) {
	return m.report, m.scanErr
}

func TestSource_FetchWithClient(t *testing.T) {
	tests := []struct {
		name      string
		issues    []Issue
		wantScore int
	}{
		{
			name:      "no issues",
			issues:    []Issue{},
			wantScore: 100,
		},
		{
			name: "one high",
			issues: []Issue{
				{Severity: "HIGH", RuleID: "G101"},
			},
			wantScore: 90, // (80*50 + 100*35 + 100*15) / 100 = 90
		},
		{
			name: "one medium",
			issues: []Issue{
				{Severity: "MEDIUM", RuleID: "G104"},
			},
			wantScore: 97, // (100*50 + 90*35 + 100*15) / 100 = 96.5 -> 97
		},
		{
			name: "one low",
			issues: []Issue{
				{Severity: "LOW", RuleID: "G201"},
			},
			wantScore: 100, // (100*50 + 100*35 + 97*15) / 100 = 99.55 -> 100
		},
		{
			name: "mixed severities",
			issues: []Issue{
				{Severity: "HIGH", RuleID: "G101"},
				{Severity: "MEDIUM", RuleID: "G104"},
				{Severity: "LOW", RuleID: "G201"},
			},
			wantScore: 86, // (80*50 + 90*35 + 97*15 + 50) / 100 = 86
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Source{}
			mock := &mockFetcher{report: &Report{Issues: tt.issues}}

			report, err := s.FetchWithClient(context.Background(), mock, defaultOpts(), "./...")
			if err != nil {
				t.Fatalf("FetchWithClient() error = %v", err)
			}

			if report.ScoreValue() != tt.wantScore {
				t.Errorf("Score = %d, want %d", report.ScoreValue(), tt.wantScore)
			}

			if report.Source != sourceName {
				t.Errorf("Source = %q, want %q", report.Source, sourceName)
			}

			if len(report.Factors) != 3 {
				t.Errorf("len(Factors) = %d, want 3", len(report.Factors))
			}
		})
	}
}

func TestSource_FetchWithClient_Title(t *testing.T) {
	s := &Source{}
	mock := &mockFetcher{report: &Report{Issues: []Issue{}}}

	opts := sources.Options{
		Project:   "./...",
		Threshold: 75,
		Title:     "Custom Title",
	}

	report, err := s.FetchWithClient(context.Background(), mock, opts, "./...")
	if err != nil {
		t.Fatalf("FetchWithClient() error = %v", err)
	}

	if report.Title != "Custom Title" {
		t.Errorf("Title = %q, want %q", report.Title, "Custom Title")
	}
}

func TestSource_Name(t *testing.T) {
	s := &Source{}
	if s.Name() != sourceName {
		t.Errorf("Name() = %q, want %q", s.Name(), sourceName)
	}
}

func defaultOpts() sources.Options {
	return sources.Options{
		Project:   "./...",
		Threshold: 75,
	}
}

func TestSource_FetchWithClient_ScanError(t *testing.T) {
	s := &Source{}
	mock := &mockFetcher{
		scanErr: context.DeadlineExceeded,
	}

	_, err := s.FetchWithClient(context.Background(), mock, defaultOpts(), "./...")
	if err == nil {
		t.Error("expected error when scan fails")
	}
}

func Test_countFromIssues(t *testing.T) {
	issues := []Issue{
		{Severity: "HIGH"},
		{Severity: "HIGH"},
		{Severity: "MEDIUM"},
		{Severity: "MEDIUM"},
		{Severity: "MEDIUM"},
		{Severity: "LOW"},
	}

	counts := countFromIssues(issues)

	if counts.High != 2 {
		t.Errorf("High = %d, want 2", counts.High)
	}
	if counts.Medium != 3 {
		t.Errorf("Medium = %d, want 3", counts.Medium)
	}
	if counts.Low != 1 {
		t.Errorf("Low = %d, want 1", counts.Low)
	}
}

func Test_countFromIssues_CaseInsensitive(t *testing.T) {
	issues := []Issue{
		{Severity: "high"},
		{Severity: "High"},
		{Severity: "HIGH"},
	}

	counts := countFromIssues(issues)

	if counts.High != 3 {
		t.Errorf("High = %d, want 3", counts.High)
	}
}

func TestIssue_Fields(t *testing.T) {
	// Test that Issue struct can be populated with expected fields
	issue := Issue{
		Severity:   "HIGH",
		Confidence: "HIGH",
		RuleID:     "G101",
		Details:    "Potential hardcoded credentials",
		File:       "main.go",
		Line:       "42",
		Column:     "10",
		CWE:        CWE{ID: "798", URL: "https://cwe.mitre.org/data/definitions/798.html"},
	}

	if issue.Severity != "HIGH" {
		t.Errorf("Severity = %q, want %q", issue.Severity, "HIGH")
	}
	if issue.RuleID != "G101" {
		t.Errorf("RuleID = %q, want %q", issue.RuleID, "G101")
	}
	if issue.CWE.ID != "798" {
		t.Errorf("CWE.ID = %q, want %q", issue.CWE.ID, "798")
	}
}

func TestReport_Fields(t *testing.T) {
	report := Report{
		Issues: []Issue{{Severity: "HIGH"}},
		Stats: Stats{
			Files: 10,
			Lines: 1000,
			NoSec: 2,
			Found: 1,
		},
		Golang: GolangInfo{Version: "go1.21"},
	}

	if report.Stats.Files != 10 {
		t.Errorf("Stats.Files = %d, want 10", report.Stats.Files)
	}
	if report.Golang.Version != "go1.21" {
		t.Errorf("Golang.Version = %q, want %q", report.Golang.Version, "go1.21")
	}
}

// writeMockScript creates an executable bash script that prints scriptOutput
// to stdout and exits with the given code.
func writeMockScript(t *testing.T, dir, name, scriptOutput string, exitCode int) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("mock-bash-script gosec tests rely on POSIX shell; skipping on Windows")
	}
	scriptPath := filepath.Join(dir, name)
	body := fmt.Sprintf("#!/bin/bash\ncat <<'GOSECEOF'\n%s\nGOSECEOF\n", scriptOutput)
	if exitCode != 0 {
		body += fmt.Sprintf("exit %d\n", exitCode)
	}
	if err := os.WriteFile(scriptPath, []byte(body), 0o755); err != nil { //#nosec G306 -- test fixture must be executable
		t.Fatalf("writing mock script: %v", err)
	}
	return scriptPath
}

// marshalReport serializes a Report to the JSON shape gosec emits. Using a
// struct-then-marshal keeps test fixtures compact and under the line-length
// lint threshold.
func marshalReport(t *testing.T, r Report) string {
	t.Helper()
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshaling fixture report: %v", err)
	}
	return string(b)
}

func TestSource_Fetch_WithMockScript(t *testing.T) {
	tmpDir := t.TempDir()
	// Gosec JSON output: object with Issues/Stats/GolangInfo. Exit code 1 =
	// issues found, valid only when stdout contains "Issues" (gosec's
	// ambiguous-exit convention).
	output := marshalReport(t, Report{
		Issues: []Issue{
			{Severity: "HIGH", Confidence: "HIGH", RuleID: "G204", File: "main.go", Line: "42"},
			{Severity: "MEDIUM", Confidence: "HIGH", RuleID: "G304", File: "reader.go", Line: "10"},
		},
		Stats:  Stats{Files: 2, Lines: 100, Found: 2},
		Golang: GolangInfo{Version: "go1.26.2"},
	})
	script := writeMockScript(t, tmpDir, "mock-gosec", output, 1)

	s := &Source{}
	opts := sources.Options{
		Project:   tmpDir,
		Threshold: 75,
		Extra:     map[string]string{"gosec-cmd": script},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if report.Source != sourceName {
		t.Errorf("Source = %q, want %q", report.Source, sourceName)
	}
	if report.Threshold != 75 {
		t.Errorf("Threshold = %d, want 75", report.Threshold)
	}
	// Gosec builds three severity factors (High/Medium/Low).
	if len(report.Factors) != 3 {
		t.Fatalf("len(Factors) = %d, want 3", len(report.Factors))
	}
}

func TestSource_Fetch_NoIssues(t *testing.T) {
	tmpDir := t.TempDir()
	// No issues: gosec still emits a valid JSON report. Exit code 0.
	output := marshalReport(t, Report{
		Issues: []Issue{},
		Stats:  Stats{Files: 1, Lines: 50},
		Golang: GolangInfo{Version: "go1.26.2"},
	})
	script := writeMockScript(t, tmpDir, "mock-gosec-clean", output, 0)

	s := &Source{}
	opts := sources.Options{
		Project:   tmpDir,
		Threshold: 75,
		Extra:     map[string]string{"gosec-cmd": script},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if report.ScoreValue() != 100 {
		t.Errorf("Score = %d, want 100 (no issues)", report.ScoreValue())
	}
}

func TestSource_Fetch_DefaultTarget(t *testing.T) {
	tmpDir := t.TempDir()
	output := marshalReport(t, Report{Issues: []Issue{}, Golang: GolangInfo{Version: "go1.26.2"}})
	script := writeMockScript(t, tmpDir, "mock-gosec-default", output, 0)

	s := &Source{}
	opts := sources.Options{
		// Project empty → Fetch defaults to "./..."
		Threshold: 75,
		Extra:     map[string]string{"gosec-cmd": script},
	}

	if _, err := s.Fetch(context.Background(), opts); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
}

func TestSource_Fetch_EnvVarFallback(t *testing.T) {
	tmpDir := t.TempDir()
	output := marshalReport(t, Report{Issues: []Issue{}, Golang: GolangInfo{Version: "go1.26.2"}})
	script := writeMockScript(t, tmpDir, "mock-gosec-env", output, 0)

	t.Setenv(EnvCommand, script)

	s := &Source{}
	opts := sources.Options{
		Project:   tmpDir,
		Threshold: 75,
	}

	if _, err := s.Fetch(context.Background(), opts); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
}
