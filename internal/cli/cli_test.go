package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildBinary builds the confvis binary for testing.
func buildBinary(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "confvis")

	cmd := exec.Command("go", "build", "-o", binPath, "../../cmd/confvis")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, output)
	}

	return binPath
}

func TestGauge_StdinSupport(t *testing.T) {
	bin := buildBinary(t)

	// Read sample JSON
	sampleJSON, err := os.ReadFile("../../testdata/sample.json")
	if err != nil {
		t.Fatalf("reading sample.json: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "output.svg")

	cmd := exec.Command(bin, "gauge", "-c", "-", "-o", outputPath)
	cmd.Stdin = bytes.NewReader(sampleJSON)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with stdin failed: %v\n%s", err, output)
	}

	// Verify output file was created
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}

	if !strings.Contains(string(content), "<svg") {
		t.Error("output does not contain SVG content")
	}
}

func TestGauge_StdoutSupport(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with stdout failed: %v\n%s", err, output)
	}

	if !strings.Contains(string(output), "<svg") {
		t.Error("stdout does not contain SVG content")
	}
}

func TestGauge_StdinAndStdout(t *testing.T) {
	bin := buildBinary(t)

	sampleJSON, err := os.ReadFile("../../testdata/sample.json")
	if err != nil {
		t.Fatalf("reading sample.json: %v", err)
	}

	cmd := exec.Command(bin, "gauge", "-c", "-", "-o", "-")
	cmd.Stdin = bytes.NewReader(sampleJSON)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with stdin+stdout failed: %v\n%s", err, output)
	}

	if !strings.Contains(string(output), "<svg") {
		t.Error("stdout does not contain SVG content")
	}
}

func TestGauge_FailUnder_Pass(t *testing.T) {
	bin := buildBinary(t)

	// sample.json has score 85
	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "--fail-under", "80")

	if err := cmd.Run(); err != nil {
		t.Fatalf("expected exit 0 for score 85 >= threshold 80, got error: %v", err)
	}
}

func TestGauge_FailUnder_Fail(t *testing.T) {
	bin := buildBinary(t)

	// sample.json has score 85
	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "--fail-under", "90")

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit 1 for score 85 < threshold 90, got exit 0")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T", err)
	}

	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}
}

func TestGauge_FailUnder_Message(t *testing.T) {
	bin := buildBinary(t)

	// sample_failing.json has score 62
	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample_failing.json", "-o", "-", "--fail-under", "75")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	_ = cmd.Run() // We expect this to fail

	if !strings.Contains(stderr.String(), "Score 62 is below threshold 75") {
		t.Errorf("expected failure message in stderr, got: %s", stderr.String())
	}
}

func TestGauge_Quiet(t *testing.T) {
	bin := buildBinary(t)

	outputPath := filepath.Join(t.TempDir(), "output.svg")

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", outputPath, "-v", "-q")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("gauge with --quiet failed: %v", err)
	}

	// --quiet should suppress verbose output even when -v is passed
	if stdout.Len() > 0 {
		t.Errorf("expected no stdout with --quiet, got: %s", stdout.String())
	}
}

func TestGauge_Quiet_SuppressesFailUnderMessage(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample_failing.json", "-o", "-", "--fail-under", "75", "-q")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	_ = cmd.Run() // We expect this to fail

	// --quiet should suppress the failure message
	if strings.Contains(stderr.String(), "Score") {
		t.Errorf("expected no message with --quiet, got: %s", stderr.String())
	}
}

func TestGenerate_StdinSupport(t *testing.T) {
	bin := buildBinary(t)

	sampleJSON, err := os.ReadFile("../../testdata/sample.json")
	if err != nil {
		t.Fatalf("reading sample.json: %v", err)
	}

	outputDir := t.TempDir()

	cmd := exec.Command(bin, "generate", "-c", "-", "-o", outputDir)
	cmd.Stdin = bytes.NewReader(sampleJSON)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generate with stdin failed: %v\n%s", err, output)
	}

	// Verify badge.svg was created
	badgePath := filepath.Join(outputDir, "badge.svg")
	if _, err := os.Stat(badgePath); os.IsNotExist(err) {
		t.Error("badge.svg was not created")
	}

	// Verify dashboard was created
	dashboardPath := filepath.Join(outputDir, "dashboard", "index.html")
	if _, err := os.Stat(dashboardPath); os.IsNotExist(err) {
		t.Error("dashboard/index.html was not created")
	}
}

func TestGenerate_FailUnder_Pass(t *testing.T) {
	bin := buildBinary(t)

	outputDir := t.TempDir()

	// sample.json has score 85
	cmd := exec.Command(bin, "generate", "-c", "../../testdata/sample.json", "-o", outputDir, "--fail-under", "80")

	if err := cmd.Run(); err != nil {
		t.Fatalf("expected exit 0 for score 85 >= threshold 80, got error: %v", err)
	}
}

func TestGenerate_FailUnder_Fail(t *testing.T) {
	bin := buildBinary(t)

	outputDir := t.TempDir()

	// sample.json has score 85
	cmd := exec.Command(bin, "generate", "-c", "../../testdata/sample.json", "-o", outputDir, "--fail-under", "90")

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit 1 for score 85 < threshold 90, got exit 0")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T", err)
	}

	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}
}

func TestGenerate_Quiet(t *testing.T) {
	bin := buildBinary(t)

	outputDir := t.TempDir()

	cmd := exec.Command(bin, "generate", "-c", "../../testdata/sample.json", "-o", outputDir, "-v", "-q")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("generate with --quiet failed: %v", err)
	}

	// --quiet should suppress verbose output even when -v is passed
	if stdout.Len() > 0 {
		t.Errorf("expected no stdout with --quiet, got: %s", stdout.String())
	}
}

func TestGauge_FormatJSON(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "-f", "json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with --format json failed: %v\n%s", err, output)
	}

	// Should contain JSON fields
	outputStr := string(output)
	if !strings.Contains(outputStr, `"score": 85`) {
		t.Error("JSON output should contain score")
	}
	if !strings.Contains(outputStr, `"passed": true`) {
		t.Error("JSON output should contain passed status")
	}
	if !strings.Contains(outputStr, `"threshold": 75`) {
		t.Error("JSON output should contain threshold")
	}
}

func TestGauge_FormatText(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "-f", "text")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with --format text failed: %v\n%s", err, output)
	}

	// Should output just the score
	if strings.TrimSpace(string(output)) != "85" {
		t.Errorf("text output should be '85', got: %s", string(output))
	}
}

func TestGauge_FormatInvalid(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "-f", "invalid")

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestGauge_CustomThresholds_CLI(t *testing.T) {
	bin := buildBinary(t)

	// With default thresholds (75, 50), score 85 is green (#1a7f37) for the arc
	// With --green-above 90, score 85 should be yellow (#9a6700) for the arc
	// Note: PASS/FAIL indicator color is separate from arc color
	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "--green-above", "90", "--yellow-above", "70")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with custom thresholds failed: %v\n%s", err, output)
	}

	// The arc stroke should use warning color (yellow)
	outputStr := string(output)
	if !strings.Contains(outputStr, "stroke:#9a6700") {
		t.Error("arc should use warning color #9a6700 for score 85 with --green-above 90")
	}
}

func TestGauge_CustomThresholds_JSON(t *testing.T) {
	bin := buildBinary(t)

	// Create a sample with custom thresholds in JSON
	jsonWithThresholds := `{
		"title": "Test",
		"score": 85,
		"threshold": 75,
		"thresholds": {
			"greenAbove": 90,
			"yellowAbove": 70
		}
	}`

	cmd := exec.Command(bin, "gauge", "-c", "-", "-o", "-")
	cmd.Stdin = strings.NewReader(jsonWithThresholds)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with JSON thresholds failed: %v\n%s", err, output)
	}

	// Arc should use warning color (yellow) since 85 < 90
	outputStr := string(output)
	if !strings.Contains(outputStr, "stroke:#9a6700") {
		t.Error("arc should use warning color for score 85 with greenAbove 90")
	}
}

func TestGauge_CustomThresholds_CLIOverridesJSON(t *testing.T) {
	bin := buildBinary(t)

	// JSON has thresholds that would make score 85 yellow
	jsonWithThresholds := `{
		"title": "Test",
		"score": 85,
		"threshold": 75,
		"thresholds": {
			"greenAbove": 90,
			"yellowAbove": 70
		}
	}`

	// CLI overrides to make score 85 green
	cmd := exec.Command(bin, "gauge", "-c", "-", "-o", "-", "--green-above", "80", "--yellow-above", "50")
	cmd.Stdin = strings.NewReader(jsonWithThresholds)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with CLI override thresholds failed: %v\n%s", err, output)
	}

	// Arc should use success color (green) since CLI says green-above is 80
	outputStr := string(output)
	if !strings.Contains(outputStr, "stroke:#1a7f37") {
		t.Error("arc should use success color when CLI overrides JSON thresholds")
	}
}

func TestGauge_StyleMinimal(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "--style", "minimal")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with --style minimal failed: %v\n%s", err, output)
	}

	// Minimal style uses #fafafa background
	outputStr := string(output)
	if !strings.Contains(outputStr, "fill:#fafafa") {
		t.Error("minimal style should use #fafafa background")
	}
}

func TestGauge_StyleHighContrast(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "--style", "high-contrast")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with --style high-contrast failed: %v\n%s", err, output)
	}

	// High contrast uses #008000 for success (score 85 is green)
	outputStr := string(output)
	if !strings.Contains(outputStr, "stroke:#008000") {
		t.Error("high-contrast style should use #008000 for success")
	}
}

func TestGauge_StyleCorporateDark(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "--style", "corporate", "--dark")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with --style corporate --dark failed: %v\n%s", err, output)
	}

	// Corporate dark uses #141414 background
	outputStr := string(output)
	if !strings.Contains(outputStr, "fill:#141414") {
		t.Error("corporate dark style should use #141414 background")
	}
}

func TestGauge_AutoCalculateScore(t *testing.T) {
	bin := buildBinary(t)

	// JSON without score but with factors
	jsonNoScore := `{
		"title": "Auto Test",
		"threshold": 75,
		"factors": [
			{"name": "A", "score": 80, "weight": 50},
			{"name": "B", "score": 60, "weight": 50}
		]
	}`

	cmd := exec.Command(bin, "gauge", "-c", "-", "-o", "-", "-f", "json")
	cmd.Stdin = strings.NewReader(jsonNoScore)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with auto-calculate failed: %v\n%s", err, output)
	}

	// Score should be auto-calculated: (80*50 + 60*50) / 100 = 70
	outputStr := string(output)
	if !strings.Contains(outputStr, `"score": 70`) {
		t.Errorf("auto-calculated score should be 70, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, `"passed": false`) {
		t.Error("passed should be false since 70 < 75")
	}
}

func TestGauge_AutoCalculate_FailUnder(t *testing.T) {
	bin := buildBinary(t)

	// JSON that auto-calculates to 70
	jsonNoScore := `{
		"title": "Auto Test",
		"threshold": 75,
		"factors": [
			{"name": "A", "score": 80, "weight": 50},
			{"name": "B", "score": 60, "weight": 50}
		]
	}`

	// Should fail because auto-calculated 70 < 75
	cmd := exec.Command(bin, "gauge", "-c", "-", "-o", "-", "-f", "text", "--fail-under", "75")
	cmd.Stdin = strings.NewReader(jsonNoScore)

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit 1 for auto-calculated score 70 < threshold 75")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T", err)
	}

	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}
}
