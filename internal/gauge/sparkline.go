package gauge

import (
	"fmt"
	"io"

	svg "github.com/ajstarks/svgo"

	"github.com/boinger/confvis/internal/confidence"
)

// SparklineOptions configures sparkline badge generation.
type SparklineOptions struct {
	Width       int   // Total badge width
	Height      int   // Total badge height
	Scores      []int // Historical scores to display
	DarkMode    bool  // Use dark mode colors
	Style       string // Color scheme style
	GreenAbove  int    // Score threshold for green color
	YellowAbove int    // Score threshold for yellow color
}

// GenerateSparkline creates an SVG badge with a sparkline showing score history.
func GenerateSparkline(w io.Writer, report *confidence.Report, opts SparklineOptions) error {
	if opts.Width == 0 {
		opts.Width = 120
	}
	if opts.Height == 0 {
		opts.Height = 28
	}

	scheme := GetColorScheme(opts.Style, opts.DarkMode)
	thresholds := resolveColorThresholds(report, opts.GreenAbove, opts.YellowAbove)

	scoreColor := scheme.ScoreColor(report.ScoreValue(), thresholds.GreenAbove, thresholds.YellowAbove)

	// Calculate dimensions
	labelWidth := 40 // Space for score text
	sparkWidth := opts.Width - labelWidth - 8
	sparkHeight := opts.Height - 8
	sparkX := 4
	sparkY := 4

	canvas := svg.New(w)
	canvas.Start(opts.Width, opts.Height)

	// Background with rounded corners
	canvas.Roundrect(0, 0, opts.Width, opts.Height, 3, 3, fmt.Sprintf("fill:%s", scheme.Background))

	// Border
	canvas.Roundrect(0, 0, opts.Width, opts.Height, 3, 3, fmt.Sprintf("fill:none;stroke:%s;stroke-width:1", scheme.Border))

	// Draw sparkline if we have scores
	if len(opts.Scores) > 1 {
		xs, ys := calculateSparklineCoords(opts.Scores, sparkX, sparkY, sparkWidth, sparkHeight)

		// Draw sparkline area (filled polygon)
		areaXs := make([]int, len(xs)+2)
		areaYs := make([]int, len(ys)+2)
		copy(areaXs, xs)
		copy(areaYs, ys)
		// Close the polygon at the bottom
		areaXs[len(xs)] = sparkX + sparkWidth
		areaYs[len(ys)] = sparkY + sparkHeight
		areaXs[len(xs)+1] = sparkX
		areaYs[len(ys)+1] = sparkY + sparkHeight

		canvas.Polygon(areaXs, areaYs, fmt.Sprintf("fill:%s;fill-opacity:0.2", scoreColor))

		// Draw sparkline line
		canvas.Polyline(xs, ys, fmt.Sprintf("fill:none;stroke:%s;stroke-width:1.5;stroke-linecap:round;stroke-linejoin:round", scoreColor))

		// Draw current point (last point)
		if len(xs) > 0 {
			canvas.Circle(xs[len(xs)-1], ys[len(ys)-1], 2, fmt.Sprintf("fill:%s", scoreColor))
		}
	} else if len(opts.Scores) == 1 {
		// Single point - draw a horizontal line
		y := sparkY + sparkHeight - (opts.Scores[0] * sparkHeight / 100)
		canvas.Line(sparkX, y, sparkX+sparkWidth, y, fmt.Sprintf("stroke:%s;stroke-width:1.5", scoreColor))
	}

	// Score text on the right
	textX := opts.Width - labelWidth/2
	textY := opts.Height/2 + 5
	canvas.Text(textX, textY, fmt.Sprintf("%d%%", report.ScoreValue()),
		fmt.Sprintf("text-anchor:middle;font-family:system-ui,-apple-system,sans-serif;font-size:12px;font-weight:bold;fill:%s", scoreColor))

	canvas.End()
	return nil
}

// calculateSparklineCoords generates x and y coordinate slices for the sparkline.
func calculateSparklineCoords(scores []int, x, y, width, height int) ([]int, []int) {
	if len(scores) == 0 {
		return nil, nil
	}

	xs := make([]int, len(scores))
	ys := make([]int, len(scores))

	var step float64
	if len(scores) > 1 {
		step = float64(width) / float64(len(scores)-1)
	}

	for i, score := range scores {
		xs[i] = x + int(float64(i)*step)
		// Invert Y since SVG Y increases downward
		ys[i] = y + height - (score * height / 100)
	}

	return xs, ys
}
