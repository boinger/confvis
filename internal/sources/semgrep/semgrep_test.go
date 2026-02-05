package semgrep

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boinger/confvis/internal/sources"
)

func Test_countFromResults(t *testing.T) {
	results := []Result{
		{Extra: Extra{Severity: "ERROR"}},
		{Extra: Extra{Severity: "ERROR"}},
		{Extra: Extra{Severity: "WARNING"}},
		{Extra: Extra{Severity: "WARNING"}},
		{Extra: Extra{Severity: "WARNING"}},
		{Extra: Extra{Severity: "INFO"}},
	}

	counts := countFromResults(results)

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

func Test_parseFromReader(t *testing.T) {
	jsonData := `{
		"results": [
			{"check_id": "rule1", "path": "test.py", "extra": {"severity": "ERROR", "message": "test"}},
			{"check_id": "rule2", "path": "test.py", "extra": {"severity": "WARNING", "message": "test"}}
		]
	}`

	report, err := parseFromReader(strings.NewReader(jsonData))
	if err != nil {
		t.Fatalf("parseFromReader() error = %v", err)
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

			if result.ScoreValue() != tt.wantScore {
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
	if result.ScoreValue() != wantScore {
		t.Errorf("Score = %d, want %d", result.Score, wantScore)
	}
}

func TestSource_Registration(t *testing.T) {
	s := sources.Get("semgrep")
	if s == nil {
		t.Error("semgrep source not registered")
	}
}

func TestClient_NewClient(t *testing.T) {
	// Default command
	c := NewClient("")
	if c.command != DefaultCommand {
		t.Errorf("NewClient(\"\").command = %q, want %q", c.command, DefaultCommand)
	}

	// Custom command
	c = NewClient("docker run returntocorp/semgrep")
	if c.command != "docker run returntocorp/semgrep" {
		t.Errorf("NewClient(custom).command = %q, want custom", c.command)
	}
}

func TestClient_Scan_EmptyCommand(t *testing.T) {
	// Create client with empty command
	c := &Client{command: ""}

	_, err := c.Scan(context.Background(), ".", "")
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestCheckSemgrepError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		stderr     string
		wantNilErr bool
	}{
		{
			name:       "exit code 1 (findings found)",
			err:        &exec.ExitError{ProcessState: newExitState(1)},
			stderr:     "",
			wantNilErr: true,
		},
		{
			name:       "exit code 2 with stderr",
			err:        &exec.ExitError{ProcessState: newExitState(2)},
			stderr:     "fatal error: config not found",
			wantNilErr: false,
		},
		{
			name:       "exit code 2 without stderr",
			err:        &exec.ExitError{ProcessState: newExitState(2)},
			stderr:     "",
			wantNilErr: false,
		},
		{
			name:       "non-ExitError",
			err:        context.DeadlineExceeded,
			stderr:     "",
			wantNilErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stderr := []byte(tt.stderr)
			err := checkSemgrepError(tt.err, stderr)
			if (err == nil) != tt.wantNilErr {
				t.Errorf("checkSemgrepError() returned error = %v, wantNilErr = %v", err, tt.wantNilErr)
			}
		})
	}
}

// newExitState creates an exec.ExitError with the given exit code for testing.
// We use a helper that actually runs a command since ProcessState is not directly constructable.
func newExitState(code int) *os.ProcessState {
	// Run a command that exits with the desired code
	var cmd *exec.Cmd
	if code == 0 {
		cmd = exec.Command("true")
	} else {
		cmd = exec.Command("sh", "-c", "exit "+string(rune('0'+code)))
	}
	_ = cmd.Run()
	return cmd.ProcessState
}

func TestSource_Fetch_WithMockScript(t *testing.T) {
	// Create a mock semgrep script that outputs valid JSON
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-semgrep")

	// Create mock script with multi-line JSON for readability
	scriptContent := `#!/bin/bash
cat << 'EOF'
{"results":[
  {"check_id":"rule1","path":"test.py","extra":{"severity":"ERROR","message":"test"}},
  {"check_id":"rule2","path":"test.py","extra":{"severity":"WARNING","message":"test"}}
]}
EOF
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	s := &Source{}
	opts := sources.Options{
		Project:   tmpDir,
		Threshold: 75,
		Extra:     map[string]string{"semgrep-cmd": mockScript},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Verify report structure
	if report.Source != "semgrep" {
		t.Errorf("Source = %q, want %q", report.Source, "semgrep")
	}

	if len(report.Factors) != 3 {
		t.Errorf("len(Factors) = %d, want 3", len(report.Factors))
	}
}

func TestSource_Fetch_CleanScan(t *testing.T) {
	// Create a mock semgrep script that outputs no findings
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-semgrep")

	scriptContent := `#!/bin/bash
echo '{"results":[]}'
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	s := &Source{}
	opts := sources.Options{
		Project:   tmpDir,
		Threshold: 75,
		Extra:     map[string]string{"semgrep-cmd": mockScript},
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
	// Create a mock semgrep script
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-semgrep")

	scriptContent := `#!/bin/bash
echo '{"results":[]}'
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
		// No semgrep-cmd in Extra - should use env var
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
	mockScript := filepath.Join(tmpDir, "mock-semgrep")

	scriptContent := `#!/bin/bash
echo '{"results":[]}'
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	s := &Source{}
	opts := sources.Options{
		Project:   tmpDir,
		Title:     "My Custom Title",
		Threshold: 75,
		Extra:     map[string]string{"semgrep-cmd": mockScript},
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
	mockScript := filepath.Join(tmpDir, "mock-semgrep")

	scriptContent := `#!/bin/bash
echo '{"results":[]}'
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	s := &Source{}
	opts := sources.Options{
		// No Project - should default to "."
		Threshold: 75,
		Extra:     map[string]string{"semgrep-cmd": mockScript},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if report.Source != "semgrep" {
		t.Errorf("Source = %q, want %q", report.Source, "semgrep")
	}
}

func TestSource_Fetch_SemgrepError(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-semgrep")

	// Script that exits with error code 2 (not 1, which means findings found)
	scriptContent := `#!/bin/bash
echo "Error: something went wrong" >&2
exit 2
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	s := &Source{}
	opts := sources.Options{
		Project: tmpDir,
		Extra:   map[string]string{"semgrep-cmd": mockScript},
	}

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error for semgrep failure")
	}
}

func TestSource_Fetch_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-semgrep")

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
		Extra:   map[string]string{"semgrep-cmd": mockScript},
	}

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSource_Fetch_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-semgrep")

	// Script that checks for config flag in args
	scriptContent := `#!/bin/bash
echo '{"results":[]}'
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	s := &Source{}
	opts := sources.Options{
		Project:   tmpDir,
		Threshold: 75,
		Extra: map[string]string{
			"semgrep-cmd": mockScript,
			"config":      "p/security-audit",
		},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if report.Source != "semgrep" {
		t.Errorf("Source = %q, want %q", report.Source, "semgrep")
	}
}

func TestSource_Fetch_FromStdin(t *testing.T) {
	jsonData := `{"results":[{"check_id":"rule1","path":"test.py","extra":{"severity":"ERROR","message":"test"}}]}`

	s := &Source{
		Stdin: strings.NewReader(jsonData),
	}
	opts := sources.Options{
		Threshold: 75,
		Extra:     map[string]string{"from-stdin": "true"},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Should have 1 error
	// Error: 80*40 = 3200
	// Warning: 100*35 = 3500
	// Info: 100*25 = 2500
	// Total: 9200/100 = 92
	if report.ScoreValue() != 92 {
		t.Errorf("Score = %d, want 92", report.Score)
	}
}

func TestSource_Fetch_ExitCode1_FindingsFound(t *testing.T) {
	// Semgrep returns exit code 1 when findings are found - this should NOT be an error
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-semgrep")

	// Script that exits with code 1 (findings found) but outputs valid JSON
	scriptContent := `#!/bin/bash
echo '{"results":[{"check_id":"rule1","path":"test.py","extra":{"severity":"ERROR","message":"test"}}]}'
exit 1
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	s := &Source{}
	opts := sources.Options{
		Project:   tmpDir,
		Threshold: 75,
		Extra:     map[string]string{"semgrep-cmd": mockScript},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() should not error on exit code 1: %v", err)
	}

	if report.Source != "semgrep" {
		t.Errorf("Source = %q, want %q", report.Source, "semgrep")
	}
}
