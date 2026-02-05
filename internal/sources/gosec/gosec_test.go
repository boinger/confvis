package gosec

import (
	"context"
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
