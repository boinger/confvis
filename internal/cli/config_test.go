package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/boinger/confvis/internal/baseline"
)

func TestConfig_LoadFromCurrentDir(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	// Create config file in temp dir - test gauge style (not flat badge)
	configContent := `
gauge:
  style: minimal
`
	configPath := filepath.Join(tmpDir, ".confvis.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	// Create a sample report
	reportPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(reportPath, []byte(`{"title": "Test", "score": 85, "threshold": 75}`), 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Run from temp dir to pick up config
	cmd := exec.Command(bin, "gauge", "-c", "report.json", "-o", "-")
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// Should use minimal style from config (#fafafa background for gauge badge)
	if !strings.Contains(outputStr, "fill:#fafafa") {
		t.Error("should use minimal style from config")
	}
}

func TestConfig_LoadYML(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	// Create config file with .yml extension
	configContent := `
gauge:
  style: corporate
`
	configPath := filepath.Join(tmpDir, ".confvis.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	// Create a sample report
	reportPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(reportPath, []byte(`{"title": "Test", "score": 85, "threshold": 75}`), 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Run from temp dir to pick up config
	cmd := exec.Command(bin, "gauge", "-c", "report.json", "-o", "-")
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// Corporate style uses #f5f5f5 light background
	if !strings.Contains(outputStr, "fill:#f5f5f5") {
		t.Error("should use corporate style from .yml config")
	}
}

func TestConfig_FlagOverridesConfig(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	// Create config file with minimal style
	configContent := `
gauge:
  style: minimal
`
	configPath := filepath.Join(tmpDir, ".confvis.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	// Create a sample report
	reportPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(reportPath, []byte(`{"title": "Test", "score": 85, "threshold": 75}`), 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Run with --style flag to override config
	cmd := exec.Command(bin, "gauge", "-c", "report.json", "-o", "-", "--style", "high-contrast")
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// High contrast style uses #008000 for green
	if !strings.Contains(outputStr, "stroke:#008000") {
		t.Error("flag should override config style")
	}
}

func TestConfig_EnvOverridesConfig(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	// Create config file with minimal style
	configContent := `
gauge:
  style: minimal
`
	configPath := filepath.Join(tmpDir, ".confvis.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	// Create a sample report
	reportPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(reportPath, []byte(`{"title": "Test", "score": 85, "threshold": 75}`), 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Run with environment variable to override config
	cmd := exec.Command(bin, "gauge", "-c", "report.json", "-o", "-")
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "CONFVIS_GAUGE_STYLE=high-contrast")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// High contrast style uses #008000 for green
	if !strings.Contains(outputStr, "stroke:#008000") {
		t.Error("env should override config style")
	}
}

func TestConfig_FlagOverridesEnv(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	// Create a sample report
	reportPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(reportPath, []byte(`{"title": "Test", "score": 85, "threshold": 75}`), 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Run with both env and flag - flag should win
	cmd := exec.Command(bin, "gauge", "-c", "report.json", "-o", "-", "--style", "high-contrast")
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "CONFVIS_GAUGE_STYLE=minimal")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// High contrast style uses #008000 for green
	if !strings.Contains(outputStr, "stroke:#008000") {
		t.Error("flag should override env")
	}
}

func TestConfig_FailUnderFromConfig(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	// Create config file with fail_under threshold
	configContent := `
gauge:
  fail_under: 90
`
	configPath := filepath.Join(tmpDir, ".confvis.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	// Create a sample report with score 85 (below 90)
	reportPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(reportPath, []byte(`{"title": "Test", "score": 85, "threshold": 75}`), 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Run from temp dir to pick up config
	cmd := exec.Command(bin, "gauge", "-c", "report.json", "-o", "-")
	cmd.Dir = tmpDir

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit 1 due to fail_under from config")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T", err)
	}

	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}
}

func TestConfig_DarkModeFromConfig(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	// Create config file with dark mode
	configContent := `
gauge:
  dark: true
`
	configPath := filepath.Join(tmpDir, ".confvis.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	// Create a sample report
	reportPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(reportPath, []byte(`{"title": "Test", "score": 85, "threshold": 75}`), 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Run from temp dir to pick up config
	cmd := exec.Command(bin, "gauge", "-c", "report.json", "-o", "-")
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// GitHub dark mode uses #0d1117 background
	if !strings.Contains(outputStr, "fill:#0d1117") {
		t.Error("should use dark mode from config")
	}
}

func TestConfig_BadgeTypeFromConfig(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	// Create config file with flat badge type
	configContent := `
gauge:
  badge_type: flat
`
	configPath := filepath.Join(tmpDir, ".confvis.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	// Create a sample report
	reportPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(reportPath, []byte(`{"title": "Test", "score": 85, "threshold": 75}`), 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Run from temp dir to pick up config
	cmd := exec.Command(bin, "gauge", "-c", "report.json", "-o", "-")
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// Flat badge contains PASS/FAIL status text
	if !strings.Contains(outputStr, "PASS") || !strings.Contains(outputStr, "85%") {
		t.Error("should generate flat badge from config")
	}
}

func TestConfig_WidthHeightFromConfig(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	// Create config file with custom dimensions
	configContent := `
gauge:
  width: 300
  height: 200
`
	configPath := filepath.Join(tmpDir, ".confvis.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	// Create a sample report
	reportPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(reportPath, []byte(`{"title": "Test", "score": 85, "threshold": 75}`), 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Run from temp dir to pick up config
	cmd := exec.Command(bin, "gauge", "-c", "report.json", "-o", "-")
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// SVG should have custom dimensions
	if !strings.Contains(outputStr, `width="300"`) || !strings.Contains(outputStr, `height="200"`) {
		t.Error("should use custom dimensions from config")
	}
}

func TestConfig_NoConfigFile(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	// Create a sample report (no config file)
	reportPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(reportPath, []byte(`{"title": "Test", "score": 85, "threshold": 75}`), 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Run from temp dir (no config file present)
	cmd := exec.Command(bin, "gauge", "-c", "report.json", "-o", "-")
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// Should use defaults (github style)
	if !strings.Contains(outputStr, "<svg") {
		t.Error("should work without config file")
	}
}

func TestConfig_SourceSpecific_SonarQubeURL(t *testing.T) {
	// This test verifies that source-specific config is parsed correctly
	// We can't actually test fetch without a real server, but we can verify
	// the config loading doesn't break anything
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	// Create config file with source-specific settings
	configContent := `
sources:
  sonarqube:
    url: https://sonar.example.com
  codecov:
    service: gitlab
  snyk:
    org: my-org-id
fetch:
  timeout: 60
  threshold: 80
`
	configPath := filepath.Join(tmpDir, ".confvis.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	// Create a sample report
	reportPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(reportPath, []byte(`{"title": "Test", "score": 85, "threshold": 75}`), 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Verify gauge still works (config loading doesn't break)
	cmd := exec.Command(bin, "gauge", "-c", "report.json", "-o", "-")
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge failed with source config present: %v\n%s", err, output)
	}

	if !strings.Contains(string(output), "<svg") {
		t.Error("gauge should work with source-specific config present")
	}
}

func TestConfig_GreenYellowThresholdsFromConfig(t *testing.T) {
	bin := buildBinary(t)
	tmpDir := t.TempDir()

	// Create config file with custom thresholds
	// With green_above=90, score 85 should be yellow
	configContent := `
gauge:
  green_above: 90
  yellow_above: 70
`
	configPath := filepath.Join(tmpDir, ".confvis.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	// Create a sample report with score 85
	reportPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(reportPath, []byte(`{"title": "Test", "score": 85, "threshold": 75}`), 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Run from temp dir to pick up config
	cmd := exec.Command(bin, "gauge", "-c", "report.json", "-o", "-")
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// With green_above=90, score 85 should use warning color (yellow) for arc
	if !strings.Contains(outputStr, "stroke:#9a6700") {
		t.Error("should use yellow/warning color for score 85 with green_above=90")
	}
}

func TestConfig_HomeDir(t *testing.T) {
	// This test verifies that config is loaded from ~/.config/confvis/
	// We'll skip this in CI where we can't reliably write to home dir
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	configDir := filepath.Join(home, ".config", "confvis")
	// Viper looks for .confvis.yaml (the config name set via SetConfigName)
	configPath := filepath.Join(configDir, ".confvis.yaml")

	// Check if we can create the config dir
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Skip("cannot create config directory in home")
	}

	// Check if config file already exists (don't overwrite user's config)
	if _, err := os.Stat(configPath); err == nil {
		t.Skip("user config file already exists, skipping to avoid overwriting")
	}

	// Create temporary config file
	configContent := `
gauge:
  style: corporate
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Skipf("cannot write config file: %v", err)
	}
	defer func() { _ = os.Remove(configPath) }()

	bin := buildBinary(t)
	tmpDir := t.TempDir()

	// Create a sample report (no config in current dir)
	reportPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(reportPath, []byte(`{"title": "Test", "score": 85, "threshold": 75}`), 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Run from temp dir (should pick up home dir config)
	cmd := exec.Command(bin, "gauge", "-c", "report.json", "-o", "-")
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// Corporate style uses #f5f5f5 light background
	if !strings.Contains(outputStr, "fill:#f5f5f5") {
		t.Error("should use corporate style from home dir config")
	}
}

func TestConfig_CurrentDirOverridesHomeDir(t *testing.T) {
	// This test verifies that current dir config takes precedence over home dir config
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	configDir := filepath.Join(home, ".config", "confvis")
	// Viper looks for .confvis.yaml (the config name set via SetConfigName)
	homeConfigPath := filepath.Join(configDir, ".confvis.yaml")

	// Check if we can create the config dir
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Skip("cannot create config directory in home")
	}

	// Check if config file already exists (don't overwrite user's config)
	if _, err := os.Stat(homeConfigPath); err == nil {
		t.Skip("user config file already exists, skipping to avoid overwriting")
	}

	// Create home dir config with corporate style
	homeConfigContent := `
gauge:
  style: corporate
`
	if err := os.WriteFile(homeConfigPath, []byte(homeConfigContent), 0o644); err != nil {
		t.Skipf("cannot write home config file: %v", err)
	}
	defer func() { _ = os.Remove(homeConfigPath) }()

	bin := buildBinary(t)
	tmpDir := t.TempDir()

	// Create current dir config with minimal style (should override home)
	currentConfigContent := `
gauge:
  style: minimal
`
	currentConfigPath := filepath.Join(tmpDir, ".confvis.yaml")
	if err := os.WriteFile(currentConfigPath, []byte(currentConfigContent), 0o644); err != nil {
		t.Fatalf("writing current config: %v", err)
	}

	// Create a sample report
	reportPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(reportPath, []byte(`{"title": "Test", "score": 85, "threshold": 75}`), 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Run from temp dir (should use current dir config, not home dir)
	cmd := exec.Command(bin, "gauge", "-c", "report.json", "-o", "-")
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// Minimal style uses #fafafa background (not corporate's #f5f5f5)
	if !strings.Contains(outputStr, "fill:#fafafa") {
		t.Error("current dir config should override home dir config")
	}
}

// Unit tests for config getter functions

func TestGetGaugeWidth_Default(t *testing.T) {
	// Reset viper state
	viper.Reset()

	width := getGaugeWidth()
	if width != 200 {
		t.Errorf("expected default width 200, got %d", width)
	}
}

func TestGetGaugeWidth_FromConfig(t *testing.T) {
	viper.Reset()
	viper.Set("gauge.width", 300)

	width := getGaugeWidth()
	if width != 300 {
		t.Errorf("expected width 300, got %d", width)
	}
}

func TestGetGaugeHeight_Default(t *testing.T) {
	viper.Reset()

	height := getGaugeHeight()
	if height != 120 {
		t.Errorf("expected default height 120, got %d", height)
	}
}

func TestGetGaugeHeight_FromConfig(t *testing.T) {
	viper.Reset()
	viper.Set("gauge.height", 180)

	height := getGaugeHeight()
	if height != 180 {
		t.Errorf("expected height 180, got %d", height)
	}
}

func TestGetGaugeStyle_Default(t *testing.T) {
	viper.Reset()

	style := getGaugeStyle()
	if style != "github" {
		t.Errorf("expected default style 'github', got %q", style)
	}
}

func TestGetGaugeStyle_FromConfig(t *testing.T) {
	viper.Reset()
	viper.Set("gauge.style", "minimal")

	style := getGaugeStyle()
	if style != "minimal" {
		t.Errorf("expected style 'minimal', got %q", style)
	}
}

func TestGetGaugeDark(t *testing.T) {
	viper.Reset()

	dark := getGaugeDark()
	if dark {
		t.Error("expected dark mode false by default")
	}

	viper.Set("gauge.dark", true)
	dark = getGaugeDark()
	if !dark {
		t.Error("expected dark mode true when set")
	}
}

func TestGetGaugeFailUnder(t *testing.T) {
	viper.Reset()

	failUnder := getGaugeFailUnder()
	if failUnder != 0 {
		t.Errorf("expected fail_under 0 by default, got %d", failUnder)
	}

	viper.Set("gauge.fail_under", 80)
	failUnder = getGaugeFailUnder()
	if failUnder != 80 {
		t.Errorf("expected fail_under 80, got %d", failUnder)
	}
}

func TestGetGaugeBadgeType_Default(t *testing.T) {
	viper.Reset()

	badgeType := getGaugeBadgeType()
	if badgeType != "gauge" {
		t.Errorf("expected default badge type 'gauge', got %q", badgeType)
	}
}

func TestGetGaugeBadgeType_FromConfig(t *testing.T) {
	viper.Reset()
	viper.Set("gauge.badge_type", "flat")

	badgeType := getGaugeBadgeType()
	if badgeType != "flat" {
		t.Errorf("expected badge type 'flat', got %q", badgeType)
	}
}

func TestGetGaugeHistoryFile(t *testing.T) {
	viper.Reset()

	file := getGaugeHistoryFile()
	if file != "" {
		t.Errorf("expected empty history file by default, got %q", file)
	}

	viper.Set("gauge.history_file", "history.json")
	file = getGaugeHistoryFile()
	if file != "history.json" {
		t.Errorf("expected history file 'history.json', got %q", file)
	}
}

func TestGetGaugeHistoryCount_Default(t *testing.T) {
	viper.Reset()

	count := getGaugeHistoryCount()
	if count != 10 {
		t.Errorf("expected default history count 10, got %d", count)
	}
}

func TestGetGaugeHistoryCount_FromConfig(t *testing.T) {
	viper.Reset()
	viper.Set("gauge.history_count", 20)

	count := getGaugeHistoryCount()
	if count != 20 {
		t.Errorf("expected history count 20, got %d", count)
	}
}

func TestGetGaugeHistoryRef(t *testing.T) {
	viper.Reset()

	ref := getGaugeHistoryRef()
	if ref != "" {
		t.Errorf("expected empty history ref by default, got %q", ref)
	}

	viper.Set("gauge.history_ref", "refs/custom/history")
	ref = getGaugeHistoryRef()
	if ref != "refs/custom/history" {
		t.Errorf("expected history ref 'refs/custom/history', got %q", ref)
	}
}

func TestGetGaugeHistoryAuto(t *testing.T) {
	viper.Reset()

	auto := getGaugeHistoryAuto()
	if auto {
		t.Error("expected history auto false by default")
	}

	viper.Set("gauge.history_auto", true)
	auto = getGaugeHistoryAuto()
	if !auto {
		t.Error("expected history auto true when set")
	}
}

func TestGetGaugeGreenAbove(t *testing.T) {
	viper.Reset()

	green := getGaugeGreenAbove()
	if green != 0 {
		t.Errorf("expected green_above 0 by default, got %d", green)
	}

	viper.Set("gauge.green_above", 80)
	green = getGaugeGreenAbove()
	if green != 80 {
		t.Errorf("expected green_above 80, got %d", green)
	}
}

func TestGetGaugeYellowAbove(t *testing.T) {
	viper.Reset()

	yellow := getGaugeYellowAbove()
	if yellow != 0 {
		t.Errorf("expected yellow_above 0 by default, got %d", yellow)
	}

	viper.Set("gauge.yellow_above", 50)
	yellow = getGaugeYellowAbove()
	if yellow != 50 {
		t.Errorf("expected yellow_above 50, got %d", yellow)
	}
}

func TestGetGaugeCompareBaseline(t *testing.T) {
	viper.Reset()

	compare := getGaugeCompareBaseline()
	if compare {
		t.Error("expected compare_baseline false by default")
	}

	viper.Set("gauge.compare_baseline", true)
	compare = getGaugeCompareBaseline()
	if !compare {
		t.Error("expected compare_baseline true when set")
	}
}

func TestGetGaugeBaselineRef_Default(t *testing.T) {
	viper.Reset()

	ref := getGaugeBaselineRef()
	if ref != baseline.DefaultBaselineRef {
		t.Errorf("expected default baseline ref %q, got %q", baseline.DefaultBaselineRef, ref)
	}
}

func TestGetGaugeBaselineRef_FromConfig(t *testing.T) {
	viper.Reset()
	viper.Set("gauge.baseline_ref", "refs/custom/baseline")

	ref := getGaugeBaselineRef()
	if ref != "refs/custom/baseline" {
		t.Errorf("expected baseline ref 'refs/custom/baseline', got %q", ref)
	}
}

func TestGetGaugeBaselineFile(t *testing.T) {
	viper.Reset()

	file := getGaugeBaselineFile()
	if file != "" {
		t.Errorf("expected empty baseline file by default, got %q", file)
	}

	viper.Set("gauge.baseline_file", "baseline.json")
	file = getGaugeBaselineFile()
	if file != "baseline.json" {
		t.Errorf("expected baseline file 'baseline.json', got %q", file)
	}
}

func TestGetFetchTimeout_Default(t *testing.T) {
	viper.Reset()

	timeout := getFetchTimeout()
	if timeout != 30 {
		t.Errorf("expected default fetch timeout 30, got %d", timeout)
	}
}

func TestGetFetchTimeout_FromConfig(t *testing.T) {
	viper.Reset()
	viper.Set("fetch.timeout", 60)

	timeout := getFetchTimeout()
	if timeout != 60 {
		t.Errorf("expected fetch timeout 60, got %d", timeout)
	}
}

func TestGetFetchThreshold_Default(t *testing.T) {
	viper.Reset()

	threshold := getFetchThreshold()
	if threshold != 75 {
		t.Errorf("expected default fetch threshold 75, got %d", threshold)
	}
}

func TestGetFetchThreshold_FromConfig(t *testing.T) {
	viper.Reset()
	viper.Set("fetch.threshold", 80)

	threshold := getFetchThreshold()
	if threshold != 80 {
		t.Errorf("expected fetch threshold 80, got %d", threshold)
	}
}

func TestGetSourceURL(t *testing.T) {
	viper.Reset()

	url := getSourceURL("sonarqube")
	if url != "" {
		t.Errorf("expected empty URL by default, got %q", url)
	}

	viper.Set("sources.sonarqube.url", "https://sonar.example.com")
	url = getSourceURL("sonarqube")
	if url != "https://sonar.example.com" {
		t.Errorf("expected URL 'https://sonar.example.com', got %q", url)
	}
}

func TestGetSourceOrg(t *testing.T) {
	viper.Reset()

	org := getSourceOrg("snyk")
	if org != "" {
		t.Errorf("expected empty org by default, got %q", org)
	}

	viper.Set("sources.snyk.org", "my-org")
	org = getSourceOrg("snyk")
	if org != "my-org" {
		t.Errorf("expected org 'my-org', got %q", org)
	}
}

func TestGetSourceService(t *testing.T) {
	viper.Reset()

	service := getSourceService("codecov")
	if service != "" {
		t.Errorf("expected empty service by default, got %q", service)
	}

	viper.Set("sources.codecov.service", "gitlab")
	service = getSourceService("codecov")
	if service != "gitlab" {
		t.Errorf("expected service 'gitlab', got %q", service)
	}
}

func TestGetConfigFile_None(t *testing.T) {
	viper.Reset()

	// With no config file loaded, should return empty string
	configFile := GetConfigFile()
	// This may or may not be empty depending on test environment
	_ = configFile // Exercise the function
}

func TestGetGaugeFactorThresholds(t *testing.T) {
	viper.Reset()

	// Default should be empty map
	thresholds := getGaugeFactorThresholds()
	if len(thresholds) != 0 {
		t.Errorf("expected empty factor thresholds by default, got %v", thresholds)
	}

	// Test with int values
	viper.Set("gauge.factor_thresholds", map[string]interface{}{
		"Coverage": 80,
		"Security": 90,
	})

	thresholds = getGaugeFactorThresholds()
	if thresholds["coverage"] != 80 {
		t.Errorf("expected Coverage threshold 80, got %d", thresholds["coverage"])
	}
	if thresholds["security"] != 90 {
		t.Errorf("expected Security threshold 90, got %d", thresholds["security"])
	}
}

func TestGetGaugeFactorThresholds_FloatValues(t *testing.T) {
	viper.Reset()

	// Test with float64 values (YAML often parses numbers as float64)
	viper.Set("gauge.factor_thresholds", map[string]interface{}{
		"Coverage": 80.5,
	})

	thresholds := getGaugeFactorThresholds()
	if thresholds["coverage"] != 80 {
		t.Errorf("expected Coverage threshold 80 (from float64), got %d", thresholds["coverage"])
	}
}

func TestGetGaugeFactorThresholds_Int64Values(t *testing.T) {
	viper.Reset()

	// Test with int64 values
	viper.Set("gauge.factor_thresholds", map[string]interface{}{
		"Coverage": int64(85),
	})

	thresholds := getGaugeFactorThresholds()
	if thresholds["coverage"] != 85 {
		t.Errorf("expected Coverage threshold 85 (from int64), got %d", thresholds["coverage"])
	}
}
