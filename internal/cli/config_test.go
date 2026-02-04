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

// Unit tests for config getter functions using table-driven tests

func TestConfigGetters_IntWithDefault(t *testing.T) {
	tests := []struct {
		name       string
		getter     func() int
		key        string
		defaultVal int
		setValue   int
	}{
		{"GaugeWidth", getGaugeWidth, "gauge.width", 200, 300},
		{"GaugeHeight", getGaugeHeight, "gauge.height", 120, 180},
		{"GaugeHistoryCount", getGaugeHistoryCount, "gauge.history_count", 10, 20},
		{"FetchTimeout", getFetchTimeout, "fetch.timeout", 30, 60},
		{"FetchThreshold", getFetchThreshold, "fetch.threshold", 75, 80},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_Default", func(t *testing.T) {
			viper.Reset()
			if got := tt.getter(); got != tt.defaultVal {
				t.Errorf("%s() = %d, want %d", tt.name, got, tt.defaultVal)
			}
		})
		t.Run(tt.name+"_FromConfig", func(t *testing.T) {
			viper.Reset()
			viper.Set(tt.key, tt.setValue)
			if got := tt.getter(); got != tt.setValue {
				t.Errorf("%s() = %d, want %d", tt.name, got, tt.setValue)
			}
		})
	}
}

func TestConfigGetters_IntNoDefault(t *testing.T) {
	tests := []struct {
		name     string
		getter   func() int
		key      string
		setValue int
	}{
		{"GaugeFailUnder", getGaugeFailUnder, "gauge.fail_under", 80},
		{"GaugeGreenAbove", getGaugeGreenAbove, "gauge.green_above", 80},
		{"GaugeYellowAbove", getGaugeYellowAbove, "gauge.yellow_above", 50},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_Default", func(t *testing.T) {
			viper.Reset()
			if got := tt.getter(); got != 0 {
				t.Errorf("%s() = %d, want 0", tt.name, got)
			}
		})
		t.Run(tt.name+"_FromConfig", func(t *testing.T) {
			viper.Reset()
			viper.Set(tt.key, tt.setValue)
			if got := tt.getter(); got != tt.setValue {
				t.Errorf("%s() = %d, want %d", tt.name, got, tt.setValue)
			}
		})
	}
}

func TestConfigGetters_StringWithDefault(t *testing.T) {
	tests := []struct {
		name       string
		getter     func() string
		key        string
		defaultVal string
		setValue   string
	}{
		{"GaugeStyle", getGaugeStyle, "gauge.style", "github", "minimal"},
		{"GaugeBadgeType", getGaugeBadgeType, "gauge.badge_type", "gauge", "flat"},
		{"GaugeBaselineRef", getGaugeBaselineRef, "gauge.baseline_ref", baseline.DefaultBaselineRef, "refs/custom/baseline"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_Default", func(t *testing.T) {
			viper.Reset()
			if got := tt.getter(); got != tt.defaultVal {
				t.Errorf("%s() = %q, want %q", tt.name, got, tt.defaultVal)
			}
		})
		t.Run(tt.name+"_FromConfig", func(t *testing.T) {
			viper.Reset()
			viper.Set(tt.key, tt.setValue)
			if got := tt.getter(); got != tt.setValue {
				t.Errorf("%s() = %q, want %q", tt.name, got, tt.setValue)
			}
		})
	}
}

func TestConfigGetters_StringNoDefault(t *testing.T) {
	tests := []struct {
		name     string
		getter   func() string
		key      string
		setValue string
	}{
		{"GaugeHistoryFile", getGaugeHistoryFile, "gauge.history_file", "history.json"},
		{"GaugeHistoryRef", getGaugeHistoryRef, "gauge.history_ref", "refs/custom/history"},
		{"GaugeBaselineFile", getGaugeBaselineFile, "gauge.baseline_file", "baseline.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_Default", func(t *testing.T) {
			viper.Reset()
			if got := tt.getter(); got != "" {
				t.Errorf("%s() = %q, want empty", tt.name, got)
			}
		})
		t.Run(tt.name+"_FromConfig", func(t *testing.T) {
			viper.Reset()
			viper.Set(tt.key, tt.setValue)
			if got := tt.getter(); got != tt.setValue {
				t.Errorf("%s() = %q, want %q", tt.name, got, tt.setValue)
			}
		})
	}
}

func TestConfigGetters_Bool(t *testing.T) {
	tests := []struct {
		name   string
		getter func() bool
		key    string
	}{
		{"GaugeDark", getGaugeDark, "gauge.dark"},
		{"GaugeHistoryAuto", getGaugeHistoryAuto, "gauge.history_auto"},
		{"GaugeCompareBaseline", getGaugeCompareBaseline, "gauge.compare_baseline"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_Default", func(t *testing.T) {
			viper.Reset()
			if got := tt.getter(); got {
				t.Errorf("%s() = true, want false", tt.name)
			}
		})
		t.Run(tt.name+"_True", func(t *testing.T) {
			viper.Reset()
			viper.Set(tt.key, true)
			if got := tt.getter(); !got {
				t.Errorf("%s() = false, want true", tt.name)
			}
		})
	}
}

func TestConfigGetters_SourceSpecific(t *testing.T) {
	t.Run("GetSourceURL", func(t *testing.T) {
		viper.Reset()
		if got := getSourceURL("sonarqube"); got != "" {
			t.Errorf("getSourceURL() = %q, want empty", got)
		}
		viper.Set("sources.sonarqube.url", "https://sonar.example.com")
		if got := getSourceURL("sonarqube"); got != "https://sonar.example.com" {
			t.Errorf("getSourceURL() = %q, want %q", got, "https://sonar.example.com")
		}
	})

	t.Run("GetSourceOrg", func(t *testing.T) {
		viper.Reset()
		if got := getSourceOrg("snyk"); got != "" {
			t.Errorf("getSourceOrg() = %q, want empty", got)
		}
		viper.Set("sources.snyk.org", "my-org")
		if got := getSourceOrg("snyk"); got != "my-org" {
			t.Errorf("getSourceOrg() = %q, want %q", got, "my-org")
		}
	})

	t.Run("GetSourceService", func(t *testing.T) {
		viper.Reset()
		if got := getSourceService("codecov"); got != "" {
			t.Errorf("getSourceService() = %q, want empty", got)
		}
		viper.Set("sources.codecov.service", "gitlab")
		if got := getSourceService("codecov"); got != "gitlab" {
			t.Errorf("getSourceService() = %q, want %q", got, "gitlab")
		}
	})
}

func TestGetConfigFile(t *testing.T) {
	viper.Reset()
	_ = GetConfigFile() // Exercise the function; result depends on test environment
}

func TestGetGaugeFactorThresholds(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		viper.Reset()
		if got := getGaugeFactorThresholds(); len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
	})

	t.Run("IntValues", func(t *testing.T) {
		viper.Reset()
		viper.Set("gauge.factor_thresholds", map[string]interface{}{"Coverage": 80, "Security": 90})
		got := getGaugeFactorThresholds()
		if got["coverage"] != 80 || got["security"] != 90 {
			t.Errorf("unexpected thresholds: %v", got)
		}
	})

	t.Run("FloatValues", func(t *testing.T) {
		viper.Reset()
		viper.Set("gauge.factor_thresholds", map[string]interface{}{"Coverage": 80.5})
		if got := getGaugeFactorThresholds(); got["coverage"] != 80 {
			t.Errorf("expected 80, got %d", got["coverage"])
		}
	})

	t.Run("Int64Values", func(t *testing.T) {
		viper.Reset()
		viper.Set("gauge.factor_thresholds", map[string]interface{}{"Coverage": int64(85)})
		if got := getGaugeFactorThresholds(); got["coverage"] != 85 {
			t.Errorf("expected 85, got %d", got["coverage"])
		}
	})
}
