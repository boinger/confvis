package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boinger/confvis/internal/confidence"
)

// Tests for sanitizeFilename

func TestSanitizeFilename_Basic(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Test Report", "test-report"},
		{"simple", "simple"},
		{"UPPERCASE", "uppercase"},
		{"already-dashed", "already-dashed"},
		{"with_underscore", "with_underscore"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeFilename_EmptyString(t *testing.T) {
	got := sanitizeFilename("")
	if got != "" {
		t.Errorf("sanitizeFilename(\"\") = %q, want \"\"", got)
	}
}

func TestSanitizeFilename_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"exclamation", "Hello!", "hello"},
		{"at sign", "test@example", "testexample"},
		{"hash", "test#1", "test1"},
		{"dollar", "$100", "100"},
		{"percent", "100%", "100"},
		{"caret", "test^2", "test2"},
		{"ampersand", "A&B", "ab"},
		{"asterisk", "star*", "star"},
		{"parens", "(test)", "test"},
		{"braces", "{config}", "config"},
		{"brackets", "[0]", "0"},
		{"plus", "a+b", "ab"},
		{"equals", "a=b", "ab"},
		{"pipe", "a|b", "ab"},
		{"backslash", "path\\to", "pathto"},
		{"forward slash", "path/to", "pathto"},
		{"colon", "c:drive", "cdrive"},
		{"semicolon", "a;b", "ab"},
		{"quotes", `"test"`, "test"},
		{"single quotes", "'test'", "test"},
		{"angle brackets", "<test>", "test"},
		{"comma", "a,b", "ab"},
		{"question mark", "test?", "test"},
		{"tilde", "~test", "test"},
		{"backtick", "`code`", "code"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeFilename_MultipleSpaces(t *testing.T) {
	got := sanitizeFilename("test   multiple   spaces")
	want := "test---multiple---spaces"
	if got != want {
		t.Errorf("sanitizeFilename(\"test   multiple   spaces\") = %q, want %q", got, want)
	}
}

func TestSanitizeFilename_ConsecutiveSpecialChars(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"!!!test", "test"},
		{"test!!!", "test"},
		{"te!!st", "test"},
		{"@#$%^&*", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeFilename_VeryLongString(t *testing.T) {
	// Create a very long string
	longInput := ""
	for i := 0; i < 500; i++ {
		longInput += "a"
	}

	got := sanitizeFilename(longInput)
	// Should preserve all 500 'a' characters
	if len(got) != 500 {
		t.Errorf("sanitizeFilename with 500 chars should produce 500 chars, got %d", len(got))
	}
}

func TestSanitizeFilename_NumbersOnly(t *testing.T) {
	got := sanitizeFilename("12345")
	if got != "12345" {
		t.Errorf("sanitizeFilename(\"12345\") = %q, want \"12345\"", got)
	}
}

func TestSanitizeFilename_MixedAlphaNumeric(t *testing.T) {
	got := sanitizeFilename("Test123Report456")
	if got != "test123report456" {
		t.Errorf("sanitizeFilename(\"Test123Report456\") = %q, want \"test123report456\"", got)
	}
}

func TestSanitizeFilename_LeadingTrailingSpecial(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  test  ", "--test--"},
		{"--test--", "--test--"},
		{"__test__", "__test__"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Tests for parseConfigsWithWeights

func TestParseConfigsWithWeights_SingleFile(t *testing.T) {
	results, err := parseConfigsWithWeights([]string{"../../testdata/sample.json"})
	if err != nil {
		t.Fatalf("parseConfigsWithWeights() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Default weight should be 100
	if results[0].Weight != 100 {
		t.Errorf("default weight should be 100, got %d", results[0].Weight)
	}

	// Check report was parsed
	if results[0].Report.Score != 85 {
		t.Errorf("report score should be 85, got %d", results[0].Report.Score)
	}

	// Check path is preserved
	if results[0].Path != "../../testdata/sample.json" {
		t.Errorf("path should be preserved, got %s", results[0].Path)
	}
}

func TestParseConfigsWithWeights_WithWeight(t *testing.T) {
	results, err := parseConfigsWithWeights([]string{"../../testdata/sample.json:60"})
	if err != nil {
		t.Fatalf("parseConfigsWithWeights() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Weight should be 60
	if results[0].Weight != 60 {
		t.Errorf("weight should be 60, got %d", results[0].Weight)
	}

	// Path should not include weight suffix
	if results[0].Path != "../../testdata/sample.json" {
		t.Errorf("path should not include weight suffix, got %s", results[0].Path)
	}
}

func TestParseConfigsWithWeights_MultipleFiles(t *testing.T) {
	configs := []string{
		"../../testdata/sample.json:60",
		"../../testdata/sample_failing.json:40",
	}

	results, err := parseConfigsWithWeights(configs)
	if err != nil {
		t.Fatalf("parseConfigsWithWeights() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Weight != 60 {
		t.Errorf("first weight should be 60, got %d", results[0].Weight)
	}

	if results[1].Weight != 40 {
		t.Errorf("second weight should be 40, got %d", results[1].Weight)
	}
}

func TestParseConfigsWithWeights_MixedWeightedAndNonWeighted(t *testing.T) {
	configs := []string{
		"../../testdata/sample.json:75",
		"../../testdata/sample_failing.json", // No weight = default 100
	}

	results, err := parseConfigsWithWeights(configs)
	if err != nil {
		t.Fatalf("parseConfigsWithWeights() error = %v", err)
	}

	if results[0].Weight != 75 {
		t.Errorf("first weight should be 75, got %d", results[0].Weight)
	}

	if results[1].Weight != 100 {
		t.Errorf("second weight (default) should be 100, got %d", results[1].Weight)
	}
}

func TestParseConfigsWithWeights_ZeroWeight(t *testing.T) {
	results, err := parseConfigsWithWeights([]string{"../../testdata/sample.json:0"})
	if err != nil {
		t.Fatalf("parseConfigsWithWeights() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Weight != 0 {
		t.Errorf("weight should be 0, got %d", results[0].Weight)
	}
}

func TestParseConfigsWithWeights_NegativeWeight(t *testing.T) {
	// Negative weight is parsed as a valid integer
	// The function doesn't validate if weight is valid business-wise
	results, err := parseConfigsWithWeights([]string{"../../testdata/sample.json:-50"})
	if err != nil {
		t.Fatalf("parseConfigsWithWeights() error = %v", err)
	}

	if results[0].Weight != -50 {
		t.Errorf("weight should be -50, got %d", results[0].Weight)
	}
}

func TestParseConfigsWithWeights_InvalidWeight(t *testing.T) {
	// Non-numeric suffix should be treated as part of path (and fail on file not found)
	_, err := parseConfigsWithWeights([]string{"../../testdata/sample.json:notanumber"})
	// Since "notanumber" is not a valid int, it stays part of the path
	// and the file "../../testdata/sample.json:notanumber" doesn't exist
	if err == nil {
		t.Error("expected error for invalid (non-existent) path")
	}
}

func TestParseConfigsWithWeights_GlobPattern(t *testing.T) {
	results, err := parseConfigsWithWeights([]string{"../../testdata/sample*.json"})
	if err != nil {
		t.Fatalf("parseConfigsWithWeights() error = %v", err)
	}

	// Should match both sample.json and sample_failing.json
	if len(results) < 2 {
		t.Errorf("expected at least 2 results from glob, got %d", len(results))
	}
}

func TestParseConfigsWithWeights_GlobPatternWithWeight(t *testing.T) {
	results, err := parseConfigsWithWeights([]string{"../../testdata/sample*.json:50"})
	if err != nil {
		t.Fatalf("parseConfigsWithWeights() error = %v", err)
	}

	// All matched files should have the same weight
	for _, r := range results {
		if r.Weight != 50 {
			t.Errorf("all glob results should have weight 50, got %d for %s", r.Weight, r.Path)
		}
	}
}

func TestParseConfigsWithWeights_NonExistentFile(t *testing.T) {
	_, err := parseConfigsWithWeights([]string{"nonexistent.json"})
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestParseConfigsWithWeights_EmptyConfigs(t *testing.T) {
	results, err := parseConfigsWithWeights([]string{})
	if err != nil {
		t.Fatalf("parseConfigsWithWeights() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestParseConfigsWithWeights_InvalidJSON(t *testing.T) {
	// Create a temp file with invalid JSON
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("writing invalid file: %v", err)
	}

	_, err := parseConfigsWithWeights([]string{invalidPath})
	if err == nil {
		t.Error("expected error for invalid JSON file")
	}
}

func TestParseConfigsWithWeights_YAMLFile(t *testing.T) {
	results, err := parseConfigsWithWeights([]string{"../../testdata/sample.yaml"})
	if err != nil {
		t.Fatalf("parseConfigsWithWeights() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// YAML file should be parsed correctly
	if results[0].Report.Score != 85 {
		t.Errorf("YAML report score should be 85, got %d", results[0].Report.Score)
	}
}

func TestParseConfigsWithWeights_YAMLFileWithWeight(t *testing.T) {
	results, err := parseConfigsWithWeights([]string{"../../testdata/sample.yaml:25"})
	if err != nil {
		t.Fatalf("parseConfigsWithWeights() error = %v", err)
	}

	if results[0].Weight != 25 {
		t.Errorf("weight should be 25, got %d", results[0].Weight)
	}
}

func TestParseConfigsWithWeights_LargeWeight(t *testing.T) {
	results, err := parseConfigsWithWeights([]string{"../../testdata/sample.json:999999"})
	if err != nil {
		t.Fatalf("parseConfigsWithWeights() error = %v", err)
	}

	if results[0].Weight != 999999 {
		t.Errorf("weight should be 999999, got %d", results[0].Weight)
	}
}

func TestParseConfigsWithWeights_ColonInPathNotWeight(t *testing.T) {
	// Create a temp file with JSON content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	testContent := `{"title": "Test", "score": 75, "threshold": 75}`
	if err := os.WriteFile(testFile, []byte(testContent), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	// Path with trailing colon followed by non-numeric should treat whole thing as path
	// This tests that "path:abc" where "abc" is not a number keeps the full path
	// In this case, the file doesn't exist so it should error
	_, err := parseConfigsWithWeights([]string{testFile + ":abc"})
	if err == nil {
		t.Error("expected error for non-existent file (path:abc)")
	}
}

func TestParseConfigsWithWeights_ReportFields(t *testing.T) {
	results, err := parseConfigsWithWeights([]string{"../../testdata/sample.json"})
	if err != nil {
		t.Fatalf("parseConfigsWithWeights() error = %v", err)
	}

	report := results[0].Report

	// Verify report was fully parsed
	if report.Title != "Code Quality Report" {
		t.Errorf("title should be 'Code Quality Report', got %q", report.Title)
	}

	if report.Threshold != 75 {
		t.Errorf("threshold should be 75, got %d", report.Threshold)
	}

	if len(report.Factors) != 4 {
		t.Errorf("should have 4 factors, got %d", len(report.Factors))
	}
}

// Tests for writeMarkdown

func TestWriteMarkdown_BasicReport(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test Report",
		Score:     85,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := writeMarkdown(&buf, report, nil, 0)
	if err != nil {
		t.Fatalf("writeMarkdown() error = %v", err)
	}

	md := buf.String()

	// Should contain header with title, score, and PASS
	if !strings.Contains(md, "## Test Report: 85% (PASS)") {
		t.Error("markdown should contain header with title, score, and PASS status")
	}
}

func TestWriteMarkdown_FailingReport(t *testing.T) {
	report := &confidence.Report{
		Title:     "Failing Report",
		Score:     60,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := writeMarkdown(&buf, report, nil, 0)
	if err != nil {
		t.Fatalf("writeMarkdown() error = %v", err)
	}

	md := buf.String()

	// Should contain FAIL status
	if !strings.Contains(md, "(FAIL)") {
		t.Error("markdown should contain FAIL status")
	}
}

func TestWriteMarkdown_WithDescription(t *testing.T) {
	report := &confidence.Report{
		Title:       "Test Report",
		Score:       85,
		Threshold:   75,
		Description: "This is a test description.",
	}

	var buf bytes.Buffer
	err := writeMarkdown(&buf, report, nil, 0)
	if err != nil {
		t.Fatalf("writeMarkdown() error = %v", err)
	}

	md := buf.String()

	if !strings.Contains(md, "This is a test description.") {
		t.Error("markdown should contain description")
	}
}

func TestWriteMarkdown_WithFactors(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test Report",
		Score:     85,
		Threshold: 75,
		Factors: []confidence.Factor{
			{Name: "Coverage", Score: 90, Weight: 50},
			{Name: "Lint", Score: 80, Weight: 50},
		},
	}

	var buf bytes.Buffer
	err := writeMarkdown(&buf, report, nil, 0)
	if err != nil {
		t.Fatalf("writeMarkdown() error = %v", err)
	}

	md := buf.String()

	// Should contain table headers
	if !strings.Contains(md, "| Factor | Score | Weight |") {
		t.Error("markdown should contain table header")
	}

	// Should contain factor rows
	if !strings.Contains(md, "| Coverage | 90% | 50% |") {
		t.Error("markdown should contain Coverage factor")
	}
	if !strings.Contains(md, "| Lint | 80% | 50% |") {
		t.Error("markdown should contain Lint factor")
	}
}

func TestWriteMarkdown_WithFactorURLs(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test Report",
		Score:     85,
		Threshold: 75,
		Factors: []confidence.Factor{
			{Name: "Coverage", Score: 90, Weight: 50, URL: "https://example.com/coverage"},
			{Name: "Lint", Score: 80, Weight: 50}, // No URL
		},
	}

	var buf bytes.Buffer
	err := writeMarkdown(&buf, report, nil, 0)
	if err != nil {
		t.Fatalf("writeMarkdown() error = %v", err)
	}

	md := buf.String()

	// Factor with URL should be a markdown link
	if !strings.Contains(md, "[Coverage](https://example.com/coverage)") {
		t.Error("factor with URL should be a markdown link")
	}

	// Factor without URL should be plain text
	if !strings.Contains(md, "| Lint |") {
		t.Error("factor without URL should be plain text")
	}
}

func TestWriteMarkdown_WithBaseline(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test Report",
		Score:     85,
		Threshold: 75,
	}

	baseline := &confidence.Report{
		Title:     "Baseline",
		Score:     62,
		Threshold: 75,
	}

	delta := 23

	var buf bytes.Buffer
	err := writeMarkdown(&buf, report, baseline, delta)
	if err != nil {
		t.Fatalf("writeMarkdown() error = %v", err)
	}

	md := buf.String()

	// Should contain delta information
	if !strings.Contains(md, "[+23 from 62%]") {
		t.Error("markdown should contain delta from baseline")
	}
}

func TestWriteMarkdown_WithNegativeDelta(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test Report",
		Score:     62,
		Threshold: 75,
	}

	baseline := &confidence.Report{
		Title:     "Baseline",
		Score:     85,
		Threshold: 75,
	}

	delta := -23

	var buf bytes.Buffer
	err := writeMarkdown(&buf, report, baseline, delta)
	if err != nil {
		t.Fatalf("writeMarkdown() error = %v", err)
	}

	md := buf.String()

	// Negative delta should not have + prefix
	if !strings.Contains(md, "[-23 from 85%]") {
		t.Error("markdown should contain negative delta from baseline")
	}
}

func TestWriteMarkdown_CustomLabels(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test Report",
		Score:     85,
		Threshold: 75,
		PassLabel: "APPROVED",
	}

	var buf bytes.Buffer
	err := writeMarkdown(&buf, report, nil, 0)
	if err != nil {
		t.Fatalf("writeMarkdown() error = %v", err)
	}

	md := buf.String()

	// Should use custom pass label
	if !strings.Contains(md, "(APPROVED)") {
		t.Error("markdown should use custom pass label")
	}
}

func TestWriteMarkdown_CustomFailLabel(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test Report",
		Score:     60,
		Threshold: 75,
		FailLabel: "NEEDS IMPROVEMENT",
	}

	var buf bytes.Buffer
	err := writeMarkdown(&buf, report, nil, 0)
	if err != nil {
		t.Fatalf("writeMarkdown() error = %v", err)
	}

	md := buf.String()

	// Should use custom fail label
	if !strings.Contains(md, "(NEEDS IMPROVEMENT)") {
		t.Error("markdown should use custom fail label")
	}
}

func TestWriteMarkdown_NoFactors(t *testing.T) {
	report := &confidence.Report{
		Title:     "Simple Report",
		Score:     85,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := writeMarkdown(&buf, report, nil, 0)
	if err != nil {
		t.Fatalf("writeMarkdown() error = %v", err)
	}

	md := buf.String()

	// Should not have factors table
	if strings.Contains(md, "| Factor |") {
		t.Error("markdown should not contain factors table when no factors")
	}
}

func TestWriteMarkdown_EdgeCases(t *testing.T) {
	t.Run("zero score", func(t *testing.T) {
		report := &confidence.Report{
			Title:     "Zero Score",
			Score:     0,
			Threshold: 75,
		}

		var buf bytes.Buffer
		err := writeMarkdown(&buf, report, nil, 0)
		if err != nil {
			t.Fatalf("writeMarkdown() error = %v", err)
		}

		if !strings.Contains(buf.String(), "0%") {
			t.Error("markdown should display 0%")
		}
	})

	t.Run("100 score", func(t *testing.T) {
		report := &confidence.Report{
			Title:     "Perfect Score",
			Score:     100,
			Threshold: 75,
		}

		var buf bytes.Buffer
		err := writeMarkdown(&buf, report, nil, 0)
		if err != nil {
			t.Fatalf("writeMarkdown() error = %v", err)
		}

		if !strings.Contains(buf.String(), "100%") {
			t.Error("markdown should display 100%")
		}
	})

	t.Run("zero delta", func(t *testing.T) {
		report := &confidence.Report{
			Title:     "Test",
			Score:     85,
			Threshold: 75,
		}

		baseline := &confidence.Report{
			Title:     "Baseline",
			Score:     85,
			Threshold: 75,
		}

		var buf bytes.Buffer
		err := writeMarkdown(&buf, report, baseline, 0)
		if err != nil {
			t.Fatalf("writeMarkdown() error = %v", err)
		}

		// Zero delta should still have + prefix (behavior of the function)
		if !strings.Contains(buf.String(), "[+0 from 85%]") {
			t.Error("markdown should show zero delta")
		}
	})
}

// Tests for generateBadge

func TestGenerateBadge_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	badgePath := filepath.Join(tmpDir, "badge.svg")

	report := &confidence.Report{
		Title:     "Test Report",
		Score:     85,
		Threshold: 75,
	}

	err := generateBadge(badgePath, report, false, false)
	if err != nil {
		t.Fatalf("generateBadge() error = %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(badgePath)
	if err != nil {
		t.Fatalf("reading badge file: %v", err)
	}

	svg := string(content)
	if !strings.Contains(svg, "<svg") {
		t.Error("badge should contain SVG content")
	}
	if !strings.Contains(svg, "85") {
		t.Error("badge should contain score")
	}
}

func TestGenerateBadge_DarkMode(t *testing.T) {
	tmpDir := t.TempDir()
	badgePath := filepath.Join(tmpDir, "badge.svg")

	report := &confidence.Report{
		Title:     "Test Report",
		Score:     85,
		Threshold: 75,
	}

	err := generateBadge(badgePath, report, true, false)
	if err != nil {
		t.Fatalf("generateBadge() error = %v", err)
	}

	content, err := os.ReadFile(badgePath)
	if err != nil {
		t.Fatalf("reading badge file: %v", err)
	}

	svg := string(content)
	// Dark mode should use dark background
	if !strings.Contains(svg, "#0d1117") {
		t.Error("dark mode badge should use dark background")
	}
}

func TestGenerateBadge_InvalidPath(t *testing.T) {
	// Non-existent directory
	badgePath := "/nonexistent/directory/badge.svg"

	report := &confidence.Report{
		Title:     "Test",
		Score:     85,
		Threshold: 75,
	}

	err := generateBadge(badgePath, report, false, false)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

// Tests for generateDashboard

func TestGenerateDashboard_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	dashPath := filepath.Join(tmpDir, "index.html")

	report := &confidence.Report{
		Title:     "Test Report",
		Score:     85,
		Threshold: 75,
	}

	err := generateDashboard(dashPath, report, false, false)
	if err != nil {
		t.Fatalf("generateDashboard() error = %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(dashPath)
	if err != nil {
		t.Fatalf("reading dashboard file: %v", err)
	}

	html := string(content)
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("dashboard should be valid HTML")
	}
	if !strings.Contains(html, "Test Report") {
		t.Error("dashboard should contain report title")
	}
}

func TestGenerateDashboard_DarkMode(t *testing.T) {
	tmpDir := t.TempDir()
	dashPath := filepath.Join(tmpDir, "index.html")

	report := &confidence.Report{
		Title:     "Test Report",
		Score:     85,
		Threshold: 75,
	}

	err := generateDashboard(dashPath, report, true, false)
	if err != nil {
		t.Fatalf("generateDashboard() error = %v", err)
	}

	content, err := os.ReadFile(dashPath)
	if err != nil {
		t.Fatalf("reading dashboard file: %v", err)
	}

	html := string(content)
	if !strings.Contains(html, `class="dark"`) {
		t.Error("dark mode dashboard should have dark class")
	}
}

func TestGenerateDashboard_WithFactors(t *testing.T) {
	tmpDir := t.TempDir()
	dashPath := filepath.Join(tmpDir, "index.html")

	report := &confidence.Report{
		Title:     "Test Report",
		Score:     85,
		Threshold: 75,
		Factors: []confidence.Factor{
			{Name: "Coverage", Score: 90, Weight: 50},
			{Name: "Lint", Score: 80, Weight: 50},
		},
	}

	err := generateDashboard(dashPath, report, false, false)
	if err != nil {
		t.Fatalf("generateDashboard() error = %v", err)
	}

	content, err := os.ReadFile(dashPath)
	if err != nil {
		t.Fatalf("reading dashboard file: %v", err)
	}

	html := string(content)
	if !strings.Contains(html, "Coverage") {
		t.Error("dashboard should contain factor names")
	}
}

func TestGenerateDashboard_InvalidPath(t *testing.T) {
	// Non-existent directory
	dashPath := "/nonexistent/directory/index.html"

	report := &confidence.Report{
		Title:     "Test",
		Score:     85,
		Threshold: 75,
	}

	err := generateDashboard(dashPath, report, false, false)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

// Tests for generateAggregateBadge

func TestGenerateAggregateBadge_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	badgePath := filepath.Join(tmpDir, "aggregate.svg")

	report := &confidence.Report{
		Title:     "Aggregate",
		Score:     80,
		Threshold: 75,
	}

	err := generateAggregateBadge(badgePath, report, false, false)
	if err != nil {
		t.Fatalf("generateAggregateBadge() error = %v", err)
	}

	content, err := os.ReadFile(badgePath)
	if err != nil {
		t.Fatalf("reading badge file: %v", err)
	}

	svg := string(content)
	if !strings.Contains(svg, "<svg") {
		t.Error("badge should contain SVG content")
	}
}

func TestGenerateAggregateBadge_DarkMode(t *testing.T) {
	tmpDir := t.TempDir()
	badgePath := filepath.Join(tmpDir, "aggregate.svg")

	report := &confidence.Report{
		Title:     "Aggregate",
		Score:     80,
		Threshold: 75,
	}

	err := generateAggregateBadge(badgePath, report, true, false)
	if err != nil {
		t.Fatalf("generateAggregateBadge() error = %v", err)
	}

	content, err := os.ReadFile(badgePath)
	if err != nil {
		t.Fatalf("reading badge file: %v", err)
	}

	svg := string(content)
	if !strings.Contains(svg, "#0d1117") {
		t.Error("dark mode badge should use dark background")
	}
}

func TestGenerateAggregateBadge_InvalidPath(t *testing.T) {
	badgePath := "/nonexistent/directory/aggregate.svg"

	report := &confidence.Report{
		Title:     "Aggregate",
		Score:     80,
		Threshold: 75,
	}

	err := generateAggregateBadge(badgePath, report, false, false)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

// Tests for generateMultiDashboard

func TestGenerateMultiDashboard_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	dashPath := filepath.Join(tmpDir, "index.html")

	reports := []reportWithWeight{
		{
			Report: &confidence.Report{
				Title:     "Report 1",
				Score:     90,
				Threshold: 75,
			},
			Weight: 60,
			Path:   "report1.json",
		},
		{
			Report: &confidence.Report{
				Title:     "Report 2",
				Score:     80,
				Threshold: 75,
			},
			Weight: 40,
			Path:   "report2.json",
		},
	}

	aggregate := &confidence.Report{
		Title:     "Aggregate",
		Score:     86,
		Threshold: 75,
	}

	err := generateMultiDashboard(dashPath, reports, aggregate, false, false)
	if err != nil {
		t.Fatalf("generateMultiDashboard() error = %v", err)
	}

	content, err := os.ReadFile(dashPath)
	if err != nil {
		t.Fatalf("reading dashboard file: %v", err)
	}

	html := string(content)
	if !strings.Contains(html, "Aggregate Confidence Report") {
		t.Error("dashboard should contain aggregate title")
	}
	if !strings.Contains(html, "Report 1") {
		t.Error("dashboard should contain first report title")
	}
	if !strings.Contains(html, "Report 2") {
		t.Error("dashboard should contain second report title")
	}
}

func TestGenerateMultiDashboard_DarkMode(t *testing.T) {
	tmpDir := t.TempDir()
	dashPath := filepath.Join(tmpDir, "index.html")

	reports := []reportWithWeight{
		{
			Report: &confidence.Report{
				Title:     "Report 1",
				Score:     85,
				Threshold: 75,
			},
			Weight: 100,
			Path:   "report1.json",
		},
	}

	aggregate := &confidence.Report{
		Title:     "Aggregate",
		Score:     85,
		Threshold: 75,
	}

	err := generateMultiDashboard(dashPath, reports, aggregate, true, false)
	if err != nil {
		t.Fatalf("generateMultiDashboard() error = %v", err)
	}

	content, err := os.ReadFile(dashPath)
	if err != nil {
		t.Fatalf("reading dashboard file: %v", err)
	}

	html := string(content)
	if !strings.Contains(html, `class="dark"`) {
		t.Error("dark mode dashboard should have dark class")
	}
}

func TestGenerateMultiDashboard_InvalidPath(t *testing.T) {
	dashPath := "/nonexistent/directory/index.html"

	reports := []reportWithWeight{}
	aggregate := &confidence.Report{
		Title:     "Aggregate",
		Score:     0,
		Threshold: 75,
	}

	err := generateMultiDashboard(dashPath, reports, aggregate, false, false)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}
