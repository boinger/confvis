package gauge

import (
	"bytes"
	"fmt"
	"math"
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

func TestGenerate_ZeroDimensionDefaults(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtrG(85),
		Threshold: 75,
	}

	tests := []struct {
		name       string
		width      int
		height     int
		wantWidth  string
		wantHeight string
		wantView   string
	}{
		{
			name:       "both zero",
			width:      0,
			height:     0,
			wantWidth:  `width="200"`,
			wantHeight: `height="120"`,
			wantView:   `viewBox="0 0 200 120"`,
		},
		{
			name:       "width-only zero",
			width:      0,
			height:     150,
			wantWidth:  `width="200"`,
			wantHeight: `height="150"`,
			wantView:   `viewBox="0 0 200 150"`,
		},
		{
			name:       "height-only zero",
			width:      250,
			height:     0,
			wantWidth:  `width="250"`,
			wantHeight: `height="120"`,
			wantView:   `viewBox="0 0 250 120"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			opts := Options{Width: tt.width, Height: tt.height}
			err := Generate(&buf, report, opts)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			svg := buf.String()

			if !strings.Contains(svg, tt.wantWidth) {
				t.Errorf("SVG should contain %s", tt.wantWidth)
			}
			if !strings.Contains(svg, tt.wantHeight) {
				t.Errorf("SVG should contain %s", tt.wantHeight)
			}
			if !strings.Contains(svg, tt.wantView) {
				t.Errorf("SVG should contain %s", tt.wantView)
			}
		})
	}
}

func TestGenerate_ThresholdMarkerPresence(t *testing.T) {
	tests := []struct {
		name       string
		threshold  int
		wantMarker bool
	}{
		{"threshold 75", 75, true},
		{"threshold 50", 50, true},
		{"threshold 1", 1, true},
		{"threshold 99", 99, true},
		{"threshold 0 (no marker)", 0, false},
		{"threshold 100 (no marker)", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &confidence.Report{
				Title:     "Test",
				Score:     intPtrG(85),
				Threshold: tt.threshold,
			}

			var buf bytes.Buffer
			err := Generate(&buf, report, defaultOptions())
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			svg := buf.String()
			hasMarker := strings.Contains(svg, "<circle")

			if hasMarker != tt.wantMarker {
				t.Errorf("threshold %d: hasMarker=%v, want %v", tt.threshold, hasMarker, tt.wantMarker)
			}

			// When present, marker should use the scheme's TextMuted color
			if tt.wantMarker {
				scheme := gitHubLight()
				if !strings.Contains(svg, scheme.TextMuted) {
					t.Errorf("threshold marker should use TextMuted color %s", scheme.TextMuted)
				}
			}
		})
	}
}

func TestGenerate_CustomPassFailLabels(t *testing.T) {
	t.Run("custom pass label", func(t *testing.T) {
		report := &confidence.Report{
			Title:     "Test",
			Score:     intPtrG(85),
			Threshold: 75,
			PassLabel: "APPROVED",
		}

		var buf bytes.Buffer
		err := Generate(&buf, report, defaultOptions())
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		svg := buf.String()

		if !strings.Contains(svg, "APPROVED") {
			t.Error("output should contain custom pass label APPROVED")
		}
		if strings.Contains(svg, "PASS") {
			t.Error("output should not contain default PASS when custom label set")
		}
	})

	t.Run("custom fail label", func(t *testing.T) {
		report := &confidence.Report{
			Title:     "Test",
			Score:     intPtrG(60),
			Threshold: 75,
			FailLabel: "NEEDS WORK",
		}

		var buf bytes.Buffer
		err := Generate(&buf, report, defaultOptions())
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}

		svg := buf.String()

		if !strings.Contains(svg, "NEEDS WORK") {
			t.Error("output should contain custom fail label NEEDS WORK")
		}
		if strings.Contains(svg, "FAIL") {
			t.Error("output should not contain default FAIL when custom label set")
		}
	})
}

func TestGenerate_AllStyles(t *testing.T) {
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
					Score:     intPtrG(85),
					Threshold: 75,
				}

				opts := Options{
					ColorOptions: ColorOptions{Style: style, DarkMode: darkMode},
					Width:        200,
					Height:       120,
				}

				var buf bytes.Buffer
				err := Generate(&buf, report, opts)
				if err != nil {
					t.Fatalf("Generate() error = %v", err)
				}

				svg := buf.String()
				scheme := GetColorScheme(style, darkMode)

				// Background rect should use the scheme's background color
				if !strings.Contains(svg, scheme.Background) {
					t.Errorf("output should contain background color %s", scheme.Background)
				}

				// Arc fill should use score color (85 >= 75 => Success)
				scoreColor := scheme.ScoreColor(85, 75, 50)
				if !strings.Contains(svg, scoreColor) {
					t.Errorf("output should contain score color %s", scoreColor)
				}
			})
		}
	}
}

func TestGenerate_ArcGeometry(t *testing.T) {
	tests := []struct {
		name   string
		score  int
		width  int
		height int
	}{
		{"score 0 default dims", 0, 200, 120},
		{"score 50 default dims", 50, 200, 120},
		{"score 100 default dims", 100, 200, 120},
		{"score 75 custom dims", 75, 300, 180},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := &confidence.Report{
				Title:     "Test",
				Score:     intPtrG(tt.score),
				Threshold: 75,
			}

			opts := Options{Width: tt.width, Height: tt.height}

			var buf bytes.Buffer
			err := Generate(&buf, report, opts)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			svg := buf.String()

			// Compute expected arc values using the same formulas as gauge.go
			radius := min(tt.width/2-20, tt.height-40)
			arcLength := math.Pi * float64(radius)
			filledLength := arcLength * float64(tt.score) / 100.0

			expected := fmt.Sprintf("stroke-dasharray:%.2f %.2f", filledLength, arcLength)
			if !strings.Contains(svg, expected) {
				t.Errorf("SVG should contain %q", expected)
			}
		})
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
