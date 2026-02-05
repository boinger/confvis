package gitleaks

import (
	"context"
	"testing"

	"github.com/boinger/confvis/internal/sources"
)

// mockFetcher implements Fetcher for testing.
type mockFetcher struct {
	report   Report
	scanErr  error
}

func (m *mockFetcher) Scan(_ context.Context, _ string) (Report, error) {
	return m.report, m.scanErr
}

func TestSource_FetchWithClient(t *testing.T) {
	tests := []struct {
		name      string
		report    Report
		wantScore int
	}{
		{
			name:      "no leaks",
			report:    Report{},
			wantScore: 100,
		},
		{
			name: "one leak",
			report: Report{
				{RuleID: "generic-api-key", File: "config.yaml", Secret: "***"},
			},
			wantScore: 75, // 100 - 25 (one secret)
		},
		{
			name: "multiple leaks",
			report: Report{
				{RuleID: "generic-api-key", File: "config.yaml"},
				{RuleID: "aws-access-key", File: ".env"},
				{RuleID: "github-token", File: "secrets.json"},
			},
			wantScore: 25, // 100 - 75 (three secrets)
		},
		{
			name: "four leaks hits zero",
			report: Report{
				{RuleID: "leak1", File: "file1"},
				{RuleID: "leak2", File: "file2"},
				{RuleID: "leak3", File: "file3"},
				{RuleID: "leak4", File: "file4"},
			},
			wantScore: 0, // 100 - 100 = 0
		},
		{
			name: "many leaks stays at zero",
			report: Report{
				{RuleID: "leak1", File: "file1"},
				{RuleID: "leak2", File: "file2"},
				{RuleID: "leak3", File: "file3"},
				{RuleID: "leak4", File: "file4"},
				{RuleID: "leak5", File: "file5"},
				{RuleID: "leak6", File: "file6"},
			},
			wantScore: 0, // Capped at 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Source{}
			mock := &mockFetcher{report: tt.report}

			report, err := s.FetchWithClient(context.Background(), mock, defaultOpts(), ".")
			if err != nil {
				t.Fatalf("FetchWithClient() error = %v", err)
			}

			if report.ScoreValue() != tt.wantScore {
				t.Errorf("Score = %d, want %d", report.Score, tt.wantScore)
			}

			if report.Source != sourceName {
				t.Errorf("Source = %q, want %q", report.Source, sourceName)
			}

			if len(report.Factors) != 1 {
				t.Errorf("len(Factors) = %d, want 1", len(report.Factors))
			}
		})
	}
}

func TestSource_FetchWithClient_Title(t *testing.T) {
	s := &Source{}
	mock := &mockFetcher{report: Report{}}

	opts := sources.Options{
		Project:   "/path/to/myproject",
		Threshold: 75,
		Title:     "Custom Title",
	}

	report, err := s.FetchWithClient(context.Background(), mock, opts, "/path/to/myproject")
	if err != nil {
		t.Fatalf("FetchWithClient() error = %v", err)
	}

	if report.Title != "Custom Title" {
		t.Errorf("Title = %q, want %q", report.Title, "Custom Title")
	}
}

func TestSource_FetchWithClient_DefaultTitle(t *testing.T) {
	s := &Source{}
	mock := &mockFetcher{report: Report{}}

	opts := sources.Options{
		Project:   "/path/to/myproject",
		Threshold: 75,
	}

	report, err := s.FetchWithClient(context.Background(), mock, opts, "/path/to/myproject")
	if err != nil {
		t.Fatalf("FetchWithClient() error = %v", err)
	}

	if report.Title != "myproject" {
		t.Errorf("Title = %q, want %q", report.Title, "myproject")
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
		Project:   ".",
		Threshold: 75,
	}
}

func TestSource_FetchWithClient_ScanError(t *testing.T) {
	s := &Source{}
	mock := &mockFetcher{
		scanErr: context.DeadlineExceeded,
	}

	_, err := s.FetchWithClient(context.Background(), mock, defaultOpts(), ".")
	if err == nil {
		t.Error("expected error when scan fails")
	}
}

func TestFinding_Fields(t *testing.T) {
	// Test that Finding struct can be populated with expected fields
	finding := Finding{
		RuleID:      "generic-api-key",
		Description: "Generic API Key",
		File:        "config.yaml",
		StartLine:   10,
		EndLine:     10,
		Secret:      "***REDACTED***",
		Commit:      "abc123",
		Author:      "developer",
		Email:       "dev@example.com",
	}

	if finding.RuleID != "generic-api-key" {
		t.Errorf("RuleID = %q, want %q", finding.RuleID, "generic-api-key")
	}
	if finding.File != "config.yaml" {
		t.Errorf("File = %q, want %q", finding.File, "config.yaml")
	}
}
