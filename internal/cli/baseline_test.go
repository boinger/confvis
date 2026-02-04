package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/boinger/confvis/internal/baseline"
	"github.com/boinger/confvis/internal/confidence"
)

func TestParseBaselineConfig_FromStdin(t *testing.T) {
	deps := &BaselineDeps{
		FS:     NewMockFileSystem(),
		Stdin:  strings.NewReader(`{"title": "Test", "score": 85, "threshold": 75}`),
		Config: "-",
	}

	report, err := parseBaselineConfig(deps)
	if err != nil {
		t.Fatalf("parseBaselineConfig() error = %v", err)
	}

	if report.Score != 85 {
		t.Errorf("Score = %d, want 85", report.Score)
	}
}

func TestParseBaselineConfig_FromFile(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.SetFileContent("report.json", `{"title": "File", "score": 90, "threshold": 75}`)

	deps := &BaselineDeps{
		FS:     mockFS,
		Stdin:  strings.NewReader(""),
		Config: "report.json",
	}

	report, err := parseBaselineConfig(deps)
	if err != nil {
		t.Fatalf("parseBaselineConfig() error = %v", err)
	}

	if report.Score != 90 {
		t.Errorf("Score = %d, want 90", report.Score)
	}
}

func TestTruncateCommit(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"abcdefghij", "abcdefg"},
		{"abc", "abc"},
		{"", ""},
		{"1234567", "1234567"},
		{"12345678", "1234567"},
	}

	for _, tt := range tests {
		if got := truncateCommit(tt.input); got != tt.want {
			t.Errorf("truncateCommit(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGetBaselineRef(t *testing.T) {
	viper.Reset()
	if got := getBaselineRef(); got != baseline.DefaultBaselineRef {
		t.Errorf("getBaselineRef() = %q, want %q", got, baseline.DefaultBaselineRef)
	}

	viper.Set("baseline.ref", "refs/custom/baseline")
	if got := getBaselineRef(); got != "refs/custom/baseline" {
		t.Errorf("getBaselineRef() = %q, want %q", got, "refs/custom/baseline")
	}
}

func TestGetBaselineFile(t *testing.T) {
	viper.Reset()
	if got := getBaselineFile(); got != "" {
		t.Errorf("getBaselineFile() = %q, want empty", got)
	}

	viper.Set("baseline.file", "baseline.json")
	if got := getBaselineFile(); got != "baseline.json" {
		t.Errorf("getBaselineFile() = %q, want %q", got, "baseline.json")
	}
}

func TestOutputBaseline_Text(t *testing.T) {
	var buf bytes.Buffer
	deps := &BaselineDeps{
		Stdout: &buf,
		Format: "text",
	}

	b := &baseline.Baseline{
		Report:  confidence.Report{Score: 85},
		SavedAt: "2024-01-01T00:00:00Z",
		Commit:  "abc1234567890",
		Branch:  "main",
	}

	err := outputBaseline(deps, b)
	if err != nil {
		t.Fatalf("outputBaseline() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "85%") {
		t.Error("output should contain score")
	}
	if !strings.Contains(output, "abc1234") {
		t.Error("output should contain truncated commit")
	}
}

func TestOutputBaseline_JSON(t *testing.T) {
	var buf bytes.Buffer
	deps := &BaselineDeps{
		Stdout: &buf,
		Format: "json",
	}

	b := &baseline.Baseline{
		Report: confidence.Report{Score: 85},
	}

	err := outputBaseline(deps, b)
	if err != nil {
		t.Fatalf("outputBaseline() error = %v", err)
	}

	if !strings.Contains(buf.String(), `"score": 85`) {
		t.Error("JSON output should contain score")
	}
}

func TestBaselineShowImpl_InvalidFormat(t *testing.T) {
	deps := &BaselineDeps{
		Format: "invalid",
	}

	err := baselineShowImpl(deps)
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestLoadBaseline_NotInGitRepo(t *testing.T) {
	deps := &BaselineDeps{
		File:      "",
		IsGitRepo: func() bool { return false },
	}

	_, _, err := loadBaseline(deps)
	if err == nil {
		t.Error("expected error when not in git repo and no file specified")
	}
}

func TestLoadBaseline_FromFile(t *testing.T) {
	deps := &BaselineDeps{
		File: "baseline.json",
		FileReader: func(path string) (*baseline.Baseline, error) {
			return &baseline.Baseline{Report: confidence.Report{Score: 75}}, nil
		},
	}

	b, source, err := loadBaseline(deps)
	if err != nil {
		t.Fatalf("loadBaseline() error = %v", err)
	}

	if b.Score != 75 {
		t.Errorf("Score = %d, want 75", b.Score)
	}
	if source != "baseline.json" {
		t.Errorf("source = %q, want %q", source, "baseline.json")
	}
}

func TestLoadBaseline_FromGitRef(t *testing.T) {
	deps := &BaselineDeps{
		File:      "",
		Ref:       "refs/confvis/baseline",
		IsGitRepo: func() bool { return true },
		GitRefReader: func(ref string) (*baseline.Baseline, error) {
			return &baseline.Baseline{Report: confidence.Report{Score: 80}}, nil
		},
	}

	b, source, err := loadBaseline(deps)
	if err != nil {
		t.Fatalf("loadBaseline() error = %v", err)
	}

	if b.Score != 80 {
		t.Errorf("Score = %d, want 80", b.Score)
	}
	if source != "refs/confvis/baseline" {
		t.Errorf("source = %q, want %q", source, "refs/confvis/baseline")
	}
}

func TestSaveBaseline_DryRun(t *testing.T) {
	var buf bytes.Buffer
	deps := &BaselineDeps{
		Stdout: &buf,
		DryRun: true,
		File:   "baseline.json",
	}

	b := &baseline.Baseline{Report: confidence.Report{Score: 85, Title: "Test"}}

	err := saveBaseline(deps, b, true)
	if err != nil {
		t.Fatalf("saveBaseline() error = %v", err)
	}

	if !strings.Contains(buf.String(), "DRY RUN") {
		t.Error("output should contain DRY RUN")
	}
}

func TestSaveBaseline_ToFile(t *testing.T) {
	var savedPath string
	var savedBaseline *baseline.Baseline

	deps := &BaselineDeps{
		Stdout:  &bytes.Buffer{},
		File:    "baseline.json",
		Verbose: true,
		FileWriter: func(path string, b *baseline.Baseline) error {
			savedPath = path
			savedBaseline = b
			return nil
		},
	}

	b := &baseline.Baseline{Report: confidence.Report{Score: 85}, Commit: "abc123"}

	err := saveBaseline(deps, b, true)
	if err != nil {
		t.Fatalf("saveBaseline() error = %v", err)
	}

	if savedPath != "baseline.json" {
		t.Errorf("saved to %q, want %q", savedPath, "baseline.json")
	}
	if savedBaseline.Score != 85 {
		t.Errorf("saved score = %d, want 85", savedBaseline.Score)
	}
}

func TestSaveBaseline_ToGitRef(t *testing.T) {
	var savedRef string

	deps := &BaselineDeps{
		Stdout:  &bytes.Buffer{},
		Ref:     "refs/confvis/baseline",
		Verbose: true,
		GitRefWriter: func(ref string, b *baseline.Baseline) error {
			savedRef = ref
			return nil
		},
	}

	b := &baseline.Baseline{Report: confidence.Report{Score: 85}}

	err := saveBaseline(deps, b, false)
	if err != nil {
		t.Fatalf("saveBaseline() error = %v", err)
	}

	if savedRef != "refs/confvis/baseline" {
		t.Errorf("saved to %q, want %q", savedRef, "refs/confvis/baseline")
	}
}
