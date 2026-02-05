package gauge

import (
	"bytes"
	"strings"
	"testing"

	"github.com/boinger/confvis/internal/confidence"
)

// intPtr is a test helper that returns a pointer to an int.
func intPtr(i int) *int { return &i }

func TestGenerateFlat_BasicRendering(t *testing.T) {
	report := &confidence.Report{
		Title:     "Coverage",
		Score:     intPtr(85),
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := GenerateFlat(&buf, report, FlatOptions{})
	if err != nil {
		t.Fatalf("GenerateFlat() error = %v", err)
	}

	svg := buf.String()

	// Should contain SVG element
	if !strings.Contains(svg, "<svg") {
		t.Error("output should contain <svg element")
	}

	// Should contain the score percentage
	if !strings.Contains(svg, "85%") {
		t.Error("output should contain score value 85%")
	}

	// Should contain PASS since 85 >= 75
	if !strings.Contains(svg, "PASS") {
		t.Error("output should contain PASS indicator")
	}

	// Should use title as label by default
	if !strings.Contains(svg, "Coverage") {
		t.Error("output should contain report title as label")
	}

	// Should have three rect elements (label, score, status sections)
	if strings.Count(svg, "<rect") < 3 {
		t.Error("output should contain at least 3 rect elements for sections")
	}
}

func TestGenerateFlat_CustomLabel(t *testing.T) {
	report := &confidence.Report{
		Title:     "Code Coverage",
		Score:     intPtr(75),
		Threshold: 75,
	}

	var buf bytes.Buffer
	opts := FlatOptions{Label: "custom-label"}
	err := GenerateFlat(&buf, report, opts)
	if err != nil {
		t.Fatalf("GenerateFlat() error = %v", err)
	}

	svg := buf.String()

	// Should use custom label instead of title
	if !strings.Contains(svg, "custom-label") {
		t.Error("output should contain custom label")
	}

	// Should NOT contain the original title
	if strings.Contains(svg, "Code Coverage") {
		t.Error("output should not contain original title when custom label is set")
	}
}

func TestGenerateFlat_EmptyLabel(t *testing.T) {
	report := &confidence.Report{
		Title:     "",
		Score:     intPtr(85),
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := GenerateFlat(&buf, report, FlatOptions{})
	if err != nil {
		t.Fatalf("GenerateFlat() error = %v", err)
	}

	// Should not error with empty title/label
	svg := buf.String()
	if !strings.Contains(svg, "<svg") {
		t.Error("output should still produce valid SVG with empty label")
	}
}

func TestGenerateFlat_ScoreColorThresholds(t *testing.T) {
	tests := []struct {
		name       string
		score      int
		wantColor  string // color from GitHubLight scheme
		colorLabel string
	}{
		{"high score (green)", 80, "#1a7f37", "Success"},
		{"threshold score (green)", 75, "#1a7f37", "Success"},
		{"medium score (yellow)", 60, "#9a6700", "Warning"},
		{"low threshold (yellow)", 50, "#9a6700", "Warning"},
		{"low score (red)", 40, "#cf222e", "Danger"},
		{"zero score (red)", 0, "#cf222e", "Danger"},
		{"perfect score (green)", 100, "#1a7f37", "Success"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := tt.score
			report := &confidence.Report{
				Title:     "Test",
				Score:     &score,
				Threshold: 75,
			}

			var buf bytes.Buffer
			err := GenerateFlat(&buf, report, FlatOptions{})
			if err != nil {
				t.Fatalf("GenerateFlat() error = %v", err)
			}

			svg := buf.String()

			// Score and status sections should use the appropriate color
			if !strings.Contains(svg, tt.wantColor) {
				t.Errorf("output should contain %s color %s for score %d", tt.colorLabel, tt.wantColor, tt.score)
			}
		})
	}
}

func TestGenerateFlat_CustomColorThresholds(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtr(85),
		Threshold: 75,
	}

	// With default thresholds (75/50), score 85 would be green
	// But with custom thresholds (90/80), score 85 is warning (yellow)
	var buf bytes.Buffer
	opts := FlatOptions{
		ColorOptions: ColorOptions{GreenAbove: 90, YellowAbove: 80},
	}
	err := GenerateFlat(&buf, report, opts)
	if err != nil {
		t.Fatalf("GenerateFlat() error = %v", err)
	}

	svg := buf.String()
	scheme := GitHubLight()

	// Should use warning color since 85 < 90 but 85 >= 80
	if !strings.Contains(svg, scheme.Warning) {
		t.Errorf("output should contain warning color %s for score 85 with greenAbove=90", scheme.Warning)
	}
}

func TestGenerateFlat_ReportColorThresholds(t *testing.T) {
	// Test that report-level thresholds are used when CLI options are not set
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtr(85),
		Threshold: 75,
		Thresholds: &confidence.ColorThresholds{
			GreenAbove:  90,
			YellowAbove: 80,
		},
	}

	var buf bytes.Buffer
	err := GenerateFlat(&buf, report, FlatOptions{})
	if err != nil {
		t.Fatalf("GenerateFlat() error = %v", err)
	}

	svg := buf.String()
	scheme := GitHubLight()

	// Should use warning color since 85 < 90 (report threshold)
	if !strings.Contains(svg, scheme.Warning) {
		t.Errorf("output should respect report-level color thresholds")
	}
}

func TestGenerateFlat_PassFailStatus(t *testing.T) {
	tests := []struct {
		name       string
		score      int
		threshold  int
		wantStatus string
	}{
		{"passing", 80, 75, "PASS"},
		{"exactly at threshold", 75, 75, "PASS"},
		{"failing", 74, 75, "FAIL"},
		{"zero score", 0, 50, "FAIL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := tt.score
			report := &confidence.Report{
				Title:     "Test",
				Score:     &score,
				Threshold: tt.threshold,
			}

			var buf bytes.Buffer
			err := GenerateFlat(&buf, report, FlatOptions{})
			if err != nil {
				t.Fatalf("GenerateFlat() error = %v", err)
			}

			svg := buf.String()

			if !strings.Contains(svg, tt.wantStatus) {
				t.Errorf("output should contain %s for score %d with threshold %d", tt.wantStatus, tt.score, tt.threshold)
			}
		})
	}
}

func TestGenerateFlat_CustomPassFailLabels(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtr(60),
		Threshold: 75,
		PassLabel: "SUCCESS",
		FailLabel: "NEEDS WORK",
	}

	var buf bytes.Buffer
	err := GenerateFlat(&buf, report, FlatOptions{})
	if err != nil {
		t.Fatalf("GenerateFlat() error = %v", err)
	}

	svg := buf.String()

	// Should use custom fail label
	if !strings.Contains(svg, "NEEDS WORK") {
		t.Error("output should use custom fail label")
	}
	if strings.Contains(svg, "FAIL") {
		t.Error("output should not contain default FAIL when custom label is set")
	}
}

func TestGenerateFlat_DarkMode(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtr(85),
		Threshold: 75,
	}

	var buf bytes.Buffer
	opts := FlatOptions{ColorOptions: ColorOptions{DarkMode: true}}
	err := GenerateFlat(&buf, report, opts)
	if err != nil {
		t.Fatalf("GenerateFlat() error = %v", err)
	}

	svg := buf.String()

	// Dark mode uses #333 for label background
	if !strings.Contains(svg, "#333") {
		t.Error("dark mode output should use dark label background #333")
	}
}

func TestGenerateFlat_LightMode(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtr(85),
		Threshold: 75,
	}

	var buf bytes.Buffer
	opts := FlatOptions{ColorOptions: ColorOptions{DarkMode: false}}
	err := GenerateFlat(&buf, report, opts)
	if err != nil {
		t.Fatalf("GenerateFlat() error = %v", err)
	}

	svg := buf.String()

	// Light mode uses #555 for label background
	if !strings.Contains(svg, "#555") {
		t.Error("light mode output should use light label background #555")
	}
}

func TestGenerateFlat_AllStyles(t *testing.T) {
	styles := []string{"github", "minimal", "corporate", "high-contrast"}

	for _, style := range styles {
		for _, darkMode := range []bool{false, true} {
			name := style
			if darkMode {
				name += "-dark"
			}

			t.Run(name, func(t *testing.T) {
				report := &confidence.Report{
					Title:     "Test",
					Score:     intPtr(85),
					Threshold: 75,
				}

				var buf bytes.Buffer
				opts := FlatOptions{
					ColorOptions: ColorOptions{Style: style, DarkMode: darkMode},
				}
				err := GenerateFlat(&buf, report, opts)
				if err != nil {
					t.Fatalf("GenerateFlat() error = %v", err)
				}

				svg := buf.String()
				scheme := GetColorScheme(style, darkMode)

				// Verify the appropriate success color is used (score 85 >= 75)
				if !strings.Contains(svg, scheme.Success) {
					t.Errorf("output should contain style's success color %s", scheme.Success)
				}
			})
		}
	}
}

func TestGenerateFlat_EdgeCases(t *testing.T) {
	t.Run("score zero", func(t *testing.T) {
		report := &confidence.Report{
			Title:     "Test",
			Score:     intPtr(0),
			Threshold: 75,
		}

		var buf bytes.Buffer
		err := GenerateFlat(&buf, report, FlatOptions{})
		if err != nil {
			t.Fatalf("GenerateFlat() error = %v", err)
		}

		svg := buf.String()
		if !strings.Contains(svg, "0%") {
			t.Error("output should display 0%")
		}
	})

	t.Run("score 100", func(t *testing.T) {
		report := &confidence.Report{
			Title:     "Test",
			Score:     intPtr(100),
			Threshold: 75,
		}

		var buf bytes.Buffer
		err := GenerateFlat(&buf, report, FlatOptions{})
		if err != nil {
			t.Fatalf("GenerateFlat() error = %v", err)
		}

		svg := buf.String()
		if !strings.Contains(svg, "100%") {
			t.Error("output should display 100%")
		}
	})

	t.Run("very long label", func(t *testing.T) {
		report := &confidence.Report{
			Title:     "This is a very long report title that should still work",
			Score:     intPtr(85),
			Threshold: 75,
		}

		var buf bytes.Buffer
		err := GenerateFlat(&buf, report, FlatOptions{})
		if err != nil {
			t.Fatalf("GenerateFlat() error = %v", err)
		}

		svg := buf.String()
		if !strings.Contains(svg, "This is a very long report title that should still work") {
			t.Error("output should handle long labels")
		}
	})
}

func TestGenerateFlat_SVGStructure(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtr(85),
		Threshold: 75,
	}

	var buf bytes.Buffer
	err := GenerateFlat(&buf, report, FlatOptions{})
	if err != nil {
		t.Fatalf("GenerateFlat() error = %v", err)
	}

	svg := buf.String()

	// Should have clip path for rounded corners
	if !strings.Contains(svg, "clipPath") {
		t.Error("output should contain clipPath for rounded corners")
	}

	// Should have group with clip-path
	if !strings.Contains(svg, `clip-path="url(#clip)"`) {
		t.Error("output should use clip-path on group")
	}

	// Should have text shadow elements (fill-opacity for shadow effect)
	if !strings.Contains(svg, "fill-opacity:0.3") {
		t.Error("output should contain text shadow styling")
	}

	// Should have text-anchor:middle for centered text
	if !strings.Contains(svg, "text-anchor:middle") {
		t.Error("output should have centered text")
	}
}
