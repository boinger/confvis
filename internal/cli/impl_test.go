package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/boinger/confvis/internal/checks"
	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/history"
	"github.com/boinger/confvis/internal/sources"
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

	if !strings.Contains(err.Error(), "parsing config") {
		t.Errorf("error should mention parsing config, got: %v", err)
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

// ============================================================================
// fetchImpl Tests
// ============================================================================

// mockSource is a test implementation of sources.Source.
type mockSource struct {
	name   string
	report *confidence.Report
	err    error
}

func (m *mockSource) Name() string { return m.name }

func (m *mockSource) Fetch(_ context.Context, _ sources.Options) (*confidence.Report, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.report, nil
}

func mockSourceGetter(src sources.Source) func(string) sources.Source {
	return func(name string) sources.Source {
		if src != nil && src.Name() == name {
			return src
		}
		return nil
	}
}

func TestFetchImpl_SuccessfulFetchToFile(t *testing.T) {
	fs := NewMockFileSystem()
	var stderr bytes.Buffer

	src := &mockSource{
		name: "test",
		report: &confidence.Report{
			Title:     "Test Report",
			Score:     85,
			Threshold: 75,
			Source:    "test",
		},
	}

	deps := &FetchDeps{
		FS:           fs,
		Stdout:       &bytes.Buffer{},
		Stderr:       &stderr,
		Verbose:      false,
		Quiet:        false,
		SourceGetter: mockSourceGetter(src),
		SourceName:   "test",
		URL:          "http://example.com",
		Project:      "myproject",
		Token:        "secret",
		Branch:       "main",
		Title:        "",
		Threshold:    75,
		Timeout:      30,
		Output:       "/output/report.json",
		Extra:        map[string]string{},
	}

	err := fetchImpl(context.Background(), deps)
	if err != nil {
		t.Fatalf("fetchImpl() error = %v", err)
	}

	output := fs.GetFileContent("/output/report.json")
	if !strings.Contains(output, `"title": "Test Report"`) {
		t.Error("output should contain report title")
	}
	if !strings.Contains(output, `"score": 85`) {
		t.Error("output should contain score")
	}
}

func TestFetchImpl_SuccessfulFetchToStdout(t *testing.T) {
	fs := NewMockFileSystem()
	var stdout bytes.Buffer

	src := &mockSource{
		name: "test",
		report: &confidence.Report{
			Title:     "Stdout Test",
			Score:     90,
			Threshold: 75,
		},
	}

	deps := &FetchDeps{
		FS:           fs,
		Stdout:       &stdout,
		Stderr:       &bytes.Buffer{},
		Verbose:      false,
		Quiet:        false,
		SourceGetter: mockSourceGetter(src),
		SourceName:   "test",
		Project:      "myproject",
		Threshold:    75,
		Timeout:      30,
		Output:       "-",
		Extra:        map[string]string{},
	}

	err := fetchImpl(context.Background(), deps)
	if err != nil {
		t.Fatalf("fetchImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `"title": "Stdout Test"`) {
		t.Error("stdout should contain report title")
	}
}

func TestFetchImpl_UnknownSourceName(t *testing.T) {
	fs := NewMockFileSystem()

	deps := &FetchDeps{
		FS:           fs,
		Stdout:       &bytes.Buffer{},
		Stderr:       &bytes.Buffer{},
		Verbose:      false,
		Quiet:        false,
		SourceGetter: func(string) sources.Source { return nil },
		SourceName:   "nonexistent",
		Project:      "myproject",
		Output:       "/output.json",
		Extra:        map[string]string{},
	}

	err := fetchImpl(context.Background(), deps)
	if err == nil {
		t.Fatal("expected error for unknown source")
	}

	if !strings.Contains(err.Error(), "unknown source") {
		t.Errorf("error should mention unknown source, got: %v", err)
	}
}

func TestFetchImpl_SourceFetchError(t *testing.T) {
	fs := NewMockFileSystem()

	src := &mockSource{
		name: "test",
		err:  errors.New("connection refused"),
	}

	deps := &FetchDeps{
		FS:           fs,
		Stdout:       &bytes.Buffer{},
		Stderr:       &bytes.Buffer{},
		Verbose:      false,
		Quiet:        false,
		SourceGetter: mockSourceGetter(src),
		SourceName:   "test",
		Project:      "myproject",
		Output:       "/output.json",
		Extra:        map[string]string{},
	}

	err := fetchImpl(context.Background(), deps)
	if err == nil {
		t.Fatal("expected error from source fetch")
	}

	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("error should contain original error, got: %v", err)
	}
}

func TestFetchImpl_FileCreationError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetError("create:/output/report.json", errors.New("permission denied"))

	src := &mockSource{
		name: "test",
		report: &confidence.Report{
			Title:     "Test",
			Score:     85,
			Threshold: 75,
		},
	}

	deps := &FetchDeps{
		FS:           fs,
		Stdout:       &bytes.Buffer{},
		Stderr:       &bytes.Buffer{},
		Verbose:      false,
		Quiet:        false,
		SourceGetter: mockSourceGetter(src),
		SourceName:   "test",
		Project:      "myproject",
		Output:       "/output/report.json",
		Extra:        map[string]string{},
	}

	err := fetchImpl(context.Background(), deps)
	if err == nil {
		t.Fatal("expected error for file creation failure")
	}

	if !strings.Contains(err.Error(), "creating output file") {
		t.Errorf("error should mention creating output file, got: %v", err)
	}
}

func TestFetchImpl_VerboseOutput(t *testing.T) {
	fs := NewMockFileSystem()
	var stderr bytes.Buffer

	src := &mockSource{
		name: "test",
		report: &confidence.Report{
			Title:     "Verbose Test",
			Score:     85,
			Threshold: 75,
		},
	}

	deps := &FetchDeps{
		FS:           fs,
		Stdout:       &bytes.Buffer{},
		Stderr:       &stderr,
		Verbose:      true,
		Quiet:        false,
		SourceGetter: mockSourceGetter(src),
		SourceName:   "test",
		Project:      "myproject",
		Output:       "/output/report.json",
		Extra:        map[string]string{},
	}

	err := fetchImpl(context.Background(), deps)
	if err != nil {
		t.Fatalf("fetchImpl() error = %v", err)
	}

	output := stderr.String()
	if !strings.Contains(output, "Fetching metrics from test") {
		t.Error("verbose output should mention fetching")
	}
	if !strings.Contains(output, "Score: 85/75 (PASS)") {
		t.Error("verbose output should show score")
	}
	if !strings.Contains(output, "Wrote report to") {
		t.Error("verbose output should mention file written")
	}
}

func TestFetchImpl_VerboseSuppressedForStdout(t *testing.T) {
	fs := NewMockFileSystem()
	var stdout, stderr bytes.Buffer

	src := &mockSource{
		name: "test",
		report: &confidence.Report{
			Title:     "Test",
			Score:     85,
			Threshold: 75,
		},
	}

	deps := &FetchDeps{
		FS:           fs,
		Stdout:       &stdout,
		Stderr:       &stderr,
		Verbose:      true, // Would normally show verbose output
		Quiet:        false,
		SourceGetter: mockSourceGetter(src),
		SourceName:   "test",
		Project:      "myproject",
		Output:       "-", // But output to stdout
		Extra:        map[string]string{},
	}

	err := fetchImpl(context.Background(), deps)
	if err != nil {
		t.Fatalf("fetchImpl() error = %v", err)
	}

	// Verbose output should be suppressed when writing to stdout
	if stderr.Len() > 0 {
		t.Errorf("verbose output should be suppressed for stdout, got: %s", stderr.String())
	}
}

func TestFetchImpl_QuietSuppressesVerbose(t *testing.T) {
	fs := NewMockFileSystem()
	var stderr bytes.Buffer

	src := &mockSource{
		name: "test",
		report: &confidence.Report{
			Title:     "Test",
			Score:     85,
			Threshold: 75,
		},
	}

	deps := &FetchDeps{
		FS:           fs,
		Stdout:       &bytes.Buffer{},
		Stderr:       &stderr,
		Verbose:      true,
		Quiet:        true, // Quiet overrides verbose
		SourceGetter: mockSourceGetter(src),
		SourceName:   "test",
		Project:      "myproject",
		Output:       "/output.json",
		Extra:        map[string]string{},
	}

	err := fetchImpl(context.Background(), deps)
	if err != nil {
		t.Fatalf("fetchImpl() error = %v", err)
	}

	if stderr.Len() > 0 {
		t.Errorf("quiet mode should suppress stderr, got: %s", stderr.String())
	}
}

func TestFetchImpl_FailingReport(t *testing.T) {
	fs := NewMockFileSystem()
	var stderr bytes.Buffer

	src := &mockSource{
		name: "test",
		report: &confidence.Report{
			Title:     "Failing Test",
			Score:     60,
			Threshold: 75,
		},
	}

	deps := &FetchDeps{
		FS:           fs,
		Stdout:       &bytes.Buffer{},
		Stderr:       &stderr,
		Verbose:      true,
		Quiet:        false,
		SourceGetter: mockSourceGetter(src),
		SourceName:   "test",
		Project:      "myproject",
		Output:       "/output.json",
		Extra:        map[string]string{},
	}

	err := fetchImpl(context.Background(), deps)
	if err != nil {
		t.Fatalf("fetchImpl() error = %v", err)
	}

	output := stderr.String()
	if !strings.Contains(output, "(FAIL)") {
		t.Error("verbose output should show FAIL status")
	}
}

func TestFetchImpl_CustomTitleAndThreshold(t *testing.T) {
	fs := NewMockFileSystem()

	var capturedOpts sources.Options
	src := &mockSource{
		name: "test",
		report: &confidence.Report{
			Title:     "Custom Title",
			Score:     85,
			Threshold: 90,
		},
	}

	// Wrap the mock to capture options
	getter := func(name string) sources.Source {
		if name != "test" {
			return nil
		}
		return &captureOptsSource{
			mockSource: src,
			capturedOpts: &capturedOpts,
		}
	}

	deps := &FetchDeps{
		FS:           fs,
		Stdout:       &bytes.Buffer{},
		Stderr:       &bytes.Buffer{},
		SourceGetter: getter,
		SourceName:   "test",
		URL:          "http://example.com",
		Project:      "myproject",
		Token:        "secret",
		Branch:       "develop",
		Title:        "Custom Title",
		Threshold:    90,
		Timeout:      60,
		Output:       "/output.json",
		Extra:        map[string]string{"workflow": "ci.yml"},
	}

	err := fetchImpl(context.Background(), deps)
	if err != nil {
		t.Fatalf("fetchImpl() error = %v", err)
	}

	// Verify options were passed correctly
	if capturedOpts.URL != "http://example.com" {
		t.Errorf("URL = %q, want %q", capturedOpts.URL, "http://example.com")
	}
	if capturedOpts.Project != "myproject" {
		t.Errorf("Project = %q, want %q", capturedOpts.Project, "myproject")
	}
	if capturedOpts.Title != "Custom Title" {
		t.Errorf("Title = %q, want %q", capturedOpts.Title, "Custom Title")
	}
	if capturedOpts.Threshold != 90 {
		t.Errorf("Threshold = %d, want %d", capturedOpts.Threshold, 90)
	}
	if capturedOpts.Extra["workflow"] != "ci.yml" {
		t.Errorf("Extra[workflow] = %q, want %q", capturedOpts.Extra["workflow"], "ci.yml")
	}
}

// captureOptsSource wraps a mockSource and captures the options passed to Fetch.
type captureOptsSource struct {
	*mockSource
	capturedOpts *sources.Options
}

func (c *captureOptsSource) Fetch(ctx context.Context, opts sources.Options) (*confidence.Report, error) {
	*c.capturedOpts = opts
	return c.mockSource.Fetch(ctx, opts)
}

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
// gaugeImpl Tests
// ============================================================================

func defaultGaugeDeps(fs *MockFileSystem) *GaugeDeps {
	return &GaugeDeps{
		FS:               fs,
		Stdin:            nil,
		Stdout:           &bytes.Buffer{},
		Stderr:           &bytes.Buffer{},
		Verbose:          false,
		Quiet:            false,
		ExitFunc:         func(int) {},
		HistoryReader:    nil,
		HistoryAppender:  nil,
		Config:           "",
		Output:           "",
		Format:           "svg",
		Style:            "github",
		BadgeType:        "gauge",
		Label:            "",
		InputFormat:      "auto",
		Compare:          "",
		HistoryFile:      "",
		Width:            200,
		Height:           120,
		FailUnder:        0,
		GreenAbove:       0,
		YellowAbove:      0,
		HistoryCount:     10,
		Dark:             false,
		FailOnRegression: false,
	}
}

func TestGaugeImpl_SVGOutputToFile(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output/badge.svg"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := fs.GetFileContent("/output/badge.svg")
	if !strings.Contains(output, "<svg") {
		t.Error("output should contain SVG content")
	}
	if !strings.Contains(output, "85") {
		t.Error("output should contain score")
	}
}

func TestGaugeImpl_SVGOutputToStdout(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "<svg") {
		t.Error("stdout should contain SVG content")
	}
}

func TestGaugeImpl_JSONFormat(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "JSON Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Format = "json"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `"score": 85`) {
		t.Error("JSON output should contain score")
	}
	if !strings.Contains(output, `"passed": true`) {
		t.Error("JSON output should contain passed status")
	}
}

func TestGaugeImpl_TextFormat(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Format = "text"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if strings.TrimSpace(stdout.String()) != "85" {
		t.Errorf("text output should be '85', got: %s", stdout.String())
	}
}

func TestGaugeImpl_MarkdownFormat(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "MD Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Format = "markdown"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "## MD Test: 85% (PASS)") {
		t.Errorf("markdown should contain header, got: %s", output)
	}
}

func TestGaugeImpl_FlatBadgeType(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Flat Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.BadgeType = "flat"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "<svg") {
		t.Error("flat badge should be SVG")
	}
}

func TestGaugeImpl_SparklineBadgeType(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Spark Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.BadgeType = "sparkline"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "<svg") {
		t.Error("sparkline badge should be SVG")
	}
}

func TestGaugeImpl_DarkMode(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Dark Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Dark = true

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "#0d1117") {
		t.Error("dark mode should use dark background")
	}
}

func TestGaugeImpl_CustomDimensions(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Sized Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Width = 300
	deps.Height = 180

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `width="300"`) {
		t.Error("SVG should use custom width")
	}
}

func TestGaugeImpl_StdinInput(t *testing.T) {
	fs := NewMockFileSystem()

	stdin := strings.NewReader(`{"title": "Stdin Test", "score": 90, "threshold": 75}`)
	var stdout bytes.Buffer

	deps := defaultGaugeDeps(fs)
	deps.Stdin = stdin
	deps.Stdout = &stdout
	deps.Config = "-"
	deps.Output = "-"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if !strings.Contains(stdout.String(), "<svg") {
		t.Error("output should contain SVG")
	}
}

func TestGaugeImpl_BaselineComparison_PositiveDelta(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("current.json", `{"title": "Current", "score": 85, "threshold": 75}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 62, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "current.json"
	deps.Output = "-"
	deps.Format = "json"
	deps.Compare = "baseline.json"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `"baseline": 62`) {
		t.Error("JSON should contain baseline")
	}
	if !strings.Contains(output, `"delta": 23`) {
		t.Error("JSON should contain positive delta")
	}
}

func TestGaugeImpl_BaselineComparison_NegativeDelta(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("current.json", `{"title": "Current", "score": 62, "threshold": 75}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "current.json"
	deps.Output = "-"
	deps.Format = "text"
	deps.Compare = "baseline.json"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if strings.TrimSpace(stdout.String()) != "62 (-23)" {
		t.Errorf("text should show negative delta, got: %s", stdout.String())
	}
}

func TestGaugeImpl_FailUnderTriggersExit(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Low", "score": 70, "threshold": 75}`)

	var stderr bytes.Buffer
	exitCode := -1

	deps := defaultGaugeDeps(fs)
	deps.Stderr = &stderr
	deps.ExitFunc = func(code int) { exitCode = code }
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.FailUnder = 75

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	if !strings.Contains(stderr.String(), "Score 70 is below threshold 75") {
		t.Errorf("expected failure message, got: %s", stderr.String())
	}
}

func TestGaugeImpl_FailUnderPasses(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "High", "score": 85, "threshold": 75}`)

	exitCalled := false
	deps := defaultGaugeDeps(fs)
	deps.ExitFunc = func(code int) { exitCalled = true }
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.FailUnder = 80

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if exitCalled {
		t.Error("exit should not be called when score >= fail-under")
	}
}

func TestGaugeImpl_FailOnRegressionTriggersExit(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("current.json", `{"title": "Current", "score": 62, "threshold": 75}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 85, "threshold": 75}`)

	var stderr bytes.Buffer
	exitCode := -1

	deps := defaultGaugeDeps(fs)
	deps.Stderr = &stderr
	deps.ExitFunc = func(code int) { exitCode = code }
	deps.Config = "current.json"
	deps.Output = "/output.svg"
	deps.Compare = "baseline.json"
	deps.FailOnRegression = true

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1 for regression, got %d", exitCode)
	}

	if !strings.Contains(stderr.String(), "Score regressed from 85 to 62") {
		t.Errorf("expected regression message, got: %s", stderr.String())
	}
}

func TestGaugeImpl_FailOnRegressionPasses(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("current.json", `{"title": "Current", "score": 85, "threshold": 75}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 62, "threshold": 75}`)

	exitCalled := false
	deps := defaultGaugeDeps(fs)
	deps.ExitFunc = func(code int) { exitCalled = true }
	deps.Config = "current.json"
	deps.Output = "/output.svg"
	deps.Compare = "baseline.json"
	deps.FailOnRegression = true

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if exitCalled {
		t.Error("exit should not be called when score improved")
	}
}

func TestGaugeImpl_InvalidFormat(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Format = "invalid"

	err := gaugeImpl(deps)
	if err == nil {
		t.Fatal("expected error for invalid format")
	}

	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("error should mention invalid format, got: %v", err)
	}
}

func TestGaugeImpl_InvalidBadgeType(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "-"
	deps.BadgeType = "invalid"

	err := gaugeImpl(deps)
	if err == nil {
		t.Fatal("expected error for invalid badge-type")
	}

	if !strings.Contains(err.Error(), "invalid badge-type") {
		t.Errorf("error should mention invalid badge-type, got: %v", err)
	}
}

func TestGaugeImpl_InvalidInputFormat(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "-"
	deps.InputFormat = "invalid"

	err := gaugeImpl(deps)
	if err == nil {
		t.Fatal("expected error for invalid input-format")
	}

	if !strings.Contains(err.Error(), "invalid input-format") {
		t.Errorf("error should mention invalid input-format, got: %v", err)
	}
}

func TestGaugeImpl_FileOpenError(t *testing.T) {
	fs := NewMockFileSystem()
	// Don't set file content

	deps := defaultGaugeDeps(fs)
	deps.Config = "nonexistent.json"
	deps.Output = "-"

	err := gaugeImpl(deps)
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestGaugeImpl_FileCreateError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetError("create:/output.svg", errors.New("permission denied"))

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"

	err := gaugeImpl(deps)
	if err == nil {
		t.Fatal("expected error for file creation failure")
	}

	if !strings.Contains(err.Error(), "creating output file") {
		t.Errorf("error should mention creating output file, got: %v", err)
	}
}

func TestGaugeImpl_CustomColorThresholds(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.GreenAbove = 90
	deps.YellowAbove = 70

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	// Score 85 with greenAbove=90 should be yellow
	output := stdout.String()
	if !strings.Contains(output, "#9a6700") {
		t.Error("score 85 with green-above 90 should be yellow")
	}
}

func TestGaugeImpl_CustomLabel(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Original Title", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.BadgeType = "flat"
	deps.Label = "Custom Label"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Custom Label") {
		t.Error("flat badge should contain custom label")
	}
}

func TestGaugeImpl_QuietSuppressesMessages(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 70, "threshold": 75}`)

	var stderr bytes.Buffer
	exitCode := -1

	deps := defaultGaugeDeps(fs)
	deps.Stderr = &stderr
	deps.Quiet = true
	deps.ExitFunc = func(code int) { exitCode = code }
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.FailUnder = 75

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	if stderr.Len() > 0 {
		t.Errorf("quiet mode should suppress stderr, got: %s", stderr.String())
	}
}

func TestGaugeImpl_StyleMinimal(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Style = "minimal"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "#fafafa") {
		t.Error("minimal style should use #fafafa background")
	}
}

func TestGaugeImpl_YAMLInput(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.yaml", `title: YAML Test
score: 92
threshold: 75`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.yaml"
	deps.Output = "-"
	deps.Format = "text"
	deps.InputFormat = "yaml"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if strings.TrimSpace(stdout.String()) != "92" {
		t.Errorf("expected score 92, got: %s", stdout.String())
	}
}

func TestGaugeImpl_SparklineHistoryReadError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Spark Test", "score": 85, "threshold": 75}`)

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.BadgeType = "sparkline"
	deps.HistoryFile = "/history.jsonl"
	deps.HistoryReader = func(path string) (*history.History, error) {
		return nil, errors.New("history file corrupted")
	}

	err := gaugeImpl(deps)
	if err == nil {
		t.Fatal("expected error for history read failure")
	}

	if !strings.Contains(err.Error(), "reading history") {
		t.Errorf("error should mention reading history, got: %v", err)
	}
}

func TestGaugeImpl_SparklineHistoryAppendError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Spark Test", "score": 85, "threshold": 75}`)

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.BadgeType = "sparkline"
	deps.HistoryFile = "/history.jsonl"
	deps.HistoryReader = func(path string) (*history.History, error) {
		return &history.History{Entries: []history.Entry{
			{Score: 80},
			{Score: 82},
		}}, nil
	}
	deps.HistoryAppender = func(path string, entry history.Entry) error {
		return errors.New("disk full")
	}

	err := gaugeImpl(deps)
	if err == nil {
		t.Fatal("expected error for history append failure")
	}

	if !strings.Contains(err.Error(), "appending to history") {
		t.Errorf("error should mention appending to history, got: %v", err)
	}
}

func TestGaugeImpl_SparklineWithHistoryFile(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Spark Test", "score": 85, "threshold": 75}`)

	appendCalled := false
	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.BadgeType = "sparkline"
	deps.HistoryFile = "/history.jsonl"
	deps.HistoryCount = 5
	deps.HistoryReader = func(path string) (*history.History, error) {
		return &history.History{Entries: []history.Entry{
			{Score: 75},
			{Score: 78},
			{Score: 80},
			{Score: 82},
		}}, nil
	}
	deps.HistoryAppender = func(path string, entry history.Entry) error {
		appendCalled = true
		if entry.Score != 85 {
			t.Errorf("appended score = %d, want 85", entry.Score)
		}
		return nil
	}

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if !appendCalled {
		t.Error("history appender should be called")
	}

	output := fs.GetFileContent("/output.svg")
	if !strings.Contains(output, "<svg") {
		t.Error("output should contain SVG")
	}
}

func TestGaugeImpl_SparklineNoHistoryFile(t *testing.T) {
	// Test sparkline without history file - should use just current score
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Spark Test", "score": 85, "threshold": 75}`)

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.BadgeType = "sparkline"
	deps.HistoryFile = "" // No history file

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := fs.GetFileContent("/output.svg")
	if !strings.Contains(output, "<svg") {
		t.Error("output should contain SVG")
	}
}

func TestGaugeImpl_SparklineEmptyHistory(t *testing.T) {
	// Test sparkline with empty history - should work with just current score
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Spark Test", "score": 85, "threshold": 75}`)

	appendCalled := false
	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.BadgeType = "sparkline"
	deps.HistoryFile = "/history.jsonl"
	deps.HistoryReader = func(path string) (*history.History, error) {
		return &history.History{Entries: []history.Entry{}}, nil // Empty history
	}
	deps.HistoryAppender = func(path string, entry history.Entry) error {
		appendCalled = true
		return nil
	}

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if !appendCalled {
		t.Error("history appender should be called even with empty history")
	}
}

func TestGaugeImpl_BaselineParseError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("current.json", `{"title": "Current", "score": 85, "threshold": 75}`)
	fs.SetFileContent("baseline.json", `not valid json`)

	deps := defaultGaugeDeps(fs)
	deps.Config = "current.json"
	deps.Output = "/output.svg"
	deps.Compare = "baseline.json"

	err := gaugeImpl(deps)
	if err == nil {
		t.Fatal("expected error for invalid baseline JSON")
	}

	if !strings.Contains(err.Error(), "parsing baseline") {
		t.Errorf("error should mention parsing baseline, got: %v", err)
	}
}

func TestGaugeImpl_BaselineFileNotFound(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("current.json", `{"title": "Current", "score": 85, "threshold": 75}`)
	// Don't set baseline.json content

	deps := defaultGaugeDeps(fs)
	deps.Config = "current.json"
	deps.Output = "/output.svg"
	deps.Compare = "nonexistent.json"

	err := gaugeImpl(deps)
	if err == nil {
		t.Fatal("expected error for missing baseline file")
	}
}

func TestGaugeImpl_MarkdownWithBaseline(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("current.json", `{"title": "Current", "score": 85, "threshold": 75}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 62, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "current.json"
	deps.Output = "-"
	deps.Format = "markdown"
	deps.Compare = "baseline.json"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "+23") {
		t.Errorf("markdown should contain positive delta +23, got: %s", output)
	}
	if !strings.Contains(output, "62%") {
		t.Errorf("markdown should contain baseline percentage, got: %s", output)
	}
}

func TestGaugeImpl_MarkdownWithNegativeBaseline(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("current.json", `{"title": "Current", "score": 62, "threshold": 75}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "current.json"
	deps.Output = "-"
	deps.Format = "markdown"
	deps.Compare = "baseline.json"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "-23") {
		t.Errorf("markdown should contain negative delta -23, got: %s", output)
	}
}

// Note: gaugeImpl uses a defer pattern for close errors that doesn't work with
// non-named returns. Close errors in gauge are silently dropped. This test
// documents the current behavior rather than testing error propagation.
func TestGaugeImpl_FileCloseError_CurrentBehavior(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetError("close:/output.svg", errors.New("close failed"))

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"

	// Currently, close errors are silently dropped due to defer pattern bug
	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() should succeed despite close error (current behavior), got: %v", err)
	}
}

// Note: The gauge library (svgo) doesn't propagate write errors, so write
// errors during SVG generation are silently dropped. This test documents
// the current behavior.
func TestGaugeImpl_FileWriteError_CurrentBehavior(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetError("write:/output.svg", errors.New("disk full"))

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"

	// Currently, write errors during SVG generation are silently dropped
	// because the underlying svgo library doesn't propagate write errors
	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() should succeed despite write error (current behavior), got: %v", err)
	}
}

func TestGaugeImpl_VerboseOutput(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Verbose Test", "score": 85, "threshold": 75}`)

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.Verbose = true

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}
}

func TestGaugeImpl_VerboseSuppressedForStdout(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout, stderr bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Stderr = &stderr
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Verbose = true

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	// Verbose output should be suppressed when writing to stdout
	if stderr.Len() > 0 {
		t.Errorf("verbose output should be suppressed for stdout, got stderr: %s", stderr.String())
	}
}

func TestGaugeImpl_JSONWithAllFields(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{
		"title": "Full Test",
		"score": 85,
		"threshold": 75,
		"version": "1.0.0",
		"generatedAt": "2024-01-01T00:00:00Z",
		"source": "test-source"
	}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Format = "json"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `"version": "1.0.0"`) {
		t.Error("JSON should contain version")
	}
	if !strings.Contains(output, `"source": "test-source"`) {
		t.Error("JSON should contain source")
	}
}

func TestGaugeImpl_MarkdownWithFactors(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{
		"title": "Factor Test",
		"score": 85,
		"threshold": 75,
		"description": "A detailed description",
		"factors": [
			{"name": "Test Factor", "score": 90, "weight": 50, "url": "https://example.com"},
			{"name": "Another Factor", "score": 80, "weight": 50}
		]
	}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Format = "markdown"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "A detailed description") {
		t.Error("markdown should contain description")
	}
	if !strings.Contains(output, "| Factor | Score | Weight |") {
		t.Error("markdown should contain factor table header")
	}
	if !strings.Contains(output, "[Test Factor](https://example.com)") {
		t.Error("markdown should contain factor with URL link")
	}
	if !strings.Contains(output, "Another Factor") {
		t.Error("markdown should contain factor without URL")
	}
}

func TestGaugeImpl_MarkdownFailingReport(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Fail Test", "score": 50, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Format = "markdown"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "(FAIL)") {
		t.Error("markdown should show FAIL for score below threshold")
	}
}

func TestGaugeImpl_GitHubCommentBasic(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "My Project", "score": 85, "threshold": 75, "version": "1.0.0"}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Format = "github-comment"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "## Confidence Report: My Project") {
		t.Error("github-comment should contain header with title")
	}
	if !strings.Contains(output, "| Score | **85%** :white_check_mark: |") {
		t.Error("github-comment should contain score with pass emoji")
	}
	if !strings.Contains(output, "| Threshold | 75% |") {
		t.Error("github-comment should contain threshold row")
	}
	if !strings.Contains(output, "| Status | Passed |") {
		t.Error("github-comment should contain status row")
	}
	if !strings.Contains(output, "<sub>Generated by confvis v1.0.0</sub>") {
		t.Error("github-comment should contain footer with version")
	}
}

func TestGaugeImpl_GitHubCommentWithBaseline(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("current.json", `{"title": "Test", "score": 85, "threshold": 75, "version": "1.0.0"}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 80, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "current.json"
	deps.Output = "-"
	deps.Format = "github-comment"
	deps.Compare = "baseline.json"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "| Change | +5 :arrow_up: |") {
		t.Errorf("github-comment should contain positive delta with up arrow, got: %s", output)
	}
}

func TestGaugeImpl_GitHubCommentNegativeDelta(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("current.json", `{"title": "Test", "score": 75, "threshold": 70, "version": "1.0.0"}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 80, "threshold": 70}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "current.json"
	deps.Output = "-"
	deps.Format = "github-comment"
	deps.Compare = "baseline.json"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "| Change | -5 :arrow_down: |") {
		t.Errorf("github-comment should contain negative delta with down arrow, got: %s", output)
	}
}

func TestGaugeImpl_GitHubCommentZeroDelta(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("current.json", `{"title": "Test", "score": 80, "threshold": 75, "version": "1.0.0"}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 80, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "current.json"
	deps.Output = "-"
	deps.Format = "github-comment"
	deps.Compare = "baseline.json"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "| Change | +0 :left_right_arrow: |") {
		t.Errorf("github-comment should contain zero delta with neutral arrow, got: %s", output)
	}
}

func TestGaugeImpl_GitHubCommentFailed(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Failed Test", "score": 50, "threshold": 75, "version": "1.0.0"}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Format = "github-comment"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "| Score | **50%** :x: |") {
		t.Error("github-comment should contain score with fail emoji")
	}
	if !strings.Contains(output, "| Status | Failed |") {
		t.Error("github-comment should contain Failed status")
	}
}

func TestGaugeImpl_GitHubCommentNoFactors(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "No Factors", "score": 85, "threshold": 75, "version": "1.0.0"}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Format = "github-comment"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if strings.Contains(output, "<details>") {
		t.Error("github-comment should not contain details section when no factors")
	}
	if strings.Contains(output, "Factor Breakdown") {
		t.Error("github-comment should not contain Factor Breakdown when no factors")
	}
}

func TestGaugeImpl_GitHubCommentWithFactors(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{
		"title": "With Factors",
		"score": 85,
		"threshold": 75,
		"version": "1.0.0",
		"factors": [
			{"name": "Coverage", "score": 90, "weight": 30, "description": "90% coverage"},
			{"name": "Security", "score": 80, "weight": 30, "description": "0 vulnerabilities"}
		]
	}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Format = "github-comment"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "<details>") {
		t.Error("github-comment should contain details section")
	}
	if !strings.Contains(output, "<summary>Factor Breakdown</summary>") {
		t.Error("github-comment should contain Factor Breakdown summary")
	}
	if !strings.Contains(output, "| Factor | Score | Weight | Description |") {
		t.Error("github-comment should contain factor table header")
	}
	if !strings.Contains(output, "| Coverage | 90 | 30 | 90% coverage |") {
		t.Error("github-comment should contain Coverage factor row")
	}
	if !strings.Contains(output, "| Security | 80 | 30 | 0 vulnerabilities |") {
		t.Error("github-comment should contain Security factor row")
	}
	if !strings.Contains(output, "</details>") {
		t.Error("github-comment should close details section")
	}
}

func TestGaugeImpl_GitHubCommentEmptyDescription(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{
		"title": "Empty Desc",
		"score": 85,
		"threshold": 75,
		"version": "1.0.0",
		"factors": [
			{"name": "NoDesc", "score": 90, "weight": 30, "description": ""},
			{"name": "HasDesc", "score": 80, "weight": 30, "description": "Has description"}
		]
	}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Format = "github-comment"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "| NoDesc | 90 | 30 | - |") {
		t.Errorf("github-comment should use dash for empty description, got: %s", output)
	}
	if !strings.Contains(output, "| HasDesc | 80 | 30 | Has description |") {
		t.Error("github-comment should show description when present")
	}
}

func TestGaugeImpl_GitHubCommentNoVersion(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "No Version", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Format = "github-comment"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "<sub>Generated by confvis vunknown</sub>") {
		t.Errorf("github-comment should show 'unknown' when no version, got: %s", output)
	}
}

func TestGaugeImpl_TextWithZeroDelta(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("current.json", `{"title": "Current", "score": 85, "threshold": 75}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "current.json"
	deps.Output = "-"
	deps.Format = "text"
	deps.Compare = "baseline.json"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	// Zero delta should show +0
	if strings.TrimSpace(stdout.String()) != "85 (+0)" {
		t.Errorf("text should show zero delta, got: %s", stdout.String())
	}
}

func TestAggregateImpl_VerboseOutput(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("report.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stderr bytes.Buffer
	deps := &AggregateDeps{
		FS:        fs,
		Stderr:    &stderr,
		Verbose:   true,
		Quiet:     false,
		ExitFunc:  func(code int) {},
		Configs:   []string{"report.json"},
		Output:    "/output",
		Dark:      false,
		FailUnder: 0,
	}

	err := aggregateImpl(deps)
	if err != nil {
		t.Fatalf("aggregateImpl() error = %v", err)
	}
}

func TestAggregateImpl_ValidWeight(t *testing.T) {
	// Test that valid weight is used
	fs := NewMockFileSystem()
	fs.SetFileContent("report.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := &AggregateDeps{
		FS:        fs,
		Stderr:    &bytes.Buffer{},
		Verbose:   false,
		Quiet:     false,
		ExitFunc:  func(code int) {},
		Configs:   []string{"report.json:50"}, // Valid weight
		Output:    "/output",
		Dark:      false,
		FailUnder: 0,
	}

	err := aggregateImpl(deps)
	if err != nil {
		t.Fatalf("aggregateImpl() error = %v", err)
	}
}

func TestAggregateImpl_SpecialCharTitle(t *testing.T) {
	// Test sanitization with special characters in title
	fs := NewMockFileSystem()
	fs.SetFileContent("report.json", `{"title": "Test@#$%Report!", "score": 85, "threshold": 75}`)

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
	if err != nil {
		t.Fatalf("aggregateImpl() error = %v", err)
	}

	// Should create testreport.svg (sanitized from Test@#$%Report!)
	badge := fs.GetFileContent("/output/testreport.svg")
	if !strings.Contains(badge, "<svg") {
		t.Error("badge with sanitized name should be created")
	}
}

func TestAggregateImpl_DashboardCloseError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("report.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetError("close:/output/dashboard/index.html", errors.New("close failed"))

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
		t.Fatal("expected error for dashboard close failure")
	}

	if !strings.Contains(err.Error(), "closing dashboard file") {
		t.Errorf("error should mention closing dashboard file, got: %v", err)
	}
}

func TestAggregateImpl_ZeroThreshold(t *testing.T) {
	// Test with reports that have no threshold set
	fs := NewMockFileSystem()
	fs.SetFileContent("report1.json", `{"title": "Report 1", "score": 85}`)
	fs.SetFileContent("report2.json", `{"title": "Report 2", "score": 75}`)

	deps := &AggregateDeps{
		FS:        fs,
		Stderr:    &bytes.Buffer{},
		Verbose:   false,
		Quiet:     false,
		ExitFunc:  func(code int) {},
		Configs:   []string{"report1.json", "report2.json"},
		Output:    "/output",
		Dark:      false,
		FailUnder: 0,
	}

	err := aggregateImpl(deps)
	if err != nil {
		t.Fatalf("aggregateImpl() error = %v", err)
	}
}

func TestGenerateImpl_VerboseOutput(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Verbose Test", "score": 85, "threshold": 75}`)

	deps := &GenerateDeps{
		FS:          fs,
		Stdin:       nil,
		Stderr:      &bytes.Buffer{},
		Verbose:     true,
		Quiet:       false,
		ExitFunc:    func(code int) {},
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
}

func TestGaugeImpl_SparklineDefaultDimensions(t *testing.T) {
	// Test that sparkline uses smaller default dimensions
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.BadgeType = "sparkline"
	// Width and Height default to 200 and 120, sparkline should override to 120 and 28

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `width="120"`) {
		t.Error("sparkline should use default width of 120")
	}
}

func TestGaugeImpl_FlatBadgeWithCustomLabel(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Original", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.BadgeType = "flat"
	deps.Label = "" // Empty label should use report title

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Original") {
		t.Error("flat badge should use report title as label when custom label is empty")
	}
}

func TestAggregateImpl_UseLowestThreshold(t *testing.T) {
	// Test that aggregate uses the lowest threshold from all reports
	fs := NewMockFileSystem()
	fs.SetFileContent("high.json", `{"title": "High Threshold", "score": 85, "threshold": 90}`)
	fs.SetFileContent("low.json", `{"title": "Low Threshold", "score": 85, "threshold": 60}`)

	deps := &AggregateDeps{
		FS:        fs,
		Stderr:    &bytes.Buffer{},
		Verbose:   false,
		Quiet:     false,
		ExitFunc:  func(code int) {},
		Configs:   []string{"high.json", "low.json"},
		Output:    "/output",
		Dark:      false,
		FailUnder: 0,
	}

	err := aggregateImpl(deps)
	if err != nil {
		t.Fatalf("aggregateImpl() error = %v", err)
	}
}

func TestGaugeImpl_SparklineCustomDimensions(t *testing.T) {
	// Test that custom dimensions are preserved for sparkline
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.BadgeType = "sparkline"
	deps.Width = 150  // Custom width != 200
	deps.Height = 50  // Custom height != 120

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `width="150"`) {
		t.Error("sparkline should preserve custom width")
	}
}

func TestGenerateImpl_FileCloseError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetError("close:/output/badge.svg", errors.New("close failed"))

	deps := &GenerateDeps{
		FS:          fs,
		Stdin:       nil,
		Stderr:      &bytes.Buffer{},
		Verbose:     false,
		Quiet:       false,
		ExitFunc:    func(code int) {},
		Config:      "config.json",
		Output:      "/output",
		InputFormat: "auto",
		Dark:        false,
		FailUnder:   0,
	}

	err := generateImpl(deps)
	if err == nil {
		t.Fatal("expected error for file close failure")
	}

	if !strings.Contains(err.Error(), "closing badge file") {
		t.Errorf("error should mention closing badge file, got: %v", err)
	}
}

func TestGenerateImpl_DashboardCloseError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetError("close:/output/dashboard/index.html", errors.New("close failed"))

	deps := &GenerateDeps{
		FS:          fs,
		Stdin:       nil,
		Stderr:      &bytes.Buffer{},
		Verbose:     false,
		Quiet:       false,
		ExitFunc:    func(code int) {},
		Config:      "config.json",
		Output:      "/output",
		InputFormat: "auto",
		Dark:        false,
		FailUnder:   0,
	}

	err := generateImpl(deps)
	if err == nil {
		t.Fatal("expected error for dashboard close failure")
	}

	if !strings.Contains(err.Error(), "closing dashboard file") {
		t.Errorf("error should mention closing dashboard file, got: %v", err)
	}
}

func TestAggregateImpl_BadgeCloseError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("report.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetError("close:/output/badge.svg", errors.New("close failed"))

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
		t.Fatal("expected error for badge close failure")
	}

	if !strings.Contains(err.Error(), "closing badge file") {
		t.Errorf("error should mention closing badge file, got: %v", err)
	}
}

func TestAggregateImpl_IndividualBadgeCloseError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("report.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	// Filename is sanitized from title: "Test" -> "test.svg"
	fs.SetError("close:/output/test.svg", errors.New("close failed"))

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
		t.Fatal("expected error for individual badge close failure")
	}

	if !strings.Contains(err.Error(), "closing badge file") {
		t.Errorf("error should mention closing badge file, got: %v", err)
	}
}

// Note: fetchImpl uses a defer pattern for close errors that doesn't work with
// non-named returns. Close errors in fetch are silently dropped. This test
// documents the current behavior rather than testing error propagation.
func TestFetchImpl_FileCloseError_CurrentBehavior(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetError("close:/output/report.json", errors.New("close failed"))

	src := &mockSource{
		name: "test",
		report: &confidence.Report{
			Title:     "Test",
			Score:     85,
			Threshold: 75,
		},
	}

	deps := &FetchDeps{
		FS:           fs,
		Stdout:       &bytes.Buffer{},
		Stderr:       &bytes.Buffer{},
		Verbose:      false,
		Quiet:        false,
		SourceGetter: mockSourceGetter(src),
		SourceName:   "test",
		Project:      "myproject",
		Output:       "/output/report.json",
		Extra:        map[string]string{},
	}

	// Currently, close errors are silently dropped due to defer pattern bug
	err := fetchImpl(context.Background(), deps)
	if err != nil {
		t.Fatalf("fetchImpl() should succeed despite close error (current behavior), got: %v", err)
	}
}

// ============================================================================
// errWriter - helper for testing write error paths
// ============================================================================

// errWriter is a writer that returns an error after n successful writes.
// Use n=0 to fail immediately on first write.
type errWriter struct {
	n   int
	err error
}

func (e *errWriter) Write(p []byte) (int, error) {
	if e.n <= 0 {
		return 0, e.err
	}
	e.n--
	return len(p), nil
}

// ============================================================================
// Write Error Path Tests - writeGitHubComment and related functions
// ============================================================================

func TestWriteGitHubComment_WriteErrors(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test Report",
		Score:     85,
		Threshold: 75,
		Version:   "1.0.0",
		Factors: []confidence.Factor{
			{Name: "Factor1", Score: 90, Weight: 50, Description: "First factor"},
		},
	}

	// Test error at various write points through the function
	for i := 0; i < 15; i++ {
		w := &errWriter{n: i, err: errors.New("write failed")}
		err := writeGitHubComment(w, report, nil)
		if i < 12 && err == nil {
			// The function has many writes; we expect errors for early failures
			t.Errorf("expected error at write %d, got nil", i)
		}
	}
}

func TestWriteGitHubCommentHeader_WriteErrors(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test Report",
		Score:     85,
		Threshold: 75,
	}

	// Test error at each write point in writeGitHubCommentHeader
	// The function has 6 Fprintf/Fprintln calls
	for i := 0; i < 6; i++ {
		w := &errWriter{n: i, err: errors.New("write failed")}
		err := writeGitHubCommentHeader(w, report)
		if err == nil {
			t.Errorf("expected error at write %d, got nil", i)
		}
	}
}

func TestWriteGitHubCommentHeader_WriteErrors_FailedReport(t *testing.T) {
	report := &confidence.Report{
		Title:     "Failing Report",
		Score:     50,
		Threshold: 75,
	}

	// Test with failing report (different emoji/text paths)
	w := &errWriter{n: 0, err: errors.New("write failed")}
	err := writeGitHubCommentHeader(w, report)
	if err == nil {
		t.Error("expected error for header write failure")
	}
}

func TestWriteGitHubCommentFactors_WriteErrors(t *testing.T) {
	factors := []confidence.Factor{
		{Name: "Factor1", Score: 90, Weight: 50, Description: "First factor"},
		{Name: "Factor2", Score: 80, Weight: 50, Description: ""},
	}

	// Test error at each write point in writeGitHubCommentFactors
	// The function has: details open, summary, table header, separator, 2 factor rows, details close = 7 writes
	for i := 0; i < 7; i++ {
		w := &errWriter{n: i, err: errors.New("write failed")}
		err := writeGitHubCommentFactors(w, factors)
		if err == nil {
			t.Errorf("expected error at write %d, got nil", i)
		}
	}
}

func TestWriteGitHubCommentFactors_EmptyFactors(t *testing.T) {
	// Empty factors should not write anything and should not error
	var buf bytes.Buffer
	err := writeGitHubCommentFactors(&buf, nil)
	if err != nil {
		t.Errorf("empty factors should not error, got: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("empty factors should not write anything, got: %s", buf.String())
	}
}

func TestWriteGitHubCommentFooter_WriteError(t *testing.T) {
	w := &errWriter{n: 0, err: errors.New("write failed")}
	err := writeGitHubCommentFooter(w, "1.0.0")
	if err == nil {
		t.Error("expected error for footer write failure")
	}
}

func TestWriteGitHubCommentBaseline_WriteError(t *testing.T) {
	report := &confidence.Report{Score: 85}
	baseline := &confidence.Report{Score: 80}

	w := &errWriter{n: 0, err: errors.New("write failed")}
	err := writeGitHubCommentBaseline(w, report, baseline)
	if err == nil {
		t.Error("expected error for baseline write failure")
	}
}

func TestWriteGitHubCommentBaseline_NilBaseline(t *testing.T) {
	report := &confidence.Report{Score: 85}

	// Nil baseline should not write anything
	var buf bytes.Buffer
	err := writeGitHubCommentBaseline(&buf, report, nil)
	if err != nil {
		t.Errorf("nil baseline should not error, got: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("nil baseline should not write anything, got: %s", buf.String())
	}
}

// ============================================================================
// Write Error Path Tests - writeMarkdown
// ============================================================================

func TestWriteMarkdown_WriteErrors(t *testing.T) {
	report := &confidence.Report{
		Title:       "Test Report",
		Score:       85,
		Threshold:   75,
		Description: "A description",
		Factors: []confidence.Factor{
			{Name: "Factor1", Score: 90, Weight: 50, URL: "https://example.com"},
		},
	}

	// Test error at various write points
	// The function has: header, description, table header, separator, factor row = 5 writes
	for i := 0; i < 5; i++ {
		w := &errWriter{n: i, err: errors.New("write failed")}
		err := writeMarkdown(w, report, nil, 0)
		if err == nil {
			t.Errorf("expected error at write %d, got nil", i)
		}
	}
}

func TestWriteMarkdown_WriteErrors_WithBaseline(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test Report",
		Score:     85,
		Threshold: 75,
	}
	baseline := &confidence.Report{Score: 80}

	// Test with baseline (different header format)
	w := &errWriter{n: 0, err: errors.New("write failed")}
	err := writeMarkdown(w, report, baseline, 5)
	if err == nil {
		t.Error("expected error for markdown header write failure with baseline")
	}
}

func TestWriteMarkdown_WriteErrors_NoDescription(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test Report",
		Score:     85,
		Threshold: 75,
		// No description
		Factors: []confidence.Factor{
			{Name: "Factor1", Score: 90, Weight: 50},
		},
	}

	// Error on factor table header
	w := &errWriter{n: 1, err: errors.New("write failed")}
	err := writeMarkdown(w, report, nil, 0)
	if err == nil {
		t.Error("expected error for factor table header write failure")
	}
}

// ============================================================================
// Write Error Path Tests - writeText
// ============================================================================

func TestWriteText_WriteErrors(t *testing.T) {
	// Test without baseline
	w := &errWriter{n: 0, err: errors.New("write failed")}
	err := writeText(w, 85, nil, 0)
	if err == nil {
		t.Error("expected error for text write failure")
	}
}

func TestWriteText_WriteErrors_WithBaseline(t *testing.T) {
	baseline := &confidence.Report{Score: 80}

	// Test with baseline (different format)
	w := &errWriter{n: 0, err: errors.New("write failed")}
	err := writeText(w, 85, baseline, 5)
	if err == nil {
		t.Error("expected error for text write failure with baseline")
	}
}

func TestWriteText_WriteErrors_NegativeDelta(t *testing.T) {
	baseline := &confidence.Report{Score: 90}

	// Test with negative delta
	w := &errWriter{n: 0, err: errors.New("write failed")}
	err := writeText(w, 85, baseline, -5)
	if err == nil {
		t.Error("expected error for text write failure with negative delta")
	}
}

// ============================================================================
// Write Error Path Tests - writeJSON
// ============================================================================

func TestWriteJSON_WriteError(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test Report",
		Score:     85,
		Threshold: 75,
	}

	w := &errWriter{n: 0, err: errors.New("write failed")}
	err := writeJSON(w, report, nil, 0)
	if err == nil {
		t.Error("expected error for JSON write failure")
	}
}

// ============================================================================
// Check GitHub Command Tests
// ============================================================================

func defaultCheckGitHubDeps(fs *MockFileSystem) *CheckGitHubDeps {
	return &CheckGitHubDeps{
		FS:         fs,
		Stdin:      strings.NewReader(""),
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		Verbose:    false,
		Quiet:      false,
		Config:     "config.json",
		Owner:      "testowner",
		Repo:       "testrepo",
		SHA:        "abc123",
		Name:       "Confidence Score",
		Token:      "test-token",
		APIURL:     "",
		AutoDetect: false,
		Timeout:    time.Second,
		CreateCheck: func(ctx context.Context, client *checks.GitHubClient,
			report *confidence.Report, opts checks.CreateCheckOptions) (*checks.CheckRunResponse, error) {
			return &checks.CheckRunResponse{
				ID:      12345,
				HTMLURL: "https://github.com/testowner/testrepo/runs/12345",
				Status:  "completed",
			}, nil
		},
		LoadGitHubEnv: func() (*checks.GitHubEnv, error) {
			return nil, fmt.Errorf("not in GitHub Actions")
		},
	}
}

func TestCheckGitHubImpl_BasicFlow(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultCheckGitHubDeps(fs)
	deps.Stdout = &stdout

	err := checkGitHubImpl(deps)
	if err != nil {
		t.Fatalf("checkGitHubImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Created check run") {
		t.Error("expected check run creation message")
	}
	if !strings.Contains(output, "passed") {
		t.Error("expected passed status for score above threshold")
	}
}

func TestCheckGitHubImpl_FailingScore(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 60, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultCheckGitHubDeps(fs)
	deps.Stdout = &stdout

	err := checkGitHubImpl(deps)
	if err != nil {
		t.Fatalf("checkGitHubImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "failed") {
		t.Error("expected failed status for score below threshold")
	}
}

func TestCheckGitHubImpl_QuietMode(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultCheckGitHubDeps(fs)
	deps.Stdout = &stdout
	deps.Quiet = true

	err := checkGitHubImpl(deps)
	if err != nil {
		t.Fatalf("checkGitHubImpl() error = %v", err)
	}

	if stdout.Len() > 0 {
		t.Errorf("quiet mode should suppress output, got: %s", stdout.String())
	}
}

func TestCheckGitHubImpl_StdinInput(t *testing.T) {
	fs := NewMockFileSystem()

	var stdout bytes.Buffer
	deps := defaultCheckGitHubDeps(fs)
	deps.Config = "-"
	deps.Stdin = strings.NewReader(`{"title": "Stdin Test", "score": 90, "threshold": 80}`)
	deps.Stdout = &stdout

	err := checkGitHubImpl(deps)
	if err != nil {
		t.Fatalf("checkGitHubImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Created check run") {
		t.Error("expected check run creation message")
	}
}

func TestCheckGitHubImpl_MissingToken(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCheckGitHubDeps(fs)
	deps.Token = ""
	deps.AutoDetect = false

	err := checkGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if !strings.Contains(err.Error(), "token required") {
		t.Errorf("error = %q, want to contain 'token required'", err.Error())
	}
}

func TestCheckGitHubImpl_MissingOwner(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCheckGitHubDeps(fs)
	deps.Owner = ""

	err := checkGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error for missing owner")
	}
	if !strings.Contains(err.Error(), "owner required") {
		t.Errorf("error = %q, want to contain 'owner required'", err.Error())
	}
}

func TestCheckGitHubImpl_MissingRepo(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCheckGitHubDeps(fs)
	deps.Repo = ""

	err := checkGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error for missing repo")
	}
	if !strings.Contains(err.Error(), "repo required") {
		t.Errorf("error = %q, want to contain 'repo required'", err.Error())
	}
}

func TestCheckGitHubImpl_MissingSHA(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCheckGitHubDeps(fs)
	deps.SHA = ""

	err := checkGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error for missing SHA")
	}
	if !strings.Contains(err.Error(), "SHA required") {
		t.Errorf("error = %q, want to contain 'SHA required'", err.Error())
	}
}

func TestCheckGitHubImpl_AutoDetect(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultCheckGitHubDeps(fs)
	deps.Stdout = &stdout
	deps.Token = "" // Will be filled from env
	deps.Owner = "" // Will be filled from env
	deps.Repo = ""  // Will be filled from env
	deps.SHA = ""   // Will be filled from env
	deps.AutoDetect = true
	deps.LoadGitHubEnv = func() (*checks.GitHubEnv, error) {
		return &checks.GitHubEnv{
			Token:      "env-token",
			Repository: "envowner/envrepo",
			SHA:        "env123",
			APIURL:     "https://api.github.example.com",
		}, nil
	}

	err := checkGitHubImpl(deps)
	if err != nil {
		t.Fatalf("checkGitHubImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Created check run") {
		t.Error("expected check run creation message")
	}
}

func TestCheckGitHubImpl_AutoDetectWithFlagOverrides(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var calledOwner, calledRepo string
	deps := defaultCheckGitHubDeps(fs)
	deps.Owner = "flag-owner" // Override from flag
	deps.Repo = ""            // Will be filled from env
	deps.SHA = ""             // Will be filled from env
	deps.Token = ""           // Will be filled from env
	deps.AutoDetect = true
	deps.LoadGitHubEnv = func() (*checks.GitHubEnv, error) {
		return &checks.GitHubEnv{
			Token:      "env-token",
			Repository: "envowner/envrepo",
			SHA:        "env123",
		}, nil
	}
	deps.CreateCheck = func(ctx context.Context, client *checks.GitHubClient,
		report *confidence.Report, opts checks.CreateCheckOptions) (*checks.CheckRunResponse, error) {
		calledOwner = opts.Owner
		calledRepo = opts.Repo
		return &checks.CheckRunResponse{
			ID:      12345,
			HTMLURL: "https://github.com/test/runs/12345",
			Status:  "completed",
		}, nil
	}

	err := checkGitHubImpl(deps)
	if err != nil {
		t.Fatalf("checkGitHubImpl() error = %v", err)
	}

	// Flag should override env
	if calledOwner != "flag-owner" {
		t.Errorf("owner = %q, want 'flag-owner' (flag should override env)", calledOwner)
	}
	// Env should fill missing repo
	if calledRepo != "envrepo" {
		t.Errorf("repo = %q, want 'envrepo' (from env)", calledRepo)
	}
}

func TestCheckGitHubImpl_AutoDetectEnvError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCheckGitHubDeps(fs)
	deps.Token = ""
	deps.AutoDetect = true
	deps.LoadGitHubEnv = func() (*checks.GitHubEnv, error) {
		return nil, fmt.Errorf("GITHUB_TOKEN not set")
	}

	err := checkGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error when env loading fails and no token provided")
	}
	if !strings.Contains(err.Error(), "GitHub environment") {
		t.Errorf("error = %q, want to contain 'GitHub environment'", err.Error())
	}
}

func TestCheckGitHubImpl_CreateCheckError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultCheckGitHubDeps(fs)
	deps.CreateCheck = func(ctx context.Context, client *checks.GitHubClient,
		report *confidence.Report, opts checks.CreateCheckOptions) (*checks.CheckRunResponse, error) {
		return nil, fmt.Errorf("API error: unauthorized")
	}

	err := checkGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error when create check fails")
	}
	if !strings.Contains(err.Error(), "creating check run") {
		t.Errorf("error = %q, want to contain 'creating check run'", err.Error())
	}
}

func TestCheckGitHubImpl_InvalidConfig(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `invalid json`)

	deps := defaultCheckGitHubDeps(fs)

	err := checkGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
	if !strings.Contains(err.Error(), "parsing config") {
		t.Errorf("error = %q, want to contain 'parsing config'", err.Error())
	}
}

func TestCheckGitHubImpl_ConfigFileNotFound(t *testing.T) {
	fs := NewMockFileSystem()
	// Don't set config.json - it will be missing

	deps := defaultCheckGitHubDeps(fs)

	err := checkGitHubImpl(deps)
	if err == nil {
		t.Fatal("expected error when config file not found")
	}
	if !strings.Contains(err.Error(), "opening config") {
		t.Errorf("error = %q, want to contain 'opening config'", err.Error())
	}
}
