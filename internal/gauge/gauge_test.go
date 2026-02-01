package gauge

import (
	"bytes"
	"strings"
	"testing"

	"github.com/boinger/confvis/internal/confidence"
)

func TestGenerate_ContainsExpectedElements(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     85,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := Generate(&buf, report, DefaultOptions())
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
		Score:     60,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := Generate(&buf, report, DefaultOptions())
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
		Score:     85,
		Threshold: 75,
	}

	opts := DefaultOptions()
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
		Score:     75,
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
	scheme := GitHubLight()

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
		got := scheme.ScoreColor(tt.score)
		if got != tt.want {
			t.Errorf("ScoreColor(%d) = %q, want %q", tt.score, got, tt.want)
		}
	}
}

func TestGenerateToString(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     85,
		Threshold: 75,
	}

	svg, err := GenerateToString(report, DefaultOptions())
	if err != nil {
		t.Fatalf("GenerateToString() error = %v", err)
	}

	if !strings.Contains(svg, "<svg") {
		t.Error("output should contain SVG content")
	}
}
