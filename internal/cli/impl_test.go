package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/boinger/confvis/internal/confidence"
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
