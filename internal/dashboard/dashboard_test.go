package dashboard

import (
	"bytes"
	"strings"
	"testing"

	"github.com/boinger/confvis/internal/confidence"
)

func TestGenerate_ValidHTML(t *testing.T) {
	report := &confidence.Report{
		Title:       "Test Report",
		Score:       85,
		Threshold:   75,
		Description: "A test description",
		Factors: []confidence.Factor{
			{Name: "Factor 1", Score: 90, Weight: 50, Description: "First factor"},
			{Name: "Factor 2", Score: 80, Weight: 50},
		},
	}

	var buf bytes.Buffer
	err := Generate(&buf, report, Options{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := buf.String()

	// Check basic HTML structure
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("output should contain DOCTYPE")
	}
	if !strings.Contains(html, "<html") {
		t.Error("output should contain html element")
	}

	// Check report content
	if !strings.Contains(html, "Test Report") {
		t.Error("output should contain report title")
	}
	if !strings.Contains(html, "A test description") {
		t.Error("output should contain report description")
	}

	// Check factors are rendered
	if !strings.Contains(html, "Factor 1") {
		t.Error("output should contain Factor 1")
	}
	if !strings.Contains(html, "Factor 2") {
		t.Error("output should contain Factor 2")
	}
	if !strings.Contains(html, "First factor") {
		t.Error("output should contain factor description")
	}

	// Check pass badge (85 >= 75)
	if !strings.Contains(html, "badge-pass") {
		t.Error("output should contain pass badge")
	}

	// Check embedded SVG
	if !strings.Contains(html, "<svg") {
		t.Error("output should contain embedded SVG gauge")
	}
}

func TestGenerate_FailingReport(t *testing.T) {
	report := &confidence.Report{
		Title:     "Failing Report",
		Score:     60,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := Generate(&buf, report, Options{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := buf.String()

	// Check fail badge (60 < 75)
	if !strings.Contains(html, "badge-fail") {
		t.Error("output should contain fail badge")
	}
}

func TestGenerate_DarkMode(t *testing.T) {
	report := &confidence.Report{
		Title:     "Dark Mode Test",
		Score:     85,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := Generate(&buf, report, Options{DarkMode: true})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := buf.String()

	// Dark mode adds class to body
	if !strings.Contains(html, `class="dark"`) {
		t.Error("dark mode output should have dark class on body")
	}
}

func TestGenerate_NoFactors(t *testing.T) {
	report := &confidence.Report{
		Title:     "Simple Report",
		Score:     75,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := Generate(&buf, report, Options{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	html := buf.String()

	// Should not have factors section element (class is in CSS, but section element shouldn't render)
	if strings.Contains(html, `class="factors-section"`) {
		t.Error("output should not contain factors section element when no factors")
	}
}
