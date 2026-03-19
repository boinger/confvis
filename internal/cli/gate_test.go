package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boinger/confvis/internal/baseline"
	"github.com/boinger/confvis/internal/confidence"
)

func defaultGateDeps(fs *MockFileSystem) *GateDeps {
	return &GateDeps{
		FS:               fs,
		Stdin:            nil,
		Stdout:           &bytes.Buffer{},
		Stderr:           &bytes.Buffer{},
		Verbose:          false,
		Quiet:            false,
		ExitFunc:         func(int) {},
		Config:           "",
		InputFormat:      "auto",
		FailUnder:        0,
		FailOnRegression: false,
		FactorThresholds: nil,
		Baseline:         BaselineConfig{},
	}
}

// ============================================================================
// Validation
// ============================================================================

func TestGateImpl_RequiresThresholdFlag(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultGateDeps(fs)
	deps.Config = "config.json"

	err := gateImpl(deps)
	if err == nil {
		t.Fatal("expected error when no threshold flags set")
	}
	if !strings.Contains(err.Error(), "at least one threshold flag is required") {
		t.Errorf("error = %q, want threshold flag message", err)
	}
}

// ============================================================================
// --fail-under
// ============================================================================

func TestGateImpl_FailUnder_Pass(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	exitCalled := false

	deps := defaultGateDeps(fs)
	deps.Stdout = &stdout
	deps.ExitFunc = func(code int) { exitCalled = true }
	deps.Config = "config.json"
	deps.FailUnder = 80

	err := gateImpl(deps)
	if err != nil {
		t.Fatalf("gateImpl() error = %v", err)
	}

	if exitCalled {
		t.Error("exit should not be called when score >= fail-under")
	}

	output := stdout.String()
	if !strings.Contains(output, "Score: 85/100") {
		t.Errorf("output should contain score, got: %s", output)
	}
	if !strings.Contains(output, "Threshold: 80 ✓") {
		t.Errorf("output should contain passing threshold, got: %s", output)
	}
}

func TestGateImpl_FailUnder_Fail(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 78, "threshold": 75}`)

	var stdout bytes.Buffer
	exitCode := -1

	deps := defaultGateDeps(fs)
	deps.Stdout = &stdout
	deps.ExitFunc = func(code int) { exitCode = code }
	deps.Config = "config.json"
	deps.FailUnder = 85

	err := gateImpl(deps)
	if err != nil {
		t.Fatalf("gateImpl() error = %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	output := stdout.String()
	if !strings.Contains(output, "Score: 78/100") {
		t.Errorf("output should contain score, got: %s", output)
	}
	if !strings.Contains(output, "Threshold: 85 ✗") {
		t.Errorf("output should contain failing threshold, got: %s", output)
	}
}

// ============================================================================
// --fail-on-regression
// ============================================================================

func TestGateImpl_FailOnRegression_Pass(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("current.json", `{"title": "Current", "score": 92, "threshold": 75}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 90, "threshold": 75}`)

	var stdout bytes.Buffer
	exitCalled := false

	deps := defaultGateDeps(fs)
	deps.Stdout = &stdout
	deps.ExitFunc = func(code int) { exitCalled = true }
	deps.Config = "current.json"
	deps.FailOnRegression = true
	deps.Baseline = BaselineConfig{
		Compare: "baseline.json",
		FS:      fs,
	}

	err := gateImpl(deps)
	if err != nil {
		t.Fatalf("gateImpl() error = %v", err)
	}

	if exitCalled {
		t.Error("exit should not be called when score improved")
	}

	output := stdout.String()
	if !strings.Contains(output, "Baseline: 90 → 92 (+2) ✓") {
		t.Errorf("output should contain baseline improvement, got: %s", output)
	}
}

func TestGateImpl_FailOnRegression_Fail(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("current.json", `{"title": "Current", "score": 78, "threshold": 75}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	exitCode := -1

	deps := defaultGateDeps(fs)
	deps.Stdout = &stdout
	deps.ExitFunc = func(code int) { exitCode = code }
	deps.Config = "current.json"
	deps.FailOnRegression = true
	deps.Baseline = BaselineConfig{
		Compare: "baseline.json",
		FS:      fs,
	}

	err := gateImpl(deps)
	if err != nil {
		t.Fatalf("gateImpl() error = %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1 for regression, got %d", exitCode)
	}

	output := stdout.String()
	if !strings.Contains(output, "Baseline: 85 → 78 (-7) ✗") {
		t.Errorf("output should contain regression info, got: %s", output)
	}
}

// ============================================================================
// --quiet
// ============================================================================

func TestGateImpl_Quiet_NoOutput(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	deps := defaultGateDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.FailUnder = 50
	deps.Quiet = true

	err := gateImpl(deps)
	if err != nil {
		t.Fatalf("gateImpl() error = %v", err)
	}

	if stdout.String() != "" {
		t.Errorf("quiet mode should produce no output, got: %s", stdout.String())
	}
}

func TestGateImpl_Quiet_FailNoOutput(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 50, "threshold": 75}`)

	var stdout bytes.Buffer
	exitCode := -1

	deps := defaultGateDeps(fs)
	deps.Stdout = &stdout
	deps.ExitFunc = func(code int) { exitCode = code }
	deps.Config = "config.json"
	deps.FailUnder = 75
	deps.Quiet = true

	err := gateImpl(deps)
	if err != nil {
		t.Fatalf("gateImpl() error = %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	if stdout.String() != "" {
		t.Errorf("quiet mode should produce no output on failure, got: %s", stdout.String())
	}
}

// ============================================================================
// --verbose
// ============================================================================

func TestGateImpl_Verbose_ShowsFactors(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{
		"title": "Test",
		"score": 92,
		"threshold": 75,
		"factors": [
			{"name": "Coverage", "score": 88, "weight": 25},
			{"name": "Security", "score": 96, "weight": 25}
		]
	}`)

	var stdout bytes.Buffer
	deps := defaultGateDeps(fs)
	deps.Stdout = &stdout
	deps.Config = "config.json"
	deps.FailUnder = 85
	deps.Verbose = true

	err := gateImpl(deps)
	if err != nil {
		t.Fatalf("gateImpl() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Score: 92/100") {
		t.Errorf("output should contain score, got: %s", output)
	}
	if !strings.Contains(output, "Coverage:") {
		t.Errorf("verbose output should contain factor names, got: %s", output)
	}
	if !strings.Contains(output, "Security:") {
		t.Errorf("verbose output should contain factor names, got: %s", output)
	}
	if !strings.Contains(output, "88") {
		t.Errorf("verbose output should contain factor scores, got: %s", output)
	}
}

// ============================================================================
// --factor-threshold
// ============================================================================

func TestGateImpl_FactorThreshold_Pass(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{
		"title": "Test",
		"score": 85,
		"threshold": 75,
		"factors": [
			{"name": "Coverage", "score": 90, "weight": 50},
			{"name": "Security", "score": 95, "weight": 50}
		]
	}`)

	exitCalled := false
	deps := defaultGateDeps(fs)
	deps.ExitFunc = func(code int) { exitCalled = true }
	deps.Config = "config.json"
	deps.FactorThresholds = map[string]int{"Coverage": 80, "Security": 90}

	err := gateImpl(deps)
	if err != nil {
		t.Fatalf("gateImpl() error = %v", err)
	}

	if exitCalled {
		t.Error("exit should not be called when all factor thresholds met")
	}
}

func TestGateImpl_FactorThreshold_Fail(t *testing.T) {
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

	var stdout bytes.Buffer
	exitCode := -1

	deps := defaultGateDeps(fs)
	deps.Stdout = &stdout
	deps.ExitFunc = func(code int) { exitCode = code }
	deps.Config = "config.json"
	deps.FactorThresholds = map[string]int{"Coverage": 80}

	err := gateImpl(deps)
	if err != nil {
		t.Fatalf("gateImpl() error = %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	output := stdout.String()
	if !strings.Contains(output, "Factor:") && !strings.Contains(output, "Coverage") {
		t.Errorf("output should contain factor failure, got: %s", output)
	}
}

// ============================================================================
// Combined flags
// ============================================================================

func TestGateImpl_Combined_FailUnderAndBaseline(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("current.json", `{"title": "Current", "score": 92, "threshold": 75}`)
	fs.SetFileContent("baseline.json", `{"title": "Baseline", "score": 90, "threshold": 75}`)

	var stdout bytes.Buffer
	exitCalled := false

	deps := defaultGateDeps(fs)
	deps.Stdout = &stdout
	deps.ExitFunc = func(code int) { exitCalled = true }
	deps.Config = "current.json"
	deps.FailUnder = 85
	deps.FailOnRegression = true
	deps.Baseline = BaselineConfig{
		Compare: "baseline.json",
		FS:      fs,
	}

	err := gateImpl(deps)
	if err != nil {
		t.Fatalf("gateImpl() error = %v", err)
	}

	if exitCalled {
		t.Error("exit should not be called when all checks pass")
	}

	output := stdout.String()
	if !strings.Contains(output, "Score: 92/100") {
		t.Errorf("output should contain score, got: %s", output)
	}
	if !strings.Contains(output, "Threshold: 85 ✓") {
		t.Errorf("output should contain passing threshold, got: %s", output)
	}
	if !strings.Contains(output, "Baseline: 90 → 92 (+2) ✓") {
		t.Errorf("output should contain baseline, got: %s", output)
	}
}

// ============================================================================
// Error cases
// ============================================================================

func TestGateImpl_InvalidConfig(t *testing.T) {
	fs := NewMockFileSystem()

	deps := defaultGateDeps(fs)
	deps.Config = "nonexistent.json"
	deps.FailUnder = 80

	err := gateImpl(deps)
	if err == nil {
		t.Fatal("expected error for nonexistent config")
	}
}

func TestGateImpl_InvalidInputFormat(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultGateDeps(fs)
	deps.Config = "config.json"
	deps.InputFormat = "invalid"
	deps.FailUnder = 80

	err := gateImpl(deps)
	if err == nil {
		t.Fatal("expected error for invalid input format")
	}
}

func TestGateImpl_BaselineError(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultGateDeps(fs)
	deps.Config = "config.json"
	deps.FailOnRegression = true
	deps.Baseline = BaselineConfig{
		CompareBaseline: true,
		BaselineFile:    "baseline.json",
		BaselineFileReader: func(path string) (*baseline.Baseline, error) {
			return nil, errors.New("corrupted")
		},
	}

	err := gateImpl(deps)
	if err == nil {
		t.Fatal("expected error for baseline load failure")
	}
	if !strings.Contains(err.Error(), "loading baseline") {
		t.Errorf("error = %q, want baseline error", err)
	}
}

func TestGateImpl_StdinInput(t *testing.T) {
	fs := NewMockFileSystem()
	stdin := strings.NewReader(`{"title": "Stdin", "score": 90, "threshold": 75}`)

	var stdout bytes.Buffer
	exitCalled := false

	deps := defaultGateDeps(fs)
	deps.Stdin = stdin
	deps.Stdout = &stdout
	deps.ExitFunc = func(code int) { exitCalled = true }
	deps.Config = "-"
	deps.FailUnder = 80

	err := gateImpl(deps)
	if err != nil {
		t.Fatalf("gateImpl() error = %v", err)
	}

	if exitCalled {
		t.Error("exit should not be called for passing score")
	}

	if !strings.Contains(stdout.String(), "Score: 90/100") {
		t.Errorf("output should contain score, got: %s", stdout.String())
	}
}

// ============================================================================
// passMarkFor
// ============================================================================

func TestPassMarkFor(t *testing.T) {
	if got := passMarkFor(true); got != "✓" {
		t.Errorf("passMarkFor(true) = %q, want ✓", got)
	}
	if got := passMarkFor(false); got != "✗" {
		t.Errorf("passMarkFor(false) = %q, want ✗", got)
	}
}

// ============================================================================
// CompareBaseline with auto-fetch
// ============================================================================

func TestGateImpl_CompareBaseline_AutoFetch(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stdout bytes.Buffer
	exitCalled := false
	score := 80

	deps := defaultGateDeps(fs)
	deps.Stdout = &stdout
	deps.ExitFunc = func(code int) { exitCalled = true }
	deps.Config = "config.json"
	deps.FailOnRegression = true
	deps.Baseline = BaselineConfig{
		CompareBaseline: true,
		BaselineFile:    "baseline.json",
		BaselineFileReader: func(path string) (*baseline.Baseline, error) {
			return &baseline.Baseline{Report: confidence.Report{Score: &score, Title: "BL"}}, nil
		},
	}

	err := gateImpl(deps)
	if err != nil {
		t.Fatalf("gateImpl() error = %v", err)
	}

	if exitCalled {
		t.Error("exit should not be called when score improved from baseline")
	}

	output := stdout.String()
	if !strings.Contains(output, "Baseline: 80 → 85 (+5) ✓") {
		t.Errorf("output should contain baseline comparison, got: %s", output)
	}
}

// ============================================================================
// $GITHUB_OUTPUT
// ============================================================================

func TestGateImpl_GitHubOutput_Pass(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	tmpFile := filepath.Join(t.TempDir(), "github_output")
	if err := os.WriteFile(tmpFile, nil, 0o644); err != nil {
		t.Fatalf("creating temp file: %v", err)
	}

	deps := defaultGateDeps(fs)
	deps.Config = "config.json"
	deps.FailUnder = 80
	deps.GitHubOutputFile = tmpFile

	err := gateImpl(deps)
	if err != nil {
		t.Fatalf("gateImpl() error = %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("reading github output: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "gate_result=pass") {
		t.Errorf("GITHUB_OUTPUT should contain gate_result=pass, got: %s", content)
	}
	if !strings.Contains(content, "gate_score=85") {
		t.Errorf("GITHUB_OUTPUT should contain gate_score=85, got: %s", content)
	}
}

func TestGateImpl_GitHubOutput_Fail(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 50, "threshold": 75}`)

	tmpFile := filepath.Join(t.TempDir(), "github_output")
	if err := os.WriteFile(tmpFile, nil, 0o644); err != nil {
		t.Fatalf("creating temp file: %v", err)
	}

	deps := defaultGateDeps(fs)
	deps.Config = "config.json"
	deps.FailUnder = 75
	deps.GitHubOutputFile = tmpFile
	deps.ExitFunc = func(int) {} // Don't actually exit

	err := gateImpl(deps)
	if err != nil {
		t.Fatalf("gateImpl() error = %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("reading github output: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "gate_result=fail") {
		t.Errorf("GITHUB_OUTPUT should contain gate_result=fail, got: %s", content)
	}
	if !strings.Contains(content, "gate_score=50") {
		t.Errorf("GITHUB_OUTPUT should contain gate_score=50, got: %s", content)
	}
}

func TestGateImpl_GitHubOutput_Unset(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	deps := defaultGateDeps(fs)
	deps.Config = "config.json"
	deps.FailUnder = 80
	// GitHubOutputFile left empty — should be a no-op

	err := gateImpl(deps)
	if err != nil {
		t.Fatalf("gateImpl() error = %v", err)
	}
	// No assertion on file — just verify no panic or error
}

func TestGateImpl_GitHubOutput_InvalidPath(t *testing.T) {
	fs := NewMockFileSystem()
	fs.SetFileContent("config.json", `{"title": "Test", "score": 85, "threshold": 75}`)

	var stderr bytes.Buffer
	deps := defaultGateDeps(fs)
	deps.Stderr = &stderr
	deps.Config = "config.json"
	deps.FailUnder = 80
	// Point at a path that cannot be opened
	deps.GitHubOutputFile = filepath.Join(t.TempDir(), "no-such-dir", "nested", "output")

	err := gateImpl(deps)
	if err != nil {
		t.Fatalf("gateImpl() error = %v (writeGitHubOutput should not propagate errors)", err)
	}
	// Should produce a warning on stderr
	if !strings.Contains(stderr.String(), "could not write to GITHUB_OUTPUT") {
		t.Errorf("stderr = %q, want warning about GITHUB_OUTPUT", stderr.String())
	}
}
