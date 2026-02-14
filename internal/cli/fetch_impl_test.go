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
			Score:     intPtrH(85),
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
			Score:     intPtrH(90),
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
			Score:     intPtrH(85),
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
			Score:     intPtrH(85),
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
			Score:     intPtrH(85),
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
			Score:     intPtrH(85),
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
			Score:     intPtrH(60),
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
			Score:     intPtrH(85),
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

