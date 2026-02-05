package gauge

import (
	"bytes"
	"strings"
	"testing"

	"github.com/boinger/confvis/internal/confidence"
)

// intPtrG is a test helper for gauge_test.go.
func intPtrG(i int) *int { return &i }

func TestGenerate_ContainsExpectedElements(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtrG(85),
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := Generate(&buf, report, defaultOptions())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	svg := buf.String()

	// Should contain SVG element
	if !strings.Contains(svg, "<svg") {
		t.Error("output should contain <svg element")
	}

	// Should contain the score
	if !strings.Contains(svg, ">85<") {
		t.Error("output should contain score value 85")
	}

	// Should contain PASS since 85 >= 75
	if !strings.Contains(svg, "PASS") {
		t.Error("output should contain PASS indicator")
	}

	// Should contain path elements for arcs
	if !strings.Contains(svg, "<path") {
		t.Error("output should contain path elements for gauge arcs")
	}
}

func TestGenerate_FailingReport(t *testing.T) {
	report := &confidence.Report{
		Title:     "Failing Test",
		Score:     intPtrG(60),
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := Generate(&buf, report, defaultOptions())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	svg := buf.String()

	// Should contain FAIL since 60 < 75
	if !strings.Contains(svg, "FAIL") {
		t.Error("output should contain FAIL indicator")
	}
}

func TestGenerate_DarkMode(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtrG(85),
		Threshold: 75,
	}

	opts := defaultOptions()
	opts.DarkMode = true

	var buf bytes.Buffer
	err := Generate(&buf, report, opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	svg := buf.String()

	// Dark mode uses #0d1117 background
	if !strings.Contains(svg, "#0d1117") {
		t.Error("dark mode output should contain dark background color")
	}
}

func TestGenerate_CustomDimensions(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtrG(75),
		Threshold: 75,
	}

	opts := Options{
		Width:  300,
		Height: 180,
	}

	var buf bytes.Buffer
	err := Generate(&buf, report, opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	svg := buf.String()

	if !strings.Contains(svg, `width="300"`) {
		t.Error("output should have width 300")
	}
	if !strings.Contains(svg, `height="180"`) {
		t.Error("output should have height 180")
	}
}

func TestScoreColor(t *testing.T) {
	scheme := gitHubLight()

	// Test with default thresholds (75, 50)
	tests := []struct {
		score int
		want  string
	}{
		{100, scheme.Success},
		{75, scheme.Success},
		{74, scheme.Warning},
		{50, scheme.Warning},
		{49, scheme.Danger},
		{0, scheme.Danger},
	}

	for _, tt := range tests {
		got := scheme.ScoreColor(tt.score, 75, 50)
		if got != tt.want {
			t.Errorf("ScoreColor(%d, 75, 50) = %q, want %q", tt.score, got, tt.want)
		}
	}
}

func TestScoreColor_CustomThresholds(t *testing.T) {
	scheme := gitHubLight()

	// Test with custom thresholds (90, 70)
	tests := []struct {
		score int
		want  string
	}{
		{100, scheme.Success},
		{90, scheme.Success},
		{89, scheme.Warning},
		{70, scheme.Warning},
		{69, scheme.Danger},
		{0, scheme.Danger},
	}

	for _, tt := range tests {
		got := scheme.ScoreColor(tt.score, 90, 70)
		if got != tt.want {
			t.Errorf("ScoreColor(%d, 90, 70) = %q, want %q", tt.score, got, tt.want)
		}
	}
}

func TestGenerateToString(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtrG(85),
		Threshold: 75,
	}

	svg, err := GenerateToString(report, defaultOptions())
	if err != nil {
		t.Fatalf("GenerateToString() error = %v", err)
	}

	if !strings.Contains(svg, "<svg") {
		t.Error("output should contain SVG content")
	}
}

func TestGetColorScheme(t *testing.T) {
	tests := []struct {
		style      string
		darkMode   bool
		wantBg     string
		wantScheme string
	}{
		{"github", false, "#ffffff", "GitHubLight"},
		{"github", true, "#0d1117", "GitHubDark"},
		{"minimal", false, "#fafafa", "Minimal"},
		{"minimal", true, "#1a1a1a", "MinimalDark"},
		{"corporate", false, "#f5f5f5", "Corporate"},
		{"corporate", true, "#141414", "CorporateDark"},
		{"high-contrast", false, "#ffffff", "HighContrast"},
		{"high-contrast", true, "#000000", "HighContrastDark"},
		{"unknown", false, "#ffffff", "GitHubLight (fallback)"},
	}

	for _, tt := range tests {
		name := tt.style
		if tt.darkMode {
			name += "-dark"
		}
		t.Run(name, func(t *testing.T) {
			scheme := GetColorScheme(tt.style, tt.darkMode)
			if scheme.Background != tt.wantBg {
				t.Errorf("GetColorScheme(%q, %v).Background = %q, want %q",
					tt.style, tt.darkMode, scheme.Background, tt.wantBg)
			}
		})
	}
}

func TestStyleNames(t *testing.T) {
	names := styleNames()
	expected := []string{"github", "minimal", "corporate", "high-contrast"}

	if len(names) != len(expected) {
		t.Errorf("styleNames() returned %d names, want %d", len(names), len(expected))
	}

	for i, name := range expected {
		if names[i] != name {
			t.Errorf("styleNames()[%d] = %q, want %q", i, names[i], name)
		}
	}
}

func TestGenerate_TransparentBG(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtrG(85),
		Threshold: 75,
	}

	// With TransparentBG, output should not contain a background rect
	opts := defaultOptions()
	opts.TransparentBG = true

	var buf bytes.Buffer
	err := Generate(&buf, report, opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	svg := buf.String()

	if strings.Contains(svg, `fill:#ffffff"`) {
		t.Error("transparent BG output should not contain background fill rect")
	}
	if !strings.Contains(svg, "<path") {
		t.Error("transparent BG output should still contain gauge arcs")
	}
}

func TestGenerate_ViewBox(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtrG(85),
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := Generate(&buf, report, defaultOptions())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	svg := buf.String()

	if !strings.Contains(svg, `viewBox="0 0 200 120"`) {
		t.Error("SVG should contain viewBox attribute for proper scaling")
	}
}

func TestGenerate_OpaqueBackground(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtrG(85),
		Threshold: 75,
	}

	// Default (TransparentBG=false) should include background rect
	var buf bytes.Buffer
	err := Generate(&buf, report, defaultOptions())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	svg := buf.String()

	if !strings.Contains(svg, "<rect") {
		t.Error("default output should contain background rect")
	}
}

func TestGenerate_ScoreZero(t *testing.T) {
	report := &confidence.Report{
		Title:     "Zero Score",
		Score:     intPtrG(0),
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := Generate(&buf, report, defaultOptions())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, "<svg") {
		t.Error("output should contain SVG element")
	}
	if !strings.Contains(svg, "FAIL") {
		t.Error("score 0 should show FAIL")
	}
}

func TestGenerate_ScoreZero_TransparentBG(t *testing.T) {
	report := &confidence.Report{
		Title:     "Zero Score Transparent",
		Score:     intPtrG(0),
		Threshold: 75,
	}

	opts := defaultOptions()
	opts.TransparentBG = true

	var buf bytes.Buffer
	err := Generate(&buf, report, opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, "<svg") {
		t.Error("output should contain SVG element")
	}
	if !strings.Contains(svg, "<path") {
		t.Error("output should still contain gauge arcs")
	}
}

func TestGenerate_Score100(t *testing.T) {
	report := &confidence.Report{
		Title:     "Perfect Score",
		Score:     intPtrG(100),
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := Generate(&buf, report, defaultOptions())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, ">100<") {
		t.Error("output should contain score value 100")
	}
	if !strings.Contains(svg, "PASS") {
		t.Error("score 100 should show PASS")
	}
	if !strings.Contains(svg, `viewBox="0 0 200 120"`) {
		t.Error("SVG should have viewBox")
	}
}

func TestGenerate_DarkMode_TransparentBG(t *testing.T) {
	report := &confidence.Report{
		Title:     "Dark Transparent",
		Score:     intPtrG(85),
		Threshold: 75,
	}

	opts := defaultOptions()
	opts.DarkMode = true
	opts.TransparentBG = true

	var buf bytes.Buffer
	err := Generate(&buf, report, opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, "<svg") {
		t.Error("output should contain SVG element")
	}
	if !strings.Contains(svg, "<path") {
		t.Error("output should contain gauge arcs")
	}
}

func TestGenerate_DarkMode_CustomDimensions(t *testing.T) {
	report := &confidence.Report{
		Title:     "Dark Custom",
		Score:     intPtrG(50),
		Threshold: 75,
	}

	opts := Options{
		ColorOptions: ColorOptions{DarkMode: true, Style: "high-contrast"},
		Width:        300,
		Height:       200,
	}

	var buf bytes.Buffer
	err := Generate(&buf, report, opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, `width="300"`) {
		t.Error("output should have width 300")
	}
	if !strings.Contains(svg, "FAIL") {
		t.Error("score 50 with threshold 75 should show FAIL")
	}
}

func TestGenerate_CustomColorThresholds(t *testing.T) {
	report := &confidence.Report{
		Title:     "Custom Thresholds",
		Score:     intPtrG(60),
		Threshold: 50,
	}

	opts := Options{
		ColorOptions: ColorOptions{GreenAbove: 90, YellowAbove: 70},
		Width:        200,
		Height:       120,
	}

	var buf bytes.Buffer
	err := Generate(&buf, report, opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	svg := buf.String()
	if !strings.Contains(svg, "PASS") {
		t.Error("score 60 with threshold 50 should show PASS")
	}
}

func TestGenerateToString_WithOptions(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtrG(50),
		Threshold: 75,
	}

	opts := Options{
		ColorOptions: ColorOptions{DarkMode: true, Style: "minimal"},
		Width:        300,
		Height:       200,
	}

	svg, err := GenerateToString(report, opts)
	if err != nil {
		t.Fatalf("GenerateToString() error = %v", err)
	}

	// Should contain dark mode background
	if !strings.Contains(svg, "#1a1a1a") {
		t.Error("dark mode should use minimal dark background")
	}
	if !strings.Contains(svg, "FAIL") {
		t.Error("failing report should show FAIL")
	}
}
