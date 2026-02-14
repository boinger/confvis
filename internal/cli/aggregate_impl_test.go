package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// ============================================================================
// aggregateImpl Tests
// ============================================================================

func TestAggregateImpl_BasicAggregation(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("report1.json", `{"title": "Report 1", "score": 90, "threshold": 75}`)
	fs.SetFileContent("report2.json", `{"title": "Report 2", "score": 70, "threshold": 75}`)

	var stderr bytes.Buffer
	exitCalled := false

	deps := &AggregateDeps{
		FS:        fs,
		Stderr:    &stderr,
		Verbose:   false,
		Quiet:     false,
		ExitFunc:  func(code int) { exitCalled = true },
		Configs:   []string{"report1.json", "report2.json"},
		Output:    "/output",
		Dark:      false,
		FailUnder: 0,
	}

	err := aggregateImpl(deps)
	if err != nil {
		t.Fatalf("aggregateImpl() error = %v", err)
	}

	// Check that badge was created
	badge := fs.GetFileContent("/output/badge.svg")
	if !strings.Contains(badge, "<svg") {
		t.Error("badge should contain SVG content")
	}

	// Check that dashboard was created
	dashboard := fs.GetFileContent("/output/dashboard/index.html")
	if !strings.Contains(dashboard, "<!DOCTYPE html>") {
		t.Error("dashboard should contain HTML content")
	}

	// Check individual badges were created
	badge1 := fs.GetFileContent("/output/report-1.svg")
	if !strings.Contains(badge1, "<svg") {
		t.Error("individual badge 1 should be created")
	}

	if exitCalled {
		t.Error("exit should not be called")
	}
}

func TestAggregateImpl_MultipleConfigsWithWeights(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("api.json", `{"title": "API", "score": 85, "threshold": 75}`)
	fs.SetFileContent("web.json", `{"title": "Web", "score": 95, "threshold": 75}`)

	deps := &AggregateDeps{
		FS:        fs,
		Stderr:    &bytes.Buffer{},
		Verbose:   false,
		Quiet:     false,
		ExitFunc:  func(code int) {},
		Configs:   []string{"api.json:60", "web.json:40"},
		Output:    "/output",
		Dark:      false,
		FailUnder: 0,
	}

	err := aggregateImpl(deps)
	if err != nil {
		t.Fatalf("aggregateImpl() error = %v", err)
	}

	// Weighted average: (85*60 + 95*40) / 100 = (5100 + 3800) / 100 = 89
	// The badge should reflect the aggregate score
	badge := fs.GetFileContent("/output/badge.svg")
	if !strings.Contains(badge, "<svg") {
		t.Error("badge should be created with weighted average")
	}
}

func TestAggregateImpl_GlobPatternExpansion(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("reports/r1.json", `{"title": "R1", "score": 80, "threshold": 75}`)
	fs.SetFileContent("reports/r2.json", `{"title": "R2", "score": 90, "threshold": 75}`)
	fs.SetGlobMatches("reports/*.json", []string{"reports/r1.json", "reports/r2.json"})

	deps := &AggregateDeps{
		FS:        fs,
		Stderr:    &bytes.Buffer{},
		Verbose:   false,
		Quiet:     false,
		ExitFunc:  func(code int) {},
		Configs:   []string{"reports/*.json"},
		Output:    "/output",
		Dark:      false,
		FailUnder: 0,
	}

	err := aggregateImpl(deps)
	if err != nil {
		t.Fatalf("aggregateImpl() error = %v", err)
	}

	dashboard := fs.GetFileContent("/output/dashboard/index.html")
	if !strings.Contains(dashboard, "R1") || !strings.Contains(dashboard, "R2") {
		t.Error("dashboard should contain both reports from glob")
	}
}

func TestAggregateImpl_EmptyConfigs(t *testing.T) {
	fs := NewMockFileSystem()

	deps := &AggregateDeps{
		FS:        fs,
		Stderr:    &bytes.Buffer{},
		Verbose:   false,
		Quiet:     false,
		ExitFunc:  func(code int) {},
		Configs:   []string{},
		Output:    "/output",
		Dark:      false,
		FailUnder: 0,
	}

	err := aggregateImpl(deps)
	if err == nil {
		t.Fatal("expected error for empty configs")
	}

	if !strings.Contains(err.Error(), "no reports found") {
		t.Errorf("error should mention no reports found, got: %v", err)
	}
}

func TestAggregateImpl_FileNotFound(t *testing.T) {
	fs := NewMockFileSystem()
	// Don't set file content - file doesn't exist

	deps := &AggregateDeps{
		FS:        fs,
		Stderr:    &bytes.Buffer{},
		Verbose:   false,
		Quiet:     false,
		ExitFunc:  func(code int) {},
		Configs:   []string{"nonexistent.json"},
		Output:    "/output",
		Dark:      false,
		FailUnder: 0,
	}

	err := aggregateImpl(deps)
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestAggregateImpl_ParseError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("invalid.json", `not valid json`)

	deps := &AggregateDeps{
		FS:        fs,
		Stderr:    &bytes.Buffer{},
		Verbose:   false,
		Quiet:     false,
		ExitFunc:  func(code int) {},
		Configs:   []string{"invalid.json"},
		Output:    "/output",
		Dark:      false,
		FailUnder: 0,
	}

	err := aggregateImpl(deps)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestAggregateImpl_FailUnderTriggersExit(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("low.json", `{"title": "Low", "score": 50, "threshold": 75}`)

	var stderr bytes.Buffer
	exitCode := -1

	deps := &AggregateDeps{
		FS:       fs,
		Stderr:   &stderr,
		Verbose:  false,
		Quiet:    false,
		ExitFunc: func(code int) { exitCode = code },
		Configs:  []string{"low.json"},
		Output:   "/output",
		Dark:     false,
		FailUnder: 60,
	}

	err := aggregateImpl(deps)
	if err != nil {
		t.Fatalf("aggregateImpl() error = %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	if !strings.Contains(stderr.String(), "Aggregate score 50 is below threshold 60") {
		t.Errorf("expected failure message in stderr, got: %s", stderr.String())
	}
}

func TestAggregateImpl_DarkMode(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("report.json", `{"title": "Dark Test", "score": 85, "threshold": 75}`)

	deps := &AggregateDeps{
		FS:        fs,
		Stderr:    &bytes.Buffer{},
		Verbose:   false,
		Quiet:     false,
		ExitFunc:  func(code int) {},
		Configs:   []string{"report.json"},
		Output:    "/output",
		Dark:      true,
		FailUnder: 0,
	}

	err := aggregateImpl(deps)
	if err != nil {
		t.Fatalf("aggregateImpl() error = %v", err)
	}

	badge := fs.GetFileContent("/output/badge.svg")
	if !strings.Contains(badge, "#0d1117") {
		t.Error("dark mode badge should use dark background")
	}

	dashboard := fs.GetFileContent("/output/dashboard/index.html")
	if !strings.Contains(dashboard, `class="dark"`) {
		t.Error("dark mode dashboard should have dark class")
	}
}

func TestAggregateImpl_DirectoryCreationError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("report.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetError("mkdir:/output/dashboard", errors.New("permission denied"))

	deps := &AggregateDeps{
		FS:        fs,
		Stderr:    &bytes.Buffer{},
		Verbose:   false,
		Quiet:     false,
		ExitFunc:  func(code int) {},
		Configs:   []string{"report.json"},
		Output:    "/output",
		Dark:      false,
		FailUnder: 0,
	}

	err := aggregateImpl(deps)
	if err == nil {
		t.Fatal("expected error for directory creation failure")
	}

	if !strings.Contains(err.Error(), "creating output directory") {
		t.Errorf("error should mention creating output directory, got: %v", err)
	}
}

func TestAggregateImpl_QuietSuppressesMessage(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("low.json", `{"title": "Low", "score": 50, "threshold": 75}`)

	var stderr bytes.Buffer
	exitCode := -1

	deps := &AggregateDeps{
		FS:       fs,
		Stderr:   &stderr,
		Verbose:  false,
		Quiet:    true,
		ExitFunc: func(code int) { exitCode = code },
		Configs:  []string{"low.json"},
		Output:   "/output",
		Dark:     false,
		FailUnder: 60,
	}

	err := aggregateImpl(deps)
	if err != nil {
		t.Fatalf("aggregateImpl() error = %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	if stderr.Len() > 0 {
		t.Errorf("quiet mode should suppress stderr, got: %s", stderr.String())
	}
}


// ============================================================================
// Aggregate Fragment Mode Tests
// ============================================================================

func TestAggregateImpl_FragmentMode(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("report1.json", `{"title": "Report 1", "score": 90, "threshold": 75}`)
	fs.SetFileContent("report2.json", `{"title": "Report 2", "score": 70, "threshold": 75}`)

	deps := &AggregateDeps{
		FS:       fs,
		Stderr:   &bytes.Buffer{},
		Verbose:  false,
		Quiet:    false,
		ExitFunc: func(code int) {},
		Configs:  []string{"report1.json", "report2.json"},
		Output:   "/output",
		Dark:     false,
		Fragment: true,
	}

	err := aggregateImpl(deps)
	if err != nil {
		t.Fatalf("aggregateImpl() error = %v", err)
	}

	dashboard := fs.GetFileContent("/output/dashboard/index.html")
	if strings.Contains(dashboard, "<!DOCTYPE html>") {
		t.Error("fragment mode should not contain DOCTYPE")
	}
	if !strings.Contains(dashboard, "<div") {
		t.Error("fragment mode should contain div elements")
	}
}

