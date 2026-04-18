package trufflehog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/boinger/confvis/internal/sources"
)

// mockFetcher implements Fetcher for testing.
type mockFetcher struct {
	findings []Finding
	scanErr  error
}

func (m *mockFetcher) Scan(_ context.Context, _ string) ([]Finding, error) {
	return m.findings, m.scanErr
}

func (m *mockFetcher) ScanGit(_ context.Context, _ string) ([]Finding, error) {
	return m.findings, m.scanErr
}

func TestSource_FetchWithClient(t *testing.T) {
	tests := []struct {
		name      string
		findings  []Finding
		wantScore int
	}{
		{
			name:      "no findings",
			findings:  []Finding{},
			wantScore: 100,
		},
		{
			name: "one verified",
			findings: []Finding{
				{DetectorName: "AWS", Verified: true},
			},
			wantScore: 82, // (70*60 + 100*40) / 100 = 82
		},
		{
			name: "one unverified",
			findings: []Finding{
				{DetectorName: "Generic", Verified: false},
			},
			wantScore: 96, // (100*60 + 90*40) / 100 = 96
		},
		{
			name: "mixed findings",
			findings: []Finding{
				{DetectorName: "AWS", Verified: true},
				{DetectorName: "Generic", Verified: false},
				{DetectorName: "GitHub", Verified: true},
			},
			wantScore: 60, // (40*60 + 90*40) / 100 = 60
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Source{}
			mock := &mockFetcher{findings: tt.findings}

			report, err := s.FetchWithClient(context.Background(), mock, defaultOpts(), ".", "filesystem")
			if err != nil {
				t.Fatalf("FetchWithClient() error = %v", err)
			}

			if report.ScoreValue() != tt.wantScore {
				t.Errorf("Score = %d, want %d", report.Score, tt.wantScore)
			}

			if report.Source != sourceName {
				t.Errorf("Source = %q, want %q", report.Source, sourceName)
			}

			if len(report.Factors) != 2 {
				t.Errorf("len(Factors) = %d, want 2", len(report.Factors))
			}
		})
	}
}

func TestSource_FetchWithClient_GitMode(t *testing.T) {
	s := &Source{}
	mock := &mockFetcher{findings: []Finding{}}

	report, err := s.FetchWithClient(context.Background(), mock, defaultOpts(), "https://github.com/owner/repo.git", "git")
	if err != nil {
		t.Fatalf("FetchWithClient() error = %v", err)
	}

	if report.Title != "repo" {
		t.Errorf("Title = %q, want %q", report.Title, "repo")
	}
}

func TestSource_FetchWithClient_Title(t *testing.T) {
	s := &Source{}
	mock := &mockFetcher{findings: []Finding{}}

	opts := sources.Options{
		Project:   "/path/to/myproject",
		Threshold: 75,
		Title:     "Custom Title",
	}

	report, err := s.FetchWithClient(context.Background(), mock, opts, "/path/to/myproject", "filesystem")
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
		Project:   ".",
		Threshold: 75,
	}
}

func TestSource_FetchWithClient_ScanError(t *testing.T) {
	s := &Source{}
	mock := &mockFetcher{
		scanErr: context.DeadlineExceeded,
	}

	_, err := s.FetchWithClient(context.Background(), mock, defaultOpts(), ".", "filesystem")
	if err == nil {
		t.Error("expected error when scan fails")
	}
}

func Test_countFindingsByVerification(t *testing.T) {
	findings := []Finding{
		{Verified: true},
		{Verified: true},
		{Verified: false},
		{Verified: false},
		{Verified: false},
	}

	counts := countFindingsByVerification(findings)

	if counts.Verified != 2 {
		t.Errorf("Verified = %d, want 2", counts.Verified)
	}
	if counts.Unverified != 3 {
		t.Errorf("Unverified = %d, want 3", counts.Unverified)
	}
}

func Test_isGitURL(t *testing.T) {
	tests := []struct {
		target string
		want   bool
	}{
		{"https://github.com/owner/repo.git", true},
		{"http://github.com/owner/repo", true},
		{"git@github.com:owner/repo.git", true},
		{".", false},
		{"/path/to/local", false},
		{"./relative/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			got := isGitURL(tt.target)
			if got != tt.want {
				t.Errorf("isGitURL(%q) = %v, want %v", tt.target, got, tt.want)
			}
		})
	}
}

func Test_parseJSONLines(t *testing.T) {
	input := []byte(`{"DetectorName":"AWS","Verified":true}
{"DetectorName":"GitHub","Verified":false}
`)

	findings, err := parseJSONLines(input)
	if err != nil {
		t.Fatalf("parseJSONLines() error = %v", err)
	}

	if len(findings) != 2 {
		t.Errorf("len(findings) = %d, want 2", len(findings))
	}

	if findings[0].DetectorName != "AWS" {
		t.Errorf("findings[0].DetectorName = %q, want %q", findings[0].DetectorName, "AWS")
	}
	if !findings[0].Verified {
		t.Error("findings[0].Verified = false, want true")
	}
}

func Test_parseJSONLines_InvalidLine(t *testing.T) {
	input := []byte(`{"DetectorName":"AWS","Verified":true}
not valid json
`)
	_, err := parseJSONLines(input)
	if err == nil {
		t.Fatal("expected error for invalid JSON line")
	}
	if !strings.Contains(err.Error(), "parsing trufflehog finding") {
		t.Errorf("error = %q, want to contain 'parsing trufflehog finding'", err.Error())
	}
}

func Test_parseJSONLines_Empty(t *testing.T) {
	findings, err := parseJSONLines([]byte{})
	if err != nil {
		t.Fatalf("parseJSONLines() error = %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("len(findings) = %d, want 0", len(findings))
	}
}

// writeMockScript creates an executable bash script at the given path that
// prints scriptOutput to stdout and exits with the given code. Returns the
// script path (same as input) for convenient chaining.
func writeMockScript(t *testing.T, dir, name, scriptOutput string, exitCode int) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("mock-bash-script trufflehog tests rely on POSIX shell; skipping on Windows")
	}
	scriptPath := filepath.Join(dir, name)
	body := fmt.Sprintf("#!/bin/bash\ncat <<'TRUFFLEEOF'\n%s\nTRUFFLEEOF\n", scriptOutput)
	if exitCode != 0 {
		body += fmt.Sprintf("exit %d\n", exitCode)
	}
	if err := os.WriteFile(scriptPath, []byte(body), 0o755); err != nil { //#nosec G306 -- test fixture must be executable
		t.Fatalf("writing mock script: %v", err)
	}
	return scriptPath
}

func TestSource_Fetch_WithMockScript(t *testing.T) {
	tmpDir := t.TempDir()
	// Two findings, one verified, one unverified. JSON-lines format.
	output := `{"Verified":true,"DetectorName":"AWS","Raw":"AKIAEXAMPLE"}
{"Verified":false,"DetectorName":"GitHub","Raw":"ghp_example"}`
	script := writeMockScript(t, tmpDir, "mock-trufflehog", output, 183) // 183 = secrets found

	s := &Source{}
	opts := sources.Options{
		Project:   tmpDir,
		Threshold: 75,
		Extra:     map[string]string{"trufflehog-cmd": script},
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
	if len(report.Factors) != 2 {
		t.Fatalf("len(Factors) = %d, want 2", len(report.Factors))
	}
}

func TestSource_Fetch_GitMode(t *testing.T) {
	tmpDir := t.TempDir()
	output := `{"Verified":true,"DetectorName":"AWS","Raw":"AKIAEXAMPLE"}`
	script := writeMockScript(t, tmpDir, "mock-trufflehog-git", output, 183)

	s := &Source{}
	opts := sources.Options{
		Project:   "/some/local/path",
		Threshold: 75,
		Extra: map[string]string{
			"trufflehog-cmd": script,
			"mode":           "git",
		},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	// One verified finding → both factor rows present.
	if len(report.Factors) != 2 {
		t.Fatalf("len(Factors) = %d, want 2", len(report.Factors))
	}
}

func TestSource_Fetch_GitURLDetection(t *testing.T) {
	tmpDir := t.TempDir()
	output := `{"Verified":false,"DetectorName":"GitHub","Raw":"ghp_example"}`
	script := writeMockScript(t, tmpDir, "mock-trufflehog-url", output, 183)

	s := &Source{}
	// mode defaults to "filesystem", but target is a git URL → isGitURL
	// picks up the git branch regardless.
	opts := sources.Options{
		Project:   "https://github.com/example/repo.git",
		Threshold: 75,
		Extra:     map[string]string{"trufflehog-cmd": script},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	// Title should be derived from the git URL's repo name.
	if report.Title != "repo" {
		t.Errorf("Title = %q, want %q", report.Title, "repo")
	}
}

func TestSource_Fetch_DefaultTarget(t *testing.T) {
	tmpDir := t.TempDir()
	// Empty output = no findings.
	script := writeMockScript(t, tmpDir, "mock-trufflehog-default", "", 0)

	s := &Source{}
	opts := sources.Options{
		// Project intentionally empty → Fetch should default to "."
		Threshold: 75,
		Extra:     map[string]string{"trufflehog-cmd": script},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if report.ScoreValue() != 100 {
		t.Errorf("Score = %d, want 100 (no findings)", report.ScoreValue())
	}
}

func TestSource_Fetch_EnvVarFallback(t *testing.T) {
	tmpDir := t.TempDir()
	output := `{"Verified":false,"DetectorName":"GitHub","Raw":"ghp_example"}`
	script := writeMockScript(t, tmpDir, "mock-trufflehog-env", output, 183)

	// Use the env var path: no trufflehog-cmd in Extra, set TRUFFLEHOG_CMD.
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
