package gauge

import (
	"bytes"
	"fmt"
	"io"
	"math"

	svg "github.com/ajstarks/svgo"

	"github.com/boinger/confvis/internal/confidence"
)

// Options configures gauge generation.
type Options struct {
	Width       int
	Height      int
	DarkMode    bool
	Style       string // Color scheme style: github, minimal, corporate, high-contrast
	GreenAbove  int    // Score threshold for green color (0 = use report default or 75)
	YellowAbove int    // Score threshold for yellow color (0 = use report default or 50)
}

// DefaultOptions returns sensible defaults for gauge rendering.
func DefaultOptions() Options {
	return Options{
		Width:    200,
		Height:   120,
		DarkMode: false,
	}
}

// Generate creates an SVG gauge for the given report and writes it to w.
func Generate(w io.Writer, report *confidence.Report, opts Options) error {
	if opts.Width == 0 {
		opts.Width = 200
	}
	if opts.Height == 0 {
		opts.Height = 120
	}

	scheme := GetColorScheme(opts.Style, opts.DarkMode)

	// Determine color thresholds: CLI overrides > report config > defaults
	thresholds := report.EffectiveColorThresholds()
	if opts.GreenAbove > 0 {
		thresholds.GreenAbove = opts.GreenAbove
	}
	if opts.YellowAbove > 0 {
		thresholds.YellowAbove = opts.YellowAbove
	}

	canvas := svg.New(w)
	canvas.Start(opts.Width, opts.Height)

	// Background
	canvas.Rect(0, 0, opts.Width, opts.Height, fmt.Sprintf("fill:%s", scheme.Background))

	centerX := opts.Width / 2
	centerY := opts.Height - 20
	radius := min(opts.Width/2-20, opts.Height-40)

	// Calculate arc geometry
	arcLength := math.Pi * float64(radius)
	filledLength := arcLength * float64(report.Score) / 100.0

	// Track path (semi-circle from left to right)
	trackPath := fmt.Sprintf("M %d %d A %d %d 0 0 1 %d %d",
		centerX-radius, centerY,
		radius, radius,
		centerX+radius, centerY)

	// Draw track (background arc)
	canvas.Path(trackPath, fmt.Sprintf(
		"fill:none;stroke:%s;stroke-width:12;stroke-linecap:round",
		scheme.TrackColor))

	// Draw filled arc
	scoreColor := scheme.ScoreColor(report.Score, thresholds.GreenAbove, thresholds.YellowAbove)
	canvas.Path(trackPath, fmt.Sprintf(
		"fill:none;stroke:%s;stroke-width:12;stroke-linecap:round;stroke-dasharray:%.2f %.2f",
		scoreColor, filledLength, arcLength))

	// Draw threshold marker
	if report.Threshold > 0 && report.Threshold < 100 {
		thresholdAngle := math.Pi * (1 - float64(report.Threshold)/100)
		markerX := centerX + int(float64(radius)*math.Cos(thresholdAngle))
		markerY := centerY - int(float64(radius)*math.Sin(thresholdAngle))
		canvas.Circle(markerX, markerY, 4, fmt.Sprintf("fill:%s", scheme.TextMuted))
	}

	// Score text
	canvas.Text(centerX, centerY-radius/3, fmt.Sprintf("%d", report.Score),
		fmt.Sprintf("text-anchor:middle;font-family:system-ui,-apple-system,sans-serif;font-size:%dpx;font-weight:bold;fill:%s",
			radius/2, scheme.TextPrimary))

	// Pass/fail indicator with custom labels
	statusText := report.EffectivePassLabel()
	statusColor := scheme.Success
	if !report.Passed() {
		statusText = report.EffectiveFailLabel()
		statusColor = scheme.Danger
	}
	canvas.Text(centerX, centerY-5, statusText,
		fmt.Sprintf("text-anchor:middle;font-family:system-ui,-apple-system,sans-serif;font-size:12px;font-weight:600;fill:%s",
			statusColor))

	canvas.End()
	return nil
}

// GenerateToString creates an SVG gauge and returns it as a string.
func GenerateToString(report *confidence.Report, opts Options) (string, error) {
	var buf bytes.Buffer
	if err := Generate(&buf, report, opts); err != nil {
		return "", err
	}
	return buf.String(), nil
}
