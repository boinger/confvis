package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/boinger/confvis/internal/baseline"
	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/history"
)

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

	if !strings.Contains(err.Error(), "loading baseline") {
		t.Errorf("error should mention loading baseline, got: %v", err)
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

// Close errors in gauge output are now properly propagated via named return.
func TestGaugeImpl_FileCloseError_CurrentBehavior(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetError("close:/output.svg", errors.New("close failed"))

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"

	err := gaugeImpl(deps)
	if err == nil {
		t.Fatal("gaugeImpl() should return close error")
	}
	if !strings.Contains(err.Error(), "close") {
		t.Errorf("error should mention close, got: %v", err)
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

func TestGaugeImpl_EmitJSON_Success(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "EmitJSON Test", "score": 85, "threshold": 75}`)

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.EmitJSON = "/metadata.json"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	// Verify SVG was written
	svg := fs.GetFileContent("/output.svg")
	if !strings.Contains(svg, "<svg") {
		t.Error("SVG output should be written")
	}

	// Verify JSON metadata was written
	jsonOut := fs.GetFileContent("/metadata.json")
	if !strings.Contains(jsonOut, `"score": 85`) {
		t.Error("JSON metadata should contain score")
	}
	if !strings.Contains(jsonOut, `"passed": true`) {
		t.Error("JSON metadata should contain passed status")
	}
}

func TestGaugeImpl_EmitJSON_WithBaseline(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "EmitJSON Test", "score": 85, "threshold": 75}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 80, "threshold": 75}`)

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.EmitJSON = "/metadata.json"
	deps.Compare = "baseline.json"

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	// Verify JSON metadata includes delta
	jsonOut := fs.GetFileContent("/metadata.json")
	if !strings.Contains(jsonOut, `"delta": 5`) {
		t.Errorf("JSON metadata should contain delta, got: %s", jsonOut)
	}
	if !strings.Contains(jsonOut, `"baseline": 80`) {
		t.Errorf("JSON metadata should contain baseline, got: %s", jsonOut)
	}
}

func TestGaugeImpl_EmitJSON_CreateError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetError("create:/metadata.json", errors.New("permission denied"))

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.EmitJSON = "/metadata.json"

	err := gaugeImpl(deps)
	if err == nil {
		t.Fatal("expected error when emit-json file creation fails")
	}
	if !strings.Contains(err.Error(), "creating emit-json file") {
		t.Errorf("error should mention creating emit-json file, got: %v", err)
	}
}

func TestGaugeImpl_EmitJSON_CloseError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetError("close:/metadata.json", errors.New("disk full"))

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.EmitJSON = "/metadata.json"

	err := gaugeImpl(deps)
	if err == nil {
		t.Fatal("expected error when emit-json file close fails")
	}
	if !strings.Contains(err.Error(), "closing emit-json file") {
		t.Errorf("error should mention closing emit-json file, got: %v", err)
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

func TestAggregateImpl_EmitJSON_Success(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("report1.json", `{"title": "Report 1", "score": 90, "threshold": 75}`)
	fs.SetFileContent("report2.json", `{"title": "Report 2", "score": 80, "threshold": 75}`)

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
		EmitJSON:  "/output/aggregate.json",
	}

	err := aggregateImpl(deps)
	if err != nil {
		t.Fatalf("aggregateImpl() error = %v", err)
	}

	// Verify JSON was written
	jsonOut := fs.GetFileContent("/output/aggregate.json")
	if jsonOut == "" {
		t.Fatal("emit-json file should be created")
	}
	if !strings.Contains(jsonOut, `"score":`) {
		t.Error("JSON should contain score")
	}
	if !strings.Contains(jsonOut, `"passed":`) {
		t.Error("JSON should contain passed status")
	}
}

func TestAggregateImpl_EmitJSON_CreateError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("report.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetError("create:/output/aggregate.json", errors.New("permission denied"))

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
		EmitJSON:  "/output/aggregate.json",
	}

	err := aggregateImpl(deps)
	if err == nil {
		t.Fatal("expected error when emit-json file creation fails")
	}
	if !strings.Contains(err.Error(), "creating JSON file") {
		t.Errorf("error should mention creating JSON file, got: %v", err)
	}
}

func TestAggregateImpl_EmitJSON_CloseError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("report.json", `{"title": "Test", "score": 85, "threshold": 75}`)
	fs.SetError("close:/output/aggregate.json", errors.New("disk full"))

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
		EmitJSON:  "/output/aggregate.json",
	}

	err := aggregateImpl(deps)
	if err == nil {
		t.Fatal("expected error when emit-json file close fails")
	}
	if !strings.Contains(err.Error(), "closing JSON file") {
		t.Errorf("error should mention closing JSON file, got: %v", err)
	}
}

// Close errors in fetch output are now properly propagated via named return.
func TestFetchImpl_FileCloseError_CurrentBehavior(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetError("close:/output/report.json", errors.New("close failed"))

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
		t.Fatal("fetchImpl() should return close error")
	}
	if !strings.Contains(err.Error(), "close") {
		t.Errorf("error should mention close, got: %v", err)
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
		Score:     intPtrH(85),
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
		Score:     intPtrH(85),
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
		Score:     intPtrH(50),
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
	report := &confidence.Report{Score: intPtrH(85)}
	baseline := &confidence.Report{Score: intPtrH(80)}

	w := &errWriter{n: 0, err: errors.New("write failed")}
	err := writeGitHubCommentBaseline(w, report, baseline)
	if err == nil {
		t.Error("expected error for baseline write failure")
	}
}

func TestWriteGitHubCommentBaseline_NilBaseline(t *testing.T) {
	report := &confidence.Report{Score: intPtrH(85)}

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
		Score:       intPtrH(85),
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
		Score:     intPtrH(85),
		Threshold: 75,
	}
	baseline := &confidence.Report{Score: intPtrH(80)}

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
		Score:     intPtrH(85),
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
	baseline := &confidence.Report{Score: intPtrH(80)}

	// Test with baseline (different format)
	w := &errWriter{n: 0, err: errors.New("write failed")}
	err := writeText(w, 85, baseline, 5)
	if err == nil {
		t.Error("expected error for text write failure with baseline")
	}
}

func TestWriteText_WriteErrors_NegativeDelta(t *testing.T) {
	baseline := &confidence.Report{Score: intPtrH(90)}

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
		Score:     intPtrH(85),
		Threshold: 75,
	}

	w := &errWriter{n: 0, err: errors.New("write failed")}
	err := writeJSON(w, report, nil, 0)
	if err == nil {
		t.Error("expected error for JSON write failure")
	}
}


// ============================================================================
// resolveHistoryStorage Tests
// ============================================================================

func TestResolveHistoryStorage(t *testing.T) {
	tests := []struct {
		name         string
		deps         *GaugeDeps
		wantGitRef   bool
		wantRef      string
		wantFile     string
	}{
		{
			name: "explicit history-ref",
			deps: &GaugeDeps{HistoryRef: "refs/custom/history"},
			wantGitRef: true,
			wantRef:    "refs/custom/history",
			wantFile:   "",
		},
		{
			name: "history-auto in git repo",
			deps: &GaugeDeps{
				HistoryAuto: true,
				IsGitRepo:   func() bool { return true },
			},
			wantGitRef: true,
			wantRef:    history.DefaultHistoryRef,
			wantFile:   "",
		},
		{
			name: "history-auto not in git repo (default file)",
			deps: &GaugeDeps{
				HistoryAuto: true,
				IsGitRepo:   func() bool { return false },
			},
			wantGitRef: false,
			wantRef:    "",
			wantFile:   ".confvis-history.jsonl",
		},
		{
			name: "history-auto not in git repo with explicit history file",
			deps: &GaugeDeps{
				HistoryAuto: true,
				IsGitRepo:   func() bool { return false },
				HistoryFile: "/custom/history.jsonl",
			},
			wantGitRef: false,
			wantRef:    "",
			wantFile:   "/custom/history.jsonl",
		},
		{
			name: "history-auto nil IsGitRepo",
			deps: &GaugeDeps{
				HistoryAuto: true,
				IsGitRepo:   nil,
			},
			wantGitRef: false,
			wantRef:    "",
			wantFile:   ".confvis-history.jsonl",
		},
		{
			name:       "explicit history-file",
			deps:       &GaugeDeps{HistoryFile: "/my/history.jsonl"},
			wantGitRef: false,
			wantRef:    "",
			wantFile:   "/my/history.jsonl",
		},
		{
			name:       "no history storage",
			deps:       &GaugeDeps{},
			wantGitRef: false,
			wantRef:    "",
			wantFile:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotGitRef, gotRef, gotFile := resolveHistoryStorage(tt.deps)
			if gotGitRef != tt.wantGitRef {
				t.Errorf("useGitRef = %v, want %v", gotGitRef, tt.wantGitRef)
			}
			if gotRef != tt.wantRef {
				t.Errorf("historyRef = %q, want %q", gotRef, tt.wantRef)
			}
			if gotFile != tt.wantFile {
				t.Errorf("historyFile = %q, want %q", gotFile, tt.wantFile)
			}
		})
	}
}

// ============================================================================
// resolveBaseline Tests
// ============================================================================

func TestResolveBaseline(t *testing.T) {
	tests := []struct {
		name      string
		deps      *GaugeDeps
		wantNil   bool
		wantScore int
		wantErr   bool
	}{
		{
			name: "baseline from file",
			deps: &GaugeDeps{
				BaselineFile: "baseline.json",
				BaselineFileReader: func(path string) (*baseline.Baseline, error) {
					score := 80
					return &baseline.Baseline{Report: confidence.Report{Score: &score, Title: "B"}}, nil
				},
			},
			wantScore: 80,
		},
		{
			name: "baseline from file error",
			deps: &GaugeDeps{
				BaselineFile: "baseline.json",
				BaselineFileReader: func(path string) (*baseline.Baseline, error) {
					return nil, errors.New("file not found")
				},
			},
			wantErr: true,
		},
		{
			name: "baseline from git ref (default ref)",
			deps: &GaugeDeps{
				IsGitRepo: func() bool { return true },
				BaselineGitRefReader: func(ref string) (*baseline.Baseline, error) {
					score := 75
					return &baseline.Baseline{Report: confidence.Report{Score: &score, Title: "G"}}, nil
				},
			},
			wantScore: 75,
		},
		{
			name: "baseline from git ref (explicit ref)",
			deps: &GaugeDeps{
				IsGitRepo:  func() bool { return true },
				BaselineRef: "refs/custom/baseline",
				BaselineGitRefReader: func(ref string) (*baseline.Baseline, error) {
					if ref != "refs/custom/baseline" {
						return nil, fmt.Errorf("unexpected ref: %s", ref)
					}
					score := 90
					return &baseline.Baseline{Report: confidence.Report{Score: &score, Title: "C"}}, nil
				},
			},
			wantScore: 90,
		},
		{
			name: "baseline git ref error",
			deps: &GaugeDeps{
				IsGitRepo: func() bool { return true },
				BaselineGitRefReader: func(ref string) (*baseline.Baseline, error) {
					return nil, errors.New("ref not found")
				},
			},
			wantErr: true,
		},
		{
			name: "not in git repo and no file",
			deps: &GaugeDeps{
				IsGitRepo: func() bool { return false },
			},
			wantNil: true,
		},
		{
			name: "nil IsGitRepo and no file",
			deps: &GaugeDeps{
				IsGitRepo: nil,
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := resolveBaseline(tt.deps)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveBaseline() error = %v", err)
			}
			if tt.wantNil {
				if b != nil {
					t.Error("expected nil baseline")
				}
				return
			}
			if b == nil {
				t.Fatal("expected non-nil baseline")
			}
			if b.ScoreValue() != tt.wantScore {
				t.Errorf("score = %d, want %d", b.ScoreValue(), tt.wantScore)
			}
		})
	}
}

// ============================================================================
// loadBaselineForComparison Tests
// ============================================================================

func TestLoadBaselineForComparison_CompareBaseline(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	score := 70
	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.CompareBaseline = true
	deps.BaselineFile = "baseline.json"
	deps.BaselineFileReader = func(path string) (*baseline.Baseline, error) {
		return &baseline.Baseline{Report: confidence.Report{Score: &score, Title: "BL"}}, nil
	}

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}
}

func TestLoadBaselineForComparison_CompareBaseline_Error(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.CompareBaseline = true
	deps.BaselineFile = "baseline.json"
	deps.BaselineFileReader = func(path string) (*baseline.Baseline, error) {
		return nil, errors.New("corrupted baseline")
	}

	err := gaugeImpl(deps)
	if err == nil {
		t.Fatal("expected error for baseline load failure")
	}
	if !strings.Contains(err.Error(), "loading baseline") {
		t.Errorf("error should mention loading baseline, got: %v", err)
	}
}

func TestLoadBaselineForComparison_CompareBaseline_NilResult(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.CompareBaseline = true
	deps.IsGitRepo = func() bool { return false }
	// No baseline file, not in git repo → nil baseline, no error

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}
}

func TestLoadBaselineForComparison_CompareBaseline_JSONOutput(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	score := 70
	deps := defaultGaugeDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.Output = "-"
	deps.Format = "json"
	deps.CompareBaseline = true
	deps.BaselineFile = "baseline.json"
	deps.BaselineFileReader = func(path string) (*baseline.Baseline, error) {
		return &baseline.Baseline{Report: confidence.Report{Score: &score, Title: "BL"}}, nil
	}

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `"delta": 15`) {
		t.Errorf("JSON should contain delta 15 (85-70), got: %s", output)
	}
	if !strings.Contains(output, `"baseline": 70`) {
		t.Errorf("JSON should contain baseline 70, got: %s", output)
	}
}

// ============================================================================
// parseFactorThresholds / parseFactorThresholdSpec / checkFactorThresholds
// ============================================================================

func TestParseFactorThresholds(t *testing.T) {
	tests := []struct {
		name      string
		cli       []string
		config    map[string]int
		wantLen   int
		wantErr   bool
	}{
		{
			name:    "empty",
			cli:     nil,
			config:  nil,
			wantLen: 0,
		},
		{
			name:    "config only",
			cli:     nil,
			config:  map[string]int{"Coverage": 80, "Security": 90},
			wantLen: 2,
		},
		{
			name:    "CLI only",
			cli:     []string{"Coverage:80", "Security:90"},
			config:  nil,
			wantLen: 2,
		},
		{
			name:    "CLI overrides config",
			cli:     []string{"Coverage:95"},
			config:  map[string]int{"Coverage": 80},
			wantLen: 1,
		},
		{
			name:    "invalid CLI spec",
			cli:     []string{"nocolon"},
			config:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFactorThresholds(tt.cli, tt.config)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseFactorThresholds() error = %v", err)
			}
			if len(result) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(result), tt.wantLen)
			}
		})
	}

	// Test that CLI overrides config value
	result, err := parseFactorThresholds([]string{"Coverage:95"}, map[string]int{"Coverage": 80})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result["Coverage"] != 95 {
		t.Errorf("Coverage = %d, want 95 (CLI override)", result["Coverage"])
	}
}

func TestParseFactorThresholdSpec(t *testing.T) {
	tests := []struct {
		spec      string
		wantName  string
		wantThres int
		wantErr   bool
	}{
		{"Coverage:80", "Coverage", 80, false},
		{"Test Coverage:90", "Test Coverage", 90, false},
		{"Score:0", "Score", 0, false},
		{"Score:100", "Score", 100, false},
		{"nocolon", "", 0, true},
		{":80", "", 0, true},
		{"Name:abc", "", 0, true},
		{"Name:-1", "", 0, true},
		{"Name:101", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			name, threshold, err := parseFactorThresholdSpec(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			if threshold != tt.wantThres {
				t.Errorf("threshold = %d, want %d", threshold, tt.wantThres)
			}
		})
	}
}

func TestCheckFactorThresholds(t *testing.T) {
	tests := []struct {
		name       string
		factors    []confidence.Factor
		overrides  map[string]int
		wantPassed bool
		wantCount  int
	}{
		{
			name:       "no factors",
			factors:    nil,
			overrides:  nil,
			wantPassed: true,
			wantCount:  0,
		},
		{
			name: "all pass with overrides",
			factors: []confidence.Factor{
				{Name: "Coverage", Score: 85, Weight: 20},
				{Name: "Security", Score: 90, Weight: 20},
			},
			overrides:  map[string]int{"Coverage": 80, "Security": 85},
			wantPassed: true,
			wantCount:  0,
		},
		{
			name: "one fails with override",
			factors: []confidence.Factor{
				{Name: "Coverage", Score: 75, Weight: 20},
				{Name: "Security", Score: 90, Weight: 20},
			},
			overrides:  map[string]int{"Coverage": 80},
			wantPassed: false,
			wantCount:  1,
		},
		{
			name: "factor threshold used when no override",
			factors: []confidence.Factor{
				{Name: "Coverage", Score: 70, Weight: 20, Threshold: 80},
			},
			overrides:  nil,
			wantPassed: false,
			wantCount:  1,
		},
		{
			name: "override takes precedence over factor threshold",
			factors: []confidence.Factor{
				{Name: "Coverage", Score: 75, Weight: 20, Threshold: 80},
			},
			overrides:  map[string]int{"Coverage": 70},
			wantPassed: true,
			wantCount:  0,
		},
		{
			name: "no threshold set - always passes",
			factors: []confidence.Factor{
				{Name: "Coverage", Score: 10, Weight: 20},
			},
			overrides:  nil,
			wantPassed: true,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &confidence.Report{
				Title:   "Test",
				Score:   intPtrH(80),
				Factors: tt.factors,
			}
			passed, failures := checkFactorThresholds(report, tt.overrides)
			if passed != tt.wantPassed {
				t.Errorf("passed = %v, want %v", passed, tt.wantPassed)
			}
			if len(failures) != tt.wantCount {
				t.Errorf("failures count = %d, want %d: %v", len(failures), tt.wantCount, failures)
			}
		})
	}
}

// ============================================================================
// checkGaugeThresholds Tests
// ============================================================================

func TestCheckGaugeThresholds_FailUnderQuiet(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 50, "threshold": 75}`)

	var stderr bytes.Buffer
	exitCode := -1
	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.Stderr = &stderr
	deps.Quiet = true
	deps.FailUnder = 60
	deps.ExitFunc = func(code int) { exitCode = code }

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if stderr.Len() > 0 {
		t.Error("quiet mode should suppress stderr")
	}
}

func TestCheckGaugeThresholds_FailOnRegressionQuiet(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("current.json", `{"title": "Current", "score": 70, "threshold": 75}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 85, "threshold": 75}`)

	var stderr bytes.Buffer
	exitCode := -1
	deps := defaultGaugeDeps(fs)
	deps.Config = "current.json"
	deps.Output = "/output.svg"
	deps.Stderr = &stderr
	deps.Quiet = true
	deps.FailOnRegression = true
	deps.Compare = "baseline.json"
	deps.ExitFunc = func(code int) { exitCode = code }

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if stderr.Len() > 0 {
		t.Error("quiet mode should suppress regression message")
	}
}

func TestCheckGaugeThresholds_FactorThresholdFailure(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{
		"title": "Test",
		"score": 85,
		"threshold": 75,
		"factors": [
			{"name": "Coverage", "score": 70, "weight": 50},
			{"name": "Security", "score": 95, "weight": 50}
		]
	}`)

	var stderr bytes.Buffer
	exitCode := -1
	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.Stderr = &stderr
	deps.FactorThresholds = map[string]int{"Coverage": 80}
	deps.ExitFunc = func(code int) { exitCode = code }

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1 for factor threshold failure, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Factor threshold failed") {
		t.Errorf("stderr should mention factor threshold failure, got: %s", stderr.String())
	}
}

func TestCheckGaugeThresholds_FactorThresholdQuiet(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{
		"title": "Test",
		"score": 85,
		"threshold": 75,
		"factors": [
			{"name": "Coverage", "score": 70, "weight": 50}
		]
	}`)

	var stderr bytes.Buffer
	exitCode := -1
	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.Stderr = &stderr
	deps.Quiet = true
	deps.FactorThresholds = map[string]int{"Coverage": 80}
	deps.ExitFunc = func(code int) { exitCode = code }

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if stderr.Len() > 0 {
		t.Error("quiet mode should suppress factor threshold message")
	}
}

// ============================================================================
// Sparkline with Git Ref Tests
// ============================================================================

func TestGaugeImpl_SparklineWithGitRef(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	gitRefAppended := false
	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.BadgeType = "sparkline"
	deps.HistoryRef = "refs/confvis/history"
	deps.HistoryCount = 5
	deps.GitRefReader = func(ref string) (*history.History, error) {
		return &history.History{Entries: []history.Entry{
			{Score: 75},
			{Score: 80},
		}}, nil
	}
	deps.GitRefAppender = func(ref string, entry history.Entry) error {
		gitRefAppended = true
		if entry.Score != 85 {
			t.Errorf("appended score = %d, want 85", entry.Score)
		}
		return nil
	}

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if !gitRefAppended {
		t.Error("git ref appender should be called")
	}

	output := fs.GetFileContent("/output.svg")
	if !strings.Contains(output, "<svg") {
		t.Error("output should contain SVG")
	}
}

func TestGaugeImpl_SparklineGitRefReadError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.BadgeType = "sparkline"
	deps.HistoryRef = "refs/confvis/history"
	deps.GitRefReader = func(ref string) (*history.History, error) {
		return nil, errors.New("git ref not found")
	}

	err := gaugeImpl(deps)
	if err == nil {
		t.Fatal("expected error for git ref read failure")
	}
	if !strings.Contains(err.Error(), "reading history from git ref") {
		t.Errorf("error should mention git ref, got: %v", err)
	}
}

func TestGaugeImpl_SparklineGitRefAppendError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.BadgeType = "sparkline"
	deps.HistoryRef = "refs/confvis/history"
	deps.GitRefReader = func(ref string) (*history.History, error) {
		return &history.History{Entries: []history.Entry{{Score: 80}}}, nil
	}
	deps.GitRefAppender = func(ref string, entry history.Entry) error {
		return errors.New("permission denied")
	}

	err := gaugeImpl(deps)
	if err == nil {
		t.Fatal("expected error for git ref append failure")
	}
	if !strings.Contains(err.Error(), "appending to history git ref") {
		t.Errorf("error should mention git ref append, got: %v", err)
	}
}

func TestGaugeImpl_HistoryAutoInGitRepo(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	gitRefRead := false
	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.BadgeType = "sparkline"
	deps.HistoryAuto = true
	deps.IsGitRepo = func() bool { return true }
	deps.GitRefReader = func(ref string) (*history.History, error) {
		gitRefRead = true
		return &history.History{Entries: []history.Entry{{Score: 80}}}, nil
	}
	deps.GitRefAppender = func(ref string, entry history.Entry) error {
		return nil
	}

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if !gitRefRead {
		t.Error("should use git ref reader in auto mode when in git repo")
	}
}

func TestGaugeImpl_HistoryAutoNotInGitRepo(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	fileRead := false
	deps := defaultGaugeDeps(fs)
	deps.Config = "config.json"
	deps.Output = "/output.svg"
	deps.BadgeType = "sparkline"
	deps.HistoryAuto = true
	deps.IsGitRepo = func() bool { return false }
	deps.HistoryReader = func(path string) (*history.History, error) {
		fileRead = true
		return &history.History{Entries: []history.Entry{{Score: 80}}}, nil
	}
	deps.HistoryAppender = func(path string, entry history.Entry) error {
		return nil
	}

	err := gaugeImpl(deps)
	if err != nil {
		t.Fatalf("gaugeImpl() error = %v", err)
	}

	if !fileRead {
		t.Error("should use file reader in auto mode when not in git repo")
	}
}

