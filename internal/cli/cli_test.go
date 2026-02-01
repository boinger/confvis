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
