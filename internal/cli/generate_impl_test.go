package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// ============================================================================
// generateImpl Tests
// ============================================================================

func TestGenerateImpl_BasicFlow(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stderr bytes.Buffer
	exitCalled := false

	deps := &GenerateDeps{
		FS:          fs,
		Stdin:       nil,
		Stderr:      &stderr,
		Verbose:     false,
		Quiet:       false,
		ExitFunc:    func(code int) { exitCalled = true },
		Config:      "config.json",
		Output:      "/output",
		InputFormat: "auto",
		Dark:        false,
		FailUnder:   0,
	}

	err := generateImpl(deps)
	if err != nil {
		t.Fatalf("generateImpl() error = %v", err)
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

	// Check that directory was created
	if !fs.HasDir("/output/dashboard") {
		t.Error("dashboard directory should be created")
	}

	if exitCalled {
		t.Error("exit should not be called when score passes")
	}
}

func TestGenerateImpl_StdinInput(t *testing.T) {
	fs := NewMockFileSystem()

	stdin := strings.NewReader(`{"title": "Stdin Test", "score": 90, "threshold": 75}`)
	var stderr bytes.Buffer

	deps := &GenerateDeps{
		FS:          fs,
		Stdin:       stdin,
		Stderr:      &stderr,
		Verbose:     false,
		Quiet:       false,
		ExitFunc:    func(code int) {},
		Config:      "-",
		Output:      "/output",
		InputFormat: "auto",
		Dark:        false,
		FailUnder:   0,
	}

	err := generateImpl(deps)
	if err != nil {
		t.Fatalf("generateImpl() error = %v", err)
	}

	badge := fs.GetFileContent("/output/badge.svg")
	if !strings.Contains(badge, "<svg") {
		t.Error("badge should contain SVG content")
	}
}

func TestGenerateImpl_DarkMode(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := &GenerateDeps{
		FS:          fs,
		Stdin:       nil,
		Stderr:      &bytes.Buffer{},
		Verbose:     false,
		Quiet:       false,
		ExitFunc:    func(code int) {},
		Config:      "config.json",
		Output:      "/output",
		InputFormat: "json",
		Dark:        true,
		FailUnder:   0,
	}

	err := generateImpl(deps)
	if err != nil {
		t.Fatalf("generateImpl() error = %v", err)
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

func TestGenerateImpl_FailUnderTriggersExit(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 70, "threshold": 75}`)

	var stderr bytes.Buffer
	exitCode := -1

	deps := &GenerateDeps{
		FS:          fs,
		Stdin:       nil,
		Stderr:      &stderr,
		Verbose:     false,
		Quiet:       false,
		ExitFunc:    func(code int) { exitCode = code },
		Config:      "config.json",
		Output:      "/output",
		InputFormat: "json",
		Dark:        false,
		FailUnder:   75,
	}

	err := generateImpl(deps)
	if err != nil {
		t.Fatalf("generateImpl() error = %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	if !strings.Contains(stderr.String(), "Score 70 is below threshold 75") {
		t.Errorf("expected failure message in stderr, got: %s", stderr.String())
	}
}

func TestGenerateImpl_FailUnderPasses(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	exitCalled := false

	deps := &GenerateDeps{
		FS:          fs,
		Stdin:       nil,
		Stderr:      &bytes.Buffer{},
		Verbose:     false,
		Quiet:       false,
		ExitFunc:    func(code int) { exitCalled = true },
		Config:      "config.json",
		Output:      "/output",
		InputFormat: "json",
		Dark:        false,
		FailUnder:   80,
	}

	err := generateImpl(deps)
	if err != nil {
		t.Fatalf("generateImpl() error = %v", err)
	}

	if exitCalled {
		t.Error("exit should not be called when score >= fail-under")
	}
}

func TestGenerateImpl_InvalidInputFormat(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := &GenerateDeps{
		FS:          fs,
		Stdin:       nil,
		Stderr:      &bytes.Buffer{},
		Verbose:     false,
		Quiet:       false,
		ExitFunc:    func(code int) {},
		Config:      "config.json",
		Output:      "/output",
		InputFormat: "invalid",
		Dark:        false,
		FailUnder:   0,
	}

	err := generateImpl(deps)
	if err == nil {
		t.Fatal("expected error for invalid input format")
	}

	if !strings.Contains(err.Error(), "invalid input-format") {
		t.Errorf("error should mention invalid input-format, got: %v", err)
	}
}

func TestGenerateImpl_FileOpenError(t *testing.T) {
	fs := NewMockFileSystem()
	// Don't set file content - file doesn't exist

	deps := &GenerateDeps{
		FS:          fs,
		Stdin:       nil,
		Stderr:      &bytes.Buffer{},
		Verbose:     false,
		Quiet:       false,
		ExitFunc:    func(code int) {},
		Config:      "nonexistent.json",
		Output:      "/output",
		InputFormat: "json",
		Dark:        false,
		FailUnder:   0,
	}

	err := generateImpl(deps)
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}

	if !strings.Contains(err.Error(), "opening config") {
		t.Errorf("error should mention opening config, got: %v", err)
	}
}

func TestGenerateImpl_DirectoryCreationError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetError("mkdir:/output/dashboard", errors.New("permission denied"))

	deps := &GenerateDeps{
		FS:          fs,
		Stdin:       nil,
		Stderr:      &bytes.Buffer{},
		Verbose:     false,
		Quiet:       false,
		ExitFunc:    func(code int) {},
		Config:      "config.json",
		Output:      "/output",
		InputFormat: "json",
		Dark:        false,
		FailUnder:   0,
	}

	err := generateImpl(deps)
	if err == nil {
		t.Fatal("expected error for directory creation failure")
	}

	if !strings.Contains(err.Error(), "creating output directory") {
		t.Errorf("error should mention creating output directory, got: %v", err)
	}
}

func TestGenerateImpl_BadgeCreationError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetError("create:/output/badge.svg", errors.New("disk full"))

	deps := &GenerateDeps{
		FS:          fs,
		Stdin:       nil,
		Stderr:      &bytes.Buffer{},
		Verbose:     false,
		Quiet:       false,
		ExitFunc:    func(code int) {},
		Config:      "config.json",
		Output:      "/output",
		InputFormat: "json",
		Dark:        false,
		FailUnder:   0,
	}

	err := generateImpl(deps)
	if err == nil {
		t.Fatal("expected error for badge creation failure")
	}

	if !strings.Contains(err.Error(), "creating badge file") {
		t.Errorf("error should mention creating badge file, got: %v", err)
	}
}

func TestGenerateImpl_DashboardCreationError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetError("create:/output/dashboard/index.html", errors.New("disk full"))

	deps := &GenerateDeps{
		FS:          fs,
		Stdin:       nil,
		Stderr:      &bytes.Buffer{},
		Verbose:     false,
		Quiet:       false,
		ExitFunc:    func(code int) {},
		Config:      "config.json",
		Output:      "/output",
		InputFormat: "json",
		Dark:        false,
		FailUnder:   0,
	}

	err := generateImpl(deps)
	if err == nil {
		t.Fatal("expected error for dashboard creation failure")
	}

	if !strings.Contains(err.Error(), "creating dashboard file") {
		t.Errorf("error should mention creating dashboard file, got: %v", err)
	}
}

func TestGenerateImpl_QuietSuppressesMessage(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 70, "threshold": 75}`)

	var stderr bytes.Buffer
	exitCode := -1

	deps := &GenerateDeps{
		FS:          fs,
		Stdin:       nil,
		Stderr:      &stderr,
		Verbose:     false,
		Quiet:       true,
		ExitFunc:    func(code int) { exitCode = code },
		Config:      "config.json",
		Output:      "/output",
		InputFormat: "json",
		Dark:        false,
		FailUnder:   75,
	}

	err := generateImpl(deps)
	if err != nil {
		t.Fatalf("generateImpl() error = %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	if stderr.Len() > 0 {
		t.Errorf("quiet mode should suppress stderr, got: %s", stderr.String())
	}
}

func TestGenerateImpl_YAMLInput(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.yaml", `title: YAML Test
score: 92
threshold: 75`)

	deps := &GenerateDeps{
		FS:          fs,
		Stdin:       nil,
		Stderr:      &bytes.Buffer{},
		Verbose:     false,
		Quiet:       false,
		ExitFunc:    func(code int) {},
		Config:      "config.yaml",
		Output:      "/output",
		InputFormat: "yaml",
		Dark:        false,
		FailUnder:   0,
	}

	err := generateImpl(deps)
	if err != nil {
		t.Fatalf("generateImpl() error = %v", err)
	}

	badge := fs.GetFileContent("/output/badge.svg")
	if !strings.Contains(badge, "92") {
		t.Error("badge should contain score from YAML")
	}
}

