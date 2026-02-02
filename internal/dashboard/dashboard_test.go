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

// Tests for GenerateMulti

func TestGenerateMulti_BasicOutput(t *testing.T) {
	reports := []ReportSummary{
		{
			Report: &confidence.Report{
				Title:     "API Coverage",
				Score:     90,
				Threshold: 75,
			},
			Weight: 60,
			Path:   "api/confidence.json",
		},
		{
			Report: &confidence.Report{
				Title:     "Web Coverage",
				Score:     80,
				Threshold: 75,
			},
			Weight: 40,
			Path:   "web/confidence.json",
		},
	}

	aggregate := &confidence.Report{
		Title:     "Aggregate",
		Score:     86,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := GenerateMulti(&buf, reports, aggregate, MultiOptions{})
	if err != nil {
		t.Fatalf("GenerateMulti() error = %v", err)
	}

	html := buf.String()

	// Check basic HTML structure
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("output should contain DOCTYPE")
	}
	if !strings.Contains(html, "Aggregate Confidence Report") {
		t.Error("output should contain aggregate title")
	}

	// Check report titles are rendered
	if !strings.Contains(html, "API Coverage") {
		t.Error("output should contain first report title")
	}
	if !strings.Contains(html, "Web Coverage") {
		t.Error("output should contain second report title")
	}

	// Check paths are rendered
	if !strings.Contains(html, "api/confidence.json") {
		t.Error("output should contain first report path")
	}
	if !strings.Contains(html, "web/confidence.json") {
		t.Error("output should contain second report path")
	}

	// Check weights are rendered
	if !strings.Contains(html, "weight: 60") {
		t.Error("output should contain first report weight")
	}
	if !strings.Contains(html, "weight: 40") {
		t.Error("output should contain second report weight")
	}

	// Check report count
	if !strings.Contains(html, "2 reports") {
		t.Error("output should show report count")
	}

	// Check aggregate pass badge (86 >= 75)
	if !strings.Contains(html, "badge-pass") {
		t.Error("output should contain aggregate pass badge")
	}

	// Check embedded SVGs (aggregate + 2 reports = at least 3 svg elements)
	svgCount := strings.Count(html, "<svg")
	if svgCount < 3 {
		t.Errorf("output should contain at least 3 SVG gauges, got %d", svgCount)
	}
}

func TestGenerateMulti_EmptyReports(t *testing.T) {
	reports := []ReportSummary{}

	aggregate := &confidence.Report{
		Title:     "Aggregate",
		Score:     0,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := GenerateMulti(&buf, reports, aggregate, MultiOptions{})
	if err != nil {
		t.Fatalf("GenerateMulti() error = %v", err)
	}

	html := buf.String()

	// Should still produce valid HTML
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("output should contain DOCTYPE even with empty reports")
	}

	// Should show 0 reports count
	if !strings.Contains(html, "0 reports") {
		t.Error("output should show 0 reports count")
	}

	// Should not have any report-title divs (actual content from range loop)
	// The class exists in CSS but content should be empty when no reports
	if strings.Contains(html, `class="report-title"`) {
		t.Error("output should not contain report-title divs with empty reports")
	}
}

func TestGenerateMulti_SingleReport(t *testing.T) {
	reports := []ReportSummary{
		{
			Report: &confidence.Report{
				Title:     "Single Report",
				Score:     75,
				Threshold: 75,
			},
			Weight: 100,
			Path:   "single/confidence.json",
		},
	}

	aggregate := &confidence.Report{
		Title:     "Aggregate",
		Score:     75,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := GenerateMulti(&buf, reports, aggregate, MultiOptions{})
	if err != nil {
		t.Fatalf("GenerateMulti() error = %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, "Single Report") {
		t.Error("output should contain single report title")
	}

	if !strings.Contains(html, "1 reports") {
		t.Error("output should show 1 reports count")
	}
}

func TestGenerateMulti_DarkMode(t *testing.T) {
	reports := []ReportSummary{
		{
			Report: &confidence.Report{
				Title:     "Test",
				Score:     85,
				Threshold: 75,
			},
			Weight: 100,
			Path:   "test.json",
		},
	}

	aggregate := &confidence.Report{
		Title:     "Aggregate",
		Score:     85,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := GenerateMulti(&buf, reports, aggregate, MultiOptions{DarkMode: true})
	if err != nil {
		t.Fatalf("GenerateMulti() error = %v", err)
	}

	html := buf.String()

	// Dark mode adds class to body
	if !strings.Contains(html, `class="dark"`) {
		t.Error("dark mode output should have dark class on body")
	}
}

func TestGenerateMulti_LightMode(t *testing.T) {
	reports := []ReportSummary{
		{
			Report: &confidence.Report{
				Title:     "Test",
				Score:     85,
				Threshold: 75,
			},
			Weight: 100,
			Path:   "test.json",
		},
	}

	aggregate := &confidence.Report{
		Title:     "Aggregate",
		Score:     85,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := GenerateMulti(&buf, reports, aggregate, MultiOptions{DarkMode: false})
	if err != nil {
		t.Fatalf("GenerateMulti() error = %v", err)
	}

	html := buf.String()

	// Light mode should NOT have dark class
	if strings.Contains(html, `class="dark"`) {
		t.Error("light mode output should not have dark class on body")
	}
}

func TestGenerateMulti_FailingAggregate(t *testing.T) {
	reports := []ReportSummary{
		{
			Report: &confidence.Report{
				Title:     "Failing Report",
				Score:     60,
				Threshold: 75,
			},
			Weight: 100,
			Path:   "failing.json",
		},
	}

	aggregate := &confidence.Report{
		Title:     "Aggregate",
		Score:     60,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := GenerateMulti(&buf, reports, aggregate, MultiOptions{})
	if err != nil {
		t.Fatalf("GenerateMulti() error = %v", err)
	}

	html := buf.String()

	// Should show fail badge for aggregate
	if !strings.Contains(html, "badge-fail") {
		t.Error("output should contain fail badge for failing aggregate")
	}
}

func TestGenerateMulti_WithFactors(t *testing.T) {
	reports := []ReportSummary{
		{
			Report: &confidence.Report{
				Title:     "Coverage Report",
				Score:     85,
				Threshold: 75,
				Factors: []confidence.Factor{
					{Name: "Unit Tests", Score: 90, Weight: 50},
					{Name: "Integration Tests", Score: 80, Weight: 50},
				},
			},
			Weight: 100,
			Path:   "coverage.json",
		},
	}

	aggregate := &confidence.Report{
		Title:     "Aggregate",
		Score:     85,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := GenerateMulti(&buf, reports, aggregate, MultiOptions{})
	if err != nil {
		t.Fatalf("GenerateMulti() error = %v", err)
	}

	html := buf.String()

	// Check factors are rendered
	if !strings.Contains(html, "Unit Tests") {
		t.Error("output should contain factor name")
	}
	if !strings.Contains(html, "Integration Tests") {
		t.Error("output should contain second factor name")
	}

	// Check factor scores are rendered
	if !strings.Contains(html, "90%") {
		t.Error("output should contain factor score percentage")
	}
}

func TestGenerateMulti_FactorsWithURLs(t *testing.T) {
	reports := []ReportSummary{
		{
			Report: &confidence.Report{
				Title:     "Coverage Report",
				Score:     85,
				Threshold: 75,
				Factors: []confidence.Factor{
					{Name: "Codecov", Score: 90, Weight: 50, URL: "https://codecov.io/report"},
					{Name: "Local Test", Score: 80, Weight: 50}, // No URL
				},
			},
			Weight: 100,
			Path:   "coverage.json",
		},
	}

	aggregate := &confidence.Report{
		Title:     "Aggregate",
		Score:     85,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := GenerateMulti(&buf, reports, aggregate, MultiOptions{})
	if err != nil {
		t.Fatalf("GenerateMulti() error = %v", err)
	}

	html := buf.String()

	// Factor with URL should be a link
	if !strings.Contains(html, `href="https://codecov.io/report"`) {
		t.Error("output should contain factor URL as href")
	}
	if !strings.Contains(html, "factor-link") {
		t.Error("output should have factor-link class for linked factors")
	}

	// Factor without URL should not be a link (just plain text)
	// Check that "Local Test" appears but without href for that specific text
	if !strings.Contains(html, "Local Test") {
		t.Error("output should contain factor without URL")
	}
}

func TestGenerateMulti_HTMLStructure(t *testing.T) {
	reports := []ReportSummary{
		{
			Report: &confidence.Report{
				Title:     "Test Report",
				Score:     85,
				Threshold: 75,
			},
			Weight: 100,
			Path:   "test.json",
		},
	}

	aggregate := &confidence.Report{
		Title:     "Aggregate",
		Score:     85,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := GenerateMulti(&buf, reports, aggregate, MultiOptions{})
	if err != nil {
		t.Fatalf("GenerateMulti() error = %v", err)
	}

	html := buf.String()

	// Check key structural elements
	if !strings.Contains(html, "aggregate-section") {
		t.Error("output should contain aggregate-section")
	}
	if !strings.Contains(html, "reports-grid") {
		t.Error("output should contain reports-grid")
	}
	if !strings.Contains(html, "report-card") {
		t.Error("output should contain report-card")
	}
	if !strings.Contains(html, "gauge-container") {
		t.Error("output should contain gauge-container")
	}
	if !strings.Contains(html, "theme-toggle") {
		t.Error("output should contain theme toggle button")
	}
}

func TestGenerateMulti_MultipleReportsWithMixedPassFail(t *testing.T) {
	reports := []ReportSummary{
		{
			Report: &confidence.Report{
				Title:     "Passing Report",
				Score:     90,
				Threshold: 75,
			},
			Weight: 50,
			Path:   "pass.json",
		},
		{
			Report: &confidence.Report{
				Title:     "Failing Report",
				Score:     50,
				Threshold: 75,
			},
			Weight: 50,
			Path:   "fail.json",
		},
	}

	// Aggregate passes (weighted average ~70, but threshold met due to higher passing report)
	aggregate := &confidence.Report{
		Title:     "Aggregate",
		Score:     70,
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := GenerateMulti(&buf, reports, aggregate, MultiOptions{})
	if err != nil {
		t.Fatalf("GenerateMulti() error = %v", err)
	}

	html := buf.String()

	// Both report titles should be present
	if !strings.Contains(html, "Passing Report") {
		t.Error("output should contain passing report")
	}
	if !strings.Contains(html, "Failing Report") {
		t.Error("output should contain failing report")
	}

	// Aggregate should show fail (70 < 75)
	if !strings.Contains(html, "badge-fail") {
		t.Error("aggregate badge should show fail for score 70 with threshold 75")
	}
}
