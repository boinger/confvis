package grype

import (
	"context"
	"os"
	"path/filepath"
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
		{"no vulnerabilities", 0, 33, 100},
		{"one critical", 1, 33, 67},
		{"two critical", 2, 33, 34},
		{"three critical", 3, 33, 1},
		{"four critical (capped)", 4, 33, 0},
		{"one high", 1, 20, 80},
		{"one medium", 1, 10, 90},
		{"one low", 1, 5, 95},
		{"many low", 25, 5, 0},
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

func TestCountFromMatches(t *testing.T) {
	matches := []Match{
		{Vulnerability: Vulnerability{Severity: "Critical"}},
		{Vulnerability: Vulnerability{Severity: "Critical"}},
		{Vulnerability: Vulnerability{Severity: "High"}},
		{Vulnerability: Vulnerability{Severity: "Medium"}},
		{Vulnerability: Vulnerability{Severity: "Medium"}},
		{Vulnerability: Vulnerability{Severity: "Medium"}},
		{Vulnerability: Vulnerability{Severity: "Low"}},
		{Vulnerability: Vulnerability{Severity: "Negligible"}},
		{Vulnerability: Vulnerability{Severity: "Unknown"}},
	}

	counts := CountFromMatches(matches)

	if counts.Critical != 2 {
		t.Errorf("Critical = %d, want 2", counts.Critical)
	}
	if counts.High != 1 {
		t.Errorf("High = %d, want 1", counts.High)
	}
	if counts.Medium != 3 {
		t.Errorf("Medium = %d, want 3", counts.Medium)
	}
	if counts.Low != 2 { // Low + Negligible
		t.Errorf("Low = %d, want 2", counts.Low)
	}
	if counts.Unknown != 1 {
		t.Errorf("Unknown = %d, want 1", counts.Unknown)
	}
}

func TestDeriveTitle(t *testing.T) {
	tests := []struct {
		target string
		want   string
	}{
		// For paths, derive to base name
		{"./src", "src"},
		// For container images, preserve the full name
		{"alpine:3.18", "alpine:3.18"},
		{"nginx:latest", "nginx:latest"},
		{"myrepo/myimage:v1.0", "myrepo/myimage:v1.0"},
		{"myimage", "myimage"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			got := deriveTitle(tt.target)
			if got != tt.want {
				t.Errorf("deriveTitle(%q) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}

func TestLooksLikeContainerImage(t *testing.T) {
	tests := []struct {
		target string
		want   bool
	}{
		{".", false},
		{"..", false},
		{"./src", false},
		{"/absolute/path", false},
		{"alpine:3.18", true},
		{"nginx:latest", true},
		{"myimage", true},            // Single word could be an image
		{"myrepo/myimage:v1.0", true}, // Image with registry/repo and tag
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			got := looksLikeContainerImage(tt.target)
			if got != tt.want {
				t.Errorf("looksLikeContainerImage(%q) = %v, want %v", tt.target, got, tt.want)
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

func TestSource_Registration(t *testing.T) {
	s := sources.Get("grype")
	if s == nil {
		t.Error("grype source not registered")
	}
}

func TestClient_NewClient(t *testing.T) {
	// Default command
	c := NewClient("")
	if c.command != DefaultCommand {
		t.Errorf("NewClient(\"\").command = %q, want %q", c.command, DefaultCommand)
	}

	// Custom command
	c = NewClient("docker run anchore/grype")
	if c.command != "docker run anchore/grype" {
		t.Errorf("NewClient(custom).command = %q, want custom", c.command)
	}
}

func TestClient_Scan_EmptyCommand(t *testing.T) {
	// Create client with empty command
	c := &Client{command: ""}

	_, err := c.Scan(context.Background(), ".")
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestSource_Fetch_WithMockScript(t *testing.T) {
	// Create a mock grype script that outputs valid JSON
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-grype")

	// Create mock script with multi-line JSON for readability
	scriptContent := `#!/bin/bash
cat << 'EOF'
{"matches":[
  {"vulnerability":{"id":"CVE-2024-1234","severity":"High"},"artifact":{"name":"test","version":"1.0"}},
  {"vulnerability":{"id":"CVE-2024-5678","severity":"Medium"},"artifact":{"name":"test2","version":"2.0"}}
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
		Extra:     map[string]string{"grype-cmd": mockScript},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Verify report structure
	if report.Source != "grype" {
		t.Errorf("Source = %q, want %q", report.Source, "grype")
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
	// Create a mock grype script that outputs no vulnerabilities
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-grype")

	scriptContent := `#!/bin/bash
echo '{"matches":[]}'
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	s := &Source{}
	opts := sources.Options{
		Project:   tmpDir,
		Threshold: 75,
		Extra:     map[string]string{"grype-cmd": mockScript},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Clean scan should have score of 100
	if report.Score != 100 {
		t.Errorf("Score = %d, want 100 for clean scan", report.Score)
	}
}

func TestSource_Fetch_EnvVarFallback(t *testing.T) {
	// Create a mock grype script
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-grype")

	scriptContent := `#!/bin/bash
echo '{"matches":[]}'
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
		// No grype-cmd in Extra - should use env var
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if report.Score != 100 {
		t.Errorf("Score = %d, want 100", report.Score)
	}
}

func TestSource_Fetch_CustomTitle(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-grype")

	scriptContent := `#!/bin/bash
echo '{"matches":[]}'
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	s := &Source{}
	opts := sources.Options{
		Project:   tmpDir,
		Title:     "My Custom Title",
		Threshold: 75,
		Extra:     map[string]string{"grype-cmd": mockScript},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if report.Title != "My Custom Title" {
		t.Errorf("Title = %q, want %q", report.Title, "My Custom Title")
	}
}

func TestSource_Fetch_DefaultTarget(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-grype")

	scriptContent := `#!/bin/bash
echo '{"matches":[]}'
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	s := &Source{}
	opts := sources.Options{
		// No Project - should default to "."
		Threshold: 75,
		Extra:     map[string]string{"grype-cmd": mockScript},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if report.Source != "grype" {
		t.Errorf("Source = %q, want %q", report.Source, "grype")
	}
}

func TestSource_Fetch_GrypeError(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-grype")

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
		Extra:   map[string]string{"grype-cmd": mockScript},
	}

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error for grype failure")
	}
}

func TestSource_Fetch_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-grype")

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
		Extra:   map[string]string{"grype-cmd": mockScript},
	}

	_, err := s.Fetch(context.Background(), opts)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSource_Fetch_ContainerImage(t *testing.T) {
	tmpDir := t.TempDir()
	mockScript := filepath.Join(tmpDir, "mock-grype")

	scriptContent := `#!/bin/bash
echo '{"matches":[]}'
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("writing mock script: %v", err)
	}

	s := &Source{}
	opts := sources.Options{
		Project:   "alpine:3.18", // Container image
		Threshold: 75,
		Extra:     map[string]string{"grype-cmd": mockScript},
	}

	report, err := s.Fetch(context.Background(), opts)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	// Title should be the full image name for container images
	if report.Title != "alpine:3.18" {
		t.Errorf("Title = %q, want %q", report.Title, "alpine:3.18")
	}
}
