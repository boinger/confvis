package gauge

import (
	"bytes"
	"strings"
	"testing"

	"github.com/boinger/confvis/internal/confidence"
)

// intPtrS is a test helper for sparkline_test.go.
func intPtrS(i int) *int { return &i }

func TestGenerateSparkline_BasicRendering(t *testing.T) {
	report := &confidence.Report{
		Title:     "Coverage",
		Score:     intPtrS(85),
		Threshold: 75,
	}

	var buf bytes.Buffer
	opts := SparklineOptions{
		Scores: []int{70, 75, 80, 85},
	}
	err := GenerateSparkline(&buf, report, opts)
	if err != nil {
		t.Fatalf("GenerateSparkline() error = %v", err)
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

	// Should contain polyline for sparkline
	if !strings.Contains(svg, "<polyline") {
		t.Error("output should contain polyline for sparkline")
	}

	// Should contain polygon for area fill
	if !strings.Contains(svg, "<polygon") {
		t.Error("output should contain polygon for area fill")
	}

	// Should contain circle for current point
	if !strings.Contains(svg, "<circle") {
		t.Error("output should contain circle for current data point")
	}
}

func TestGenerateSparkline_SingleScore(t *testing.T) {
	report := &confidence.Report{
		Title:     "Coverage",
		Score:     intPtrS(75),
		Threshold: 75,
	}

	var buf bytes.Buffer
	opts := SparklineOptions{
		Scores: []int{75},
	}
	err := GenerateSparkline(&buf, report, opts)
	if err != nil {
		t.Fatalf("GenerateSparkline() error = %v", err)
	}

	svg := buf.String()

	// Single score should draw a horizontal line
	if !strings.Contains(svg, "<line") {
		t.Error("single score output should contain horizontal line element")
	}

	// Should NOT have polyline (that's for multiple points)
	if strings.Contains(svg, "<polyline") {
		t.Error("single score output should not contain polyline")
	}
}

func TestGenerateSparkline_EmptyScores(t *testing.T) {
	report := &confidence.Report{
		Title:     "Coverage",
		Score:     intPtrS(85),
		Threshold: 75,
	}

	var buf bytes.Buffer
	opts := SparklineOptions{
		Scores: []int{},
	}
	err := GenerateSparkline(&buf, report, opts)
	if err != nil {
		t.Fatalf("GenerateSparkline() error = %v", err)
	}

	svg := buf.String()

	// Should still produce valid SVG
	if !strings.Contains(svg, "<svg") {
		t.Error("output should contain SVG even with no scores")
	}

	// Should not have sparkline elements
	if strings.Contains(svg, "<polyline") {
		t.Error("output should not contain polyline with empty scores")
	}
	if strings.Contains(svg, "<line") {
		t.Error("output should not contain line with empty scores")
	}

	// Should still show the score
	if !strings.Contains(svg, "85%") {
		t.Error("output should still contain score even with no history")
	}
}

func TestGenerateSparkline_NilScores(t *testing.T) {
	report := &confidence.Report{
		Title:     "Coverage",
		Score:     intPtrS(85),
		Threshold: 75,
	}

	var buf bytes.Buffer
	opts := SparklineOptions{
		Scores: nil,
	}
	err := GenerateSparkline(&buf, report, opts)
	if err != nil {
		t.Fatalf("GenerateSparkline() error = %v", err)
	}

	svg := buf.String()

	// Should still produce valid SVG
	if !strings.Contains(svg, "<svg") {
		t.Error("output should produce valid SVG with nil scores")
	}
}

func TestGenerateSparkline_DefaultDimensions(t *testing.T) {
	report := &confidence.Report{
		Title:     "Coverage",
		Score:     intPtrS(85),
		Threshold: 75,
	}

	var buf bytes.Buffer
	opts := SparklineOptions{
		Scores: []int{70, 75, 80, 85},
	}
	err := GenerateSparkline(&buf, report, opts)
	if err != nil {
		t.Fatalf("GenerateSparkline() error = %v", err)
	}

	svg := buf.String()

	// Default width is 120
	if !strings.Contains(svg, `width="120"`) {
		t.Error("output should have default width 120")
	}

	// Default height is 28
	if !strings.Contains(svg, `height="28"`) {
		t.Error("output should have default height 28")
	}
}

func TestGenerateSparkline_CustomDimensions(t *testing.T) {
	report := &confidence.Report{
		Title:     "Coverage",
		Score:     intPtrS(85),
		Threshold: 75,
	}

	var buf bytes.Buffer
	opts := SparklineOptions{
		Width:  200,
		Height: 50,
		Scores: []int{70, 75, 80, 85},
	}
	err := GenerateSparkline(&buf, report, opts)
	if err != nil {
		t.Fatalf("GenerateSparkline() error = %v", err)
	}

	svg := buf.String()

	if !strings.Contains(svg, `width="200"`) {
		t.Error("output should have custom width 200")
	}

	if !strings.Contains(svg, `height="50"`) {
		t.Error("output should have custom height 50")
	}
}

func TestGenerateSparkline_ColorThresholds(t *testing.T) {
	tests := []struct {
		name      string
		score     int
		wantColor string
	}{
		{"high score (green)", 85, "#1a7f37"},
		{"medium score (yellow)", 60, "#9a6700"},
		{"low score (red)", 40, "#cf222e"},
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
			opts := SparklineOptions{
				Scores: []int{50, 60, 70, tt.score},
			}
			err := GenerateSparkline(&buf, report, opts)
			if err != nil {
				t.Fatalf("GenerateSparkline() error = %v", err)
			}

			svg := buf.String()

			if !strings.Contains(svg, tt.wantColor) {
				t.Errorf("output should contain color %s for score %d", tt.wantColor, tt.score)
			}
		})
	}
}

func TestGenerateSparkline_CustomColorThresholds(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtrS(85),
		Threshold: 75,
	}

	var buf bytes.Buffer
	opts := SparklineOptions{
		ColorOptions: ColorOptions{GreenAbove: 90, YellowAbove: 80},
		Scores:       []int{80, 82, 84, 85},
	}
	err := GenerateSparkline(&buf, report, opts)
	if err != nil {
		t.Fatalf("GenerateSparkline() error = %v", err)
	}

	svg := buf.String()
	scheme := GitHubLight()

	// Score 85 with greenAbove=90 should be warning (yellow)
	if !strings.Contains(svg, scheme.Warning) {
		t.Errorf("output should use warning color for score 85 with greenAbove=90")
	}
}

func TestGenerateSparkline_DarkMode(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtrS(85),
		Threshold: 75,
	}

	var buf bytes.Buffer
	opts := SparklineOptions{
		ColorOptions: ColorOptions{DarkMode: true},
		Scores:       []int{70, 75, 80, 85},
	}
	err := GenerateSparkline(&buf, report, opts)
	if err != nil {
		t.Fatalf("GenerateSparkline() error = %v", err)
	}

	svg := buf.String()
	scheme := GitHubDark()

	// Should use dark mode background
	if !strings.Contains(svg, scheme.Background) {
		t.Errorf("dark mode output should contain background color %s", scheme.Background)
	}

	// Should use dark mode border
	if !strings.Contains(svg, scheme.Border) {
		t.Errorf("dark mode output should contain border color %s", scheme.Border)
	}
}

func TestGenerateSparkline_AllStyles(t *testing.T) {
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
					Score:     intPtrS(85),
					Threshold: 75,
				}

				var buf bytes.Buffer
				opts := SparklineOptions{
					ColorOptions: ColorOptions{Style: style, DarkMode: darkMode},
					Scores:       []int{70, 75, 80, 85},
				}
				err := GenerateSparkline(&buf, report, opts)
				if err != nil {
					t.Fatalf("GenerateSparkline() error = %v", err)
				}

				svg := buf.String()
				scheme := GetColorScheme(style, darkMode)

				// Verify background color
				if !strings.Contains(svg, scheme.Background) {
					t.Errorf("output should contain style's background color %s", scheme.Background)
				}
			})
		}
	}
}

func TestGenerateSparkline_EdgeCases(t *testing.T) {
	t.Run("all same values", func(t *testing.T) {
		report := &confidence.Report{
			Title:     "Test",
			Score:     intPtrS(75),
			Threshold: 75,
		}

		var buf bytes.Buffer
		opts := SparklineOptions{
			Scores: []int{75, 75, 75, 75},
		}
		err := GenerateSparkline(&buf, report, opts)
		if err != nil {
			t.Fatalf("GenerateSparkline() error = %v", err)
		}

		svg := buf.String()
		if !strings.Contains(svg, "<svg") {
			t.Error("output should produce valid SVG with all same values")
		}
	})

	t.Run("extreme values", func(t *testing.T) {
		report := &confidence.Report{
			Title:     "Test",
			Score:     intPtrS(100),
			Threshold: 75,
		}

		var buf bytes.Buffer
		opts := SparklineOptions{
			Scores: []int{0, 50, 100, 0, 100},
		}
		err := GenerateSparkline(&buf, report, opts)
		if err != nil {
			t.Fatalf("GenerateSparkline() error = %v", err)
		}

		svg := buf.String()
		if !strings.Contains(svg, "<svg") {
			t.Error("output should handle extreme values (0 and 100)")
		}
	})

	t.Run("two points", func(t *testing.T) {
		report := &confidence.Report{
			Title:     "Test",
			Score:     intPtrS(85),
			Threshold: 75,
		}

		var buf bytes.Buffer
		opts := SparklineOptions{
			Scores: []int{70, 85},
		}
		err := GenerateSparkline(&buf, report, opts)
		if err != nil {
			t.Fatalf("GenerateSparkline() error = %v", err)
		}

		svg := buf.String()
		if !strings.Contains(svg, "<polyline") {
			t.Error("two points should still create a polyline")
		}
	})
}

func TestGenerateSparkline_SVGStructure(t *testing.T) {
	report := &confidence.Report{
		Title:     "Test",
		Score:     intPtrS(85),
		Threshold: 75,
	}

	var buf bytes.Buffer
	opts := SparklineOptions{
		Scores: []int{70, 75, 80, 85},
	}
	err := GenerateSparkline(&buf, report, opts)
	if err != nil {
		t.Fatalf("GenerateSparkline() error = %v", err)
	}

	svg := buf.String()

	// Should have rounded rectangle for background
	if !strings.Contains(svg, "<rect") {
		t.Error("output should contain rect for background")
	}

	// Should have fill-opacity for area transparency
	if !strings.Contains(svg, "fill-opacity:0.2") {
		t.Error("output should contain fill-opacity for area transparency")
	}

	// Should have stroke-linecap:round for smooth lines
	if !strings.Contains(svg, "stroke-linecap:round") {
		t.Error("output should have rounded line caps")
	}
}

// Tests for calculateSparklineCoords helper function

func TestCalculateSparklineCoords_Basic(t *testing.T) {
	scores := []int{0, 50, 100}
	x, y, width, height := 0, 0, 100, 100

	xs, ys := calculateSparklineCoords(scores, x, y, width, height)

	if len(xs) != 3 || len(ys) != 3 {
		t.Fatalf("expected 3 coordinates, got xs=%d, ys=%d", len(xs), len(ys))
	}

	// First point should be at x=0
	if xs[0] != 0 {
		t.Errorf("first x coordinate should be 0, got %d", xs[0])
	}

	// Last point should be at x=width (100)
	if xs[2] != 100 {
		t.Errorf("last x coordinate should be 100, got %d", xs[2])
	}

	// Score 0 should be at bottom (y = height = 100)
	if ys[0] != 100 {
		t.Errorf("score 0 should map to y=100, got %d", ys[0])
	}

	// Score 50 should be at middle (y = 50)
	if ys[1] != 50 {
		t.Errorf("score 50 should map to y=50, got %d", ys[1])
	}

	// Score 100 should be at top (y = 0)
	if ys[2] != 0 {
		t.Errorf("score 100 should map to y=0, got %d", ys[2])
	}
}

func TestCalculateSparklineCoords_EmptyScores(t *testing.T) {
	xs, ys := calculateSparklineCoords([]int{}, 0, 0, 100, 100)

	if xs != nil || ys != nil {
		t.Error("empty scores should return nil coordinates")
	}
}

func TestCalculateSparklineCoords_SingleScore(t *testing.T) {
	scores := []int{75}
	x, y, width, height := 0, 0, 100, 100

	xs, ys := calculateSparklineCoords(scores, x, y, width, height)

	if len(xs) != 1 || len(ys) != 1 {
		t.Fatalf("expected 1 coordinate, got xs=%d, ys=%d", len(xs), len(ys))
	}

	// Single point should be at x=0
	if xs[0] != 0 {
		t.Errorf("single point x should be 0, got %d", xs[0])
	}

	// Score 75 should map to y=25 (100 - 75)
	if ys[0] != 25 {
		t.Errorf("score 75 should map to y=25, got %d", ys[0])
	}
}

func TestCalculateSparklineCoords_WithOffset(t *testing.T) {
	scores := []int{50, 100}
	x, y, width, height := 10, 20, 100, 100

	xs, ys := calculateSparklineCoords(scores, x, y, width, height)

	// First point should be at x + 0 = 10
	if xs[0] != 10 {
		t.Errorf("first x with offset should be 10, got %d", xs[0])
	}

	// Second point should be at x + width = 110
	if xs[1] != 110 {
		t.Errorf("second x with offset should be 110, got %d", xs[1])
	}

	// Score 50 should map to y + height - 50 = 20 + 100 - 50 = 70
	if ys[0] != 70 {
		t.Errorf("score 50 with y offset should be 70, got %d", ys[0])
	}

	// Score 100 should map to y + height - 100 = 20 + 0 = 20
	if ys[1] != 20 {
		t.Errorf("score 100 with y offset should be 20, got %d", ys[1])
	}
}

func TestCalculateSparklineCoords_ManyPoints(t *testing.T) {
	scores := []int{0, 25, 50, 75, 100}
	x, y, width, height := 0, 0, 100, 100

	xs, ys := calculateSparklineCoords(scores, x, y, width, height)

	// Check x spacing (100 / 4 = 25 per step)
	expectedXs := []int{0, 25, 50, 75, 100}
	for i, expected := range expectedXs {
		if xs[i] != expected {
			t.Errorf("xs[%d] should be %d, got %d", i, expected, xs[i])
		}
	}

	// Check y mapping (inverted: 100 - score)
	expectedYs := []int{100, 75, 50, 25, 0}
	for i, expected := range expectedYs {
		if ys[i] != expected {
			t.Errorf("ys[%d] should be %d, got %d", i, expected, ys[i])
		}
	}
}
