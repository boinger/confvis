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

func TestGauge_CustomPassLabel(t *testing.T) {
	bin := buildBinary(t)

	jsonCustomLabel := `{
		"title": "Test",
		"score": 85,
		"threshold": 75,
		"passLabel": "APPROVED"
	}`

	cmd := exec.Command(bin, "gauge", "-c", "-", "-o", "-")
	cmd.Stdin = strings.NewReader(jsonCustomLabel)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with custom pass label failed: %v\n%s", err, output)
	}

	if !strings.Contains(string(output), ">APPROVED<") {
		t.Error("output should contain custom pass label APPROVED")
	}
}

func TestGauge_CustomFailLabel(t *testing.T) {
	bin := buildBinary(t)

	jsonCustomLabel := `{
		"title": "Test",
		"score": 60,
		"threshold": 75,
		"failLabel": "NEEDS WORK"
	}`

	cmd := exec.Command(bin, "gauge", "-c", "-", "-o", "-")
	cmd.Stdin = strings.NewReader(jsonCustomLabel)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with custom fail label failed: %v\n%s", err, output)
	}

	if !strings.Contains(string(output), ">NEEDS WORK<") {
		t.Error("output should contain custom fail label NEEDS WORK")
	}
}

func TestGauge_MetadataInJSONOutput(t *testing.T) {
	bin := buildBinary(t)

	jsonWithMetadata := `{
		"title": "Test",
		"score": 85,
		"threshold": 75,
		"version": "2.0",
		"generatedAt": "2024-01-15T12:00:00Z",
		"source": "test-suite"
	}`

	cmd := exec.Command(bin, "gauge", "-c", "-", "-o", "-", "-f", "json")
	cmd.Stdin = strings.NewReader(jsonWithMetadata)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with metadata failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, `"version": "2.0"`) {
		t.Error("JSON output should contain version")
	}
	if !strings.Contains(outputStr, `"generatedAt": "2024-01-15T12:00:00Z"`) {
		t.Error("JSON output should contain generatedAt")
	}
	if !strings.Contains(outputStr, `"source": "test-suite"`) {
		t.Error("JSON output should contain source")
	}
}

func TestGauge_FormatMarkdown(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "-f", "markdown")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with --format markdown failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// Should contain header with title, score, and PASS
	if !strings.Contains(outputStr, "## Code Quality Report: 85% (PASS)") {
		t.Error("markdown should contain header with title, score, and status")
	}
	// Should contain table headers
	if !strings.Contains(outputStr, "| Factor | Score | Weight |") {
		t.Error("markdown should contain factor table header")
	}
	// Should contain factor data
	if !strings.Contains(outputStr, "| Test Coverage | 92% | 30% |") {
		t.Error("markdown should contain factor rows")
	}
}

func TestGauge_FormatMarkdown_WithURLs(t *testing.T) {
	bin := buildBinary(t)

	jsonWithURLs := `{
		"title": "Test",
		"score": 85,
		"threshold": 75,
		"factors": [
			{"name": "Coverage", "score": 90, "weight": 50, "url": "https://example.com/coverage"},
			{"name": "Lint", "score": 80, "weight": 50}
		]
	}`

	cmd := exec.Command(bin, "gauge", "-c", "-", "-o", "-", "-f", "markdown")
	cmd.Stdin = strings.NewReader(jsonWithURLs)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge markdown with URLs failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// Factor with URL should be a markdown link
	if !strings.Contains(outputStr, "[Coverage](https://example.com/coverage)") {
		t.Error("markdown should contain URL as markdown link")
	}
	// Factor without URL should be plain text
	if !strings.Contains(outputStr, "| Lint |") {
		t.Error("factor without URL should be plain text")
	}
}

func TestGauge_FormatMarkdown_Failing(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample_failing.json", "-o", "-", "-f", "markdown")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge markdown with failing report failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// Should show FAIL status
	if !strings.Contains(outputStr, "(FAIL)") {
		t.Error("markdown should show FAIL status for failing report")
	}
}

func TestGauge_FormatMarkdown_CustomLabels(t *testing.T) {
	bin := buildBinary(t)

	jsonCustomLabels := `{
		"title": "Test",
		"score": 85,
		"threshold": 75,
		"passLabel": "APPROVED"
	}`

	cmd := exec.Command(bin, "gauge", "-c", "-", "-o", "-", "-f", "markdown")
	cmd.Stdin = strings.NewReader(jsonCustomLabels)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge markdown with custom labels failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// Should use custom pass label
	if !strings.Contains(outputStr, "(APPROVED)") {
		t.Error("markdown should use custom pass label")
	}
}

func TestGauge_Compare_JSON(t *testing.T) {
	bin := buildBinary(t)

	// sample.json has score 85, sample_failing.json has score 62
	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "--compare", "../../testdata/sample_failing.json", "-o", "-", "-f", "json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge compare JSON failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, `"baseline": 62`) {
		t.Error("JSON output should contain baseline score")
	}
	if !strings.Contains(outputStr, `"delta": 23`) {
		t.Error("JSON output should contain delta")
	}
}

func TestGauge_Compare_Text(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "--compare", "../../testdata/sample_failing.json", "-o", "-", "-f", "text")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge compare text failed: %v\n%s", err, output)
	}

	if strings.TrimSpace(string(output)) != "85 (+23)" {
		t.Errorf("text output should be '85 (+23)', got: %s", string(output))
	}
}

func TestGauge_Compare_Text_Negative(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample_failing.json", "--compare", "../../testdata/sample.json", "-o", "-", "-f", "text")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge compare text failed: %v\n%s", err, output)
	}

	if strings.TrimSpace(string(output)) != "62 (-23)" {
		t.Errorf("text output should be '62 (-23)', got: %s", string(output))
	}
}

func TestGauge_Compare_Markdown(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "--compare", "../../testdata/sample_failing.json", "-o", "-", "-f", "markdown")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge compare markdown failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "[+23 from 62%]") {
		t.Error("markdown output should contain delta from baseline")
	}
}

func TestGauge_FailOnRegression_Pass(t *testing.T) {
	bin := buildBinary(t)

	// Score improved: 85 > 62, no regression
	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "--compare", "../../testdata/sample_failing.json", "-o", "-", "-f", "text", "--fail-on-regression")

	if err := cmd.Run(); err != nil {
		t.Fatalf("expected exit 0 when score improved, got error: %v", err)
	}
}

func TestGauge_FailOnRegression_Fail(t *testing.T) {
	bin := buildBinary(t)

	// Score regressed: 62 < 85
	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample_failing.json", "--compare", "../../testdata/sample.json", "-o", "-", "-f", "text", "--fail-on-regression")

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit 1 when score regressed, got exit 0")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T", err)
	}

	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}
}

func TestGauge_FailOnRegression_Message(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample_failing.json", "--compare", "../../testdata/sample.json", "-o", "-", "-f", "text", "--fail-on-regression")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	_ = cmd.Run() // Expected to fail

	if !strings.Contains(stderr.String(), "Score regressed from 85 to 62 (-23)") {
		t.Errorf("expected regression message in stderr, got: %s", stderr.String())
	}
}

func TestGauge_FlatBadge(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "--badge-type", "flat")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with --badge-type flat failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// Should be an SVG with different structure than gauge
	if !strings.Contains(outputStr, "<svg") {
		t.Error("flat badge should be an SVG")
	}
	// Should contain the title as label
	if !strings.Contains(outputStr, "Code Quality Report") {
		t.Error("flat badge should contain report title as label")
	}
	// Should contain PASS status
	if !strings.Contains(outputStr, "PASS") {
		t.Error("flat badge should contain PASS status")
	}
	// Should contain score percentage
	if !strings.Contains(outputStr, "85%") {
		t.Error("flat badge should contain score percentage")
	}
}

func TestGauge_FlatBadge_CustomLabel(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "--badge-type", "flat", "--label", "Quality")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with --badge-type flat --label failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// Should contain custom label
	if !strings.Contains(outputStr, "Quality") {
		t.Error("flat badge should contain custom label")
	}
	// Should NOT contain the report title
	if strings.Contains(outputStr, "Code Quality Report") {
		t.Error("flat badge should not contain report title when custom label is used")
	}
}

func TestGauge_FlatBadge_Failing(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample_failing.json", "-o", "-", "--badge-type", "flat")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge flat badge with failing report failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// Should contain FAIL status
	if !strings.Contains(outputStr, "FAIL") {
		t.Error("flat badge should contain FAIL status for failing report")
	}
	// Should contain score percentage
	if !strings.Contains(outputStr, "62%") {
		t.Error("flat badge should contain score percentage")
	}
}

func TestGauge_FlatBadge_DarkMode(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "--badge-type", "flat", "--dark")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge flat badge with dark mode failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// Dark mode should use darker label background (#333 instead of #555)
	if !strings.Contains(outputStr, "#333") {
		t.Error("flat badge dark mode should use darker label background")
	}
}

func TestGauge_BadgeTypeInvalid(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "--badge-type", "invalid")

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error for invalid badge-type")
	}
}

func TestGauge_YAMLInput(t *testing.T) {
	bin := buildBinary(t)

	// Auto-detect YAML from extension
	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.yaml", "-o", "-", "-f", "json")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with YAML input failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, `"score": 85`) {
		t.Error("YAML input should produce same score as JSON")
	}
	if !strings.Contains(outputStr, `"passed": true`) {
		t.Error("YAML input should produce same passed status as JSON")
	}
}

func TestGauge_YAMLInput_Stdin(t *testing.T) {
	bin := buildBinary(t)

	// Read sample YAML
	sampleYAML, err := os.ReadFile("../../testdata/sample.yaml")
	if err != nil {
		t.Fatalf("reading sample.yaml: %v", err)
	}

	cmd := exec.Command(bin, "gauge", "-c", "-", "--input-format", "yaml", "-o", "-", "-f", "text")
	cmd.Stdin = bytes.NewReader(sampleYAML)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge with YAML stdin failed: %v\n%s", err, output)
	}

	if strings.TrimSpace(string(output)) != "85" {
		t.Errorf("expected score 85, got: %s", string(output))
	}
}

func TestGauge_InputFormatInvalid(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "--input-format", "invalid")

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error for invalid input-format")
	}
}

func TestGenerate_YAMLInput(t *testing.T) {
	bin := buildBinary(t)

	outputDir := t.TempDir()

	cmd := exec.Command(bin, "generate", "-c", "../../testdata/sample.yaml", "-o", outputDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generate with YAML input failed: %v\n%s", err, output)
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

func TestGauge_Sparkline(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "--badge-type", "sparkline")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge sparkline failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "<svg") {
		t.Error("sparkline should be an SVG")
	}
	if !strings.Contains(outputStr, "85%") {
		t.Error("sparkline should contain score percentage")
	}
}

func TestGauge_Sparkline_WithHistory(t *testing.T) {
	bin := buildBinary(t)

	// Create history file
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.jsonl")
	historyContent := `{"score": 70, "timestamp": "2024-01-01T10:00:00Z"}
{"score": 75, "timestamp": "2024-01-02T10:00:00Z"}
{"score": 80, "timestamp": "2024-01-03T10:00:00Z"}
`
	if err := os.WriteFile(historyPath, []byte(historyContent), 0o644); err != nil {
		t.Fatalf("writing history file: %v", err)
	}

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "--badge-type", "sparkline", "--history-file", historyPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge sparkline with history failed: %v\n%s", err, output)
	}

	outputStr := string(output)
	// Should have polyline for sparkline
	if !strings.Contains(outputStr, "polyline") {
		t.Error("sparkline with history should contain polyline element")
	}

	// Check that current score was appended to history
	historyData, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("reading history file: %v", err)
	}
	if !strings.Contains(string(historyData), `"score":85`) {
		t.Error("current score should be appended to history file")
	}
}

func TestGauge_Sparkline_HistoryCount(t *testing.T) {
	bin := buildBinary(t)

	// Create history file with more entries than default
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "history.jsonl")
	historyContent := `{"score": 60, "timestamp": "2024-01-01T10:00:00Z"}
{"score": 62, "timestamp": "2024-01-02T10:00:00Z"}
{"score": 65, "timestamp": "2024-01-03T10:00:00Z"}
{"score": 68, "timestamp": "2024-01-04T10:00:00Z"}
{"score": 70, "timestamp": "2024-01-05T10:00:00Z"}
{"score": 72, "timestamp": "2024-01-06T10:00:00Z"}
{"score": 75, "timestamp": "2024-01-07T10:00:00Z"}
{"score": 78, "timestamp": "2024-01-08T10:00:00Z"}
{"score": 80, "timestamp": "2024-01-09T10:00:00Z"}
{"score": 82, "timestamp": "2024-01-10T10:00:00Z"}
`
	if err := os.WriteFile(historyPath, []byte(historyContent), 0o644); err != nil {
		t.Fatalf("writing history file: %v", err)
	}

	// Test with custom history count
	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "--badge-type", "sparkline", "--history-file", historyPath, "--history-count", "5")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge sparkline with history count failed: %v\n%s", err, output)
	}

	if !strings.Contains(string(output), "<svg") {
		t.Error("output should be an SVG")
	}
}

func TestGauge_BadgeType_Sparkline_Invalid(t *testing.T) {
	bin := buildBinary(t)

	// Test with non-existent history file (should work, creates new file)
	tmpDir := t.TempDir()
	historyPath := filepath.Join(tmpDir, "nonexistent", "history.jsonl")

	cmd := exec.Command(bin, "gauge", "-c", "../../testdata/sample.json", "-o", "-", "--badge-type", "sparkline", "--history-file", historyPath)

	// This should fail because the directory doesn't exist
	err := cmd.Run()
	if err == nil {
		t.Error("expected error when history file directory doesn't exist")
	}
}

func TestGauge_Sparkline_HistoryRef(t *testing.T) {
	bin := buildBinary(t)

	// Create a temporary git repo
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	// Create sample report in the repo
	reportPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(reportPath, []byte(`{"title": "Test", "score": 85, "threshold": 75}`), 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Run with --history-ref
	cmd = exec.Command(bin, "gauge", "-c", "report.json", "-o", "-", "--badge-type", "sparkline", "--history-ref", "refs/confvis/test-history")
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge sparkline with history-ref failed: %v\n%s", err, output)
	}

	if !strings.Contains(string(output), "<svg") {
		t.Error("output should be an SVG")
	}

	// Verify the ref was created
	cmd = exec.Command("git", "show-ref", "--verify", "refs/confvis/test-history")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Error("history ref should be created after sparkline generation")
	}

	// Run again to verify history is read and appended
	cmd = exec.Command(bin, "gauge", "-c", "report.json", "-o", "-", "--badge-type", "sparkline", "--history-ref", "refs/confvis/test-history")
	cmd.Dir = tmpDir

	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("second gauge sparkline with history-ref failed: %v\n%s", err, output)
	}

	// Verify the ref content has two entries
	cmd = exec.Command("git", "cat-file", "-p", "refs/confvis/test-history")
	cmd.Dir = tmpDir
	refOutput, err := cmd.Output()
	if err != nil {
		t.Fatalf("reading git ref: %v", err)
	}

	// Count entries (lines with score)
	lines := strings.Split(string(refOutput), "\n")
	entryCount := 0
	for _, line := range lines {
		if strings.Contains(line, `"score":85`) {
			entryCount++
		}
	}
	if entryCount != 2 {
		t.Errorf("expected 2 history entries, got %d in:\n%s", entryCount, string(refOutput))
	}
}

func TestGauge_Sparkline_HistoryAuto_InGitRepo(t *testing.T) {
	bin := buildBinary(t)

	// Create a temporary git repo
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	// Create sample report in the repo
	reportPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(reportPath, []byte(`{"title": "Test", "score": 85, "threshold": 75}`), 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Run with --history-auto (should use git ref since we're in a repo)
	cmd = exec.Command(bin, "gauge", "-c", "report.json", "-o", "-", "--badge-type", "sparkline", "--history-auto")
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge sparkline with history-auto failed: %v\n%s", err, output)
	}

	if !strings.Contains(string(output), "<svg") {
		t.Error("output should be an SVG")
	}

	// Verify the default ref was created
	cmd = exec.Command("git", "show-ref", "--verify", "refs/confvis/history")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Error("default history ref should be created with --history-auto in a git repo")
	}
}

func TestGauge_Sparkline_HistoryAuto_NotInGitRepo(t *testing.T) {
	bin := buildBinary(t)

	// Create a temporary directory (NOT a git repo)
	tmpDir := t.TempDir()

	// Create sample report
	reportPath := filepath.Join(tmpDir, "report.json")
	if err := os.WriteFile(reportPath, []byte(`{"title": "Test", "score": 85, "threshold": 75}`), 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Run with --history-auto (should use file since we're not in a repo)
	cmd := exec.Command(bin, "gauge", "-c", "report.json", "-o", "-", "--badge-type", "sparkline", "--history-auto")
	cmd.Dir = tmpDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge sparkline with history-auto failed: %v\n%s", err, output)
	}

	if !strings.Contains(string(output), "<svg") {
		t.Error("output should be an SVG")
	}

	// Verify the default history file was created
	historyPath := filepath.Join(tmpDir, ".confvis-history.jsonl")
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		t.Error("default history file should be created with --history-auto outside a git repo")
	}

	// Verify the file has an entry
	historyData, err := os.ReadFile(historyPath)
	if err != nil {
		t.Fatalf("reading history file: %v", err)
	}
	if !strings.Contains(string(historyData), `"score":85`) {
		t.Error("history file should contain score entry")
	}
}

func TestAggregate_Basic(t *testing.T) {
	bin := buildBinary(t)

	outputDir := t.TempDir()

	cmd := exec.Command(bin, "aggregate", "-c", "../../testdata/sample.json", "-c", "../../testdata/sample_failing.json", "-o", outputDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("aggregate failed: %v\n%s", err, output)
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

	// Read dashboard and verify content
	dashboardContent, err := os.ReadFile(dashboardPath)
	if err != nil {
		t.Fatalf("reading dashboard: %v", err)
	}

	dashStr := string(dashboardContent)
	if !strings.Contains(dashStr, "Aggregate Confidence Report") {
		t.Error("dashboard should contain aggregate title")
	}
	if !strings.Contains(dashStr, "Code Quality Report") {
		t.Error("dashboard should contain component titles")
	}
}

func TestAggregate_WithWeights(t *testing.T) {
	bin := buildBinary(t)

	outputDir := t.TempDir()

	// Weight sample.json (85) more heavily than sample_failing.json (62)
	cmd := exec.Command(bin, "aggregate", "-c", "../../testdata/sample.json:80", "-c", "../../testdata/sample_failing.json:20", "-o", outputDir, "-v")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("aggregate with weights failed: %v\n%s", err, output)
	}

	// Weighted average: (85*80 + 62*20) / 100 = (6800 + 1240) / 100 = 80.4 ≈ 80
	// Should show the aggregate score in verbose output
	outputStr := string(output)
	if !strings.Contains(outputStr, "weight: 80") || !strings.Contains(outputStr, "weight: 20") {
		t.Error("verbose output should show weights")
	}
}

func TestAggregate_GlobPattern(t *testing.T) {
	bin := buildBinary(t)

	outputDir := t.TempDir()

	// Use glob pattern to match all JSON files in testdata
	cmd := exec.Command(bin, "aggregate", "-c", "../../testdata/sample*.json", "-o", outputDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("aggregate with glob failed: %v\n%s", err, output)
	}

	// Verify dashboard was created
	dashboardPath := filepath.Join(outputDir, "dashboard", "index.html")
	if _, err := os.Stat(dashboardPath); os.IsNotExist(err) {
		t.Error("dashboard/index.html was not created")
	}
}

func TestAggregate_DarkMode(t *testing.T) {
	bin := buildBinary(t)

	outputDir := t.TempDir()

	cmd := exec.Command(bin, "aggregate", "-c", "../../testdata/sample.json", "-o", outputDir, "--dark")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("aggregate dark mode failed: %v\n%s", err, output)
	}

	// Read dashboard and verify dark mode class
	dashboardPath := filepath.Join(outputDir, "dashboard", "index.html")
	dashboardContent, err := os.ReadFile(dashboardPath)
	if err != nil {
		t.Fatalf("reading dashboard: %v", err)
	}

	if !strings.Contains(string(dashboardContent), `class="dark"`) {
		t.Error("dashboard should have dark mode class")
	}
}

func TestAggregate_FailUnder_Pass(t *testing.T) {
	bin := buildBinary(t)

	outputDir := t.TempDir()

	// Average of 85 and 62 is ~73 or 74 depending on rounding
	cmd := exec.Command(bin, "aggregate", "-c", "../../testdata/sample.json", "-c", "../../testdata/sample_failing.json", "-o", outputDir, "--fail-under", "70")

	if err := cmd.Run(); err != nil {
		t.Fatalf("expected exit 0 for aggregate score above threshold, got error: %v", err)
	}
}

func TestAggregate_FailUnder_Fail(t *testing.T) {
	bin := buildBinary(t)

	outputDir := t.TempDir()

	// Average of 85 and 62 is ~73 or 74
	cmd := exec.Command(bin, "aggregate", "-c", "../../testdata/sample.json", "-c", "../../testdata/sample_failing.json", "-o", outputDir, "--fail-under", "80")

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit 1 for aggregate score below threshold, got exit 0")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T", err)
	}

	if exitErr.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
	}
}

func TestAggregate_IndividualBadges(t *testing.T) {
	bin := buildBinary(t)

	outputDir := t.TempDir()

	cmd := exec.Command(bin, "aggregate", "-c", "../../testdata/sample.json", "-c", "../../testdata/sample_failing.json", "-o", outputDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("aggregate failed: %v\n%s", err, output)
	}

	// Should create individual badges with sanitized names
	// sample.json has title "Code Quality Report"
	codeBadgePath := filepath.Join(outputDir, "code-quality-report.svg")
	if _, err := os.Stat(codeBadgePath); os.IsNotExist(err) {
		t.Error("individual badge for 'Code Quality Report' was not created")
	}

	// sample_failing.json also has title "Code Quality Report"
	// (since they both have same title, only one file gets created)
}

func TestAggregate_NoReports(t *testing.T) {
	bin := buildBinary(t)

	outputDir := t.TempDir()

	// Use a glob pattern that matches nothing
	cmd := exec.Command(bin, "aggregate", "-c", "../../testdata/nonexistent*.json", "-o", outputDir)

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error when no reports found")
	}
}

func TestAggregate_MissingOutput(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "aggregate", "-c", "../../testdata/sample.json")

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error when output flag is missing")
	}
}

func TestAggregate_MissingConfig(t *testing.T) {
	bin := buildBinary(t)

	outputDir := t.TempDir()

	cmd := exec.Command(bin, "aggregate", "-o", outputDir)

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error when config flag is missing")
	}
}
