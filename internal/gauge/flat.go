package gauge

import (
	"fmt"
	"io"

	svg "github.com/ajstarks/svgo"

	"github.com/boinger/confvis/internal/confidence"
)

const (
	// SVG style fragments for flat badges.
	styleFillFmt      = "fill:%s"
	styleShadowAnchor = ";fill:#010101;fill-opacity:0.3;text-anchor:middle"
	styleTextAnchor   = ";text-anchor:middle"
)

// FlatOptions configures flat badge generation.
type FlatOptions struct {
	ColorOptions
	Label string // Label text (left side), defaults to report title
	Icon  string // SVG path data for icon (rendered in label section)
}

// GenerateFlat creates a shields.io-style flat badge for the given report.
func GenerateFlat(w io.Writer, report *confidence.Report, opts FlatOptions) error {
	scheme := GetColorScheme(opts.Style, opts.DarkMode)
	thresholds := resolveColorThresholds(report, opts.GreenAbove, opts.YellowAbove)

	// Use custom label or default to report title
	label := opts.Label
	if label == "" {
		label = report.Title
	}

	// Score and status text
	scoreText := fmt.Sprintf("%d%%", report.ScoreValue())
	statusText := report.EffectivePassLabel()
	if !report.Passed() {
		statusText = report.EffectiveFailLabel()
	}

	// Calculate widths based on text length
	// Approximate: 7px per character + padding
	iconWidth := 0
	if opts.Icon != "" {
		iconWidth = 16 // 14px icon + 2px padding
	}
	labelWidth := len(label)*7 + 12 + iconWidth
	scoreWidth := len(scoreText)*7 + 12
	statusWidth := len(statusText)*7 + 12
	totalWidth := labelWidth + scoreWidth + statusWidth

	height := 20
	radius := 3

	scoreColor := scheme.ScoreColor(report.ScoreValue(), thresholds.GreenAbove, thresholds.YellowAbove)

	canvas := svg.New(w)
	canvas.Start(totalWidth, height)

	// Define clip path for rounded corners
	canvas.Def()
	canvas.ClipPath(`id="clip"`)
	canvas.Roundrect(0, 0, totalWidth, height, radius, radius)
	canvas.ClipEnd()
	canvas.DefEnd()

	// Apply clip path
	canvas.Group(`clip-path="url(#clip)"`)

	// Label section (left) - dark background
	labelBg := "#555"
	if opts.DarkMode {
		labelBg = "#333"
	}
	canvas.Rect(0, 0, labelWidth, height, fmt.Sprintf(styleFillFmt, labelBg))

	// Score section (middle) - colored by score
	canvas.Rect(labelWidth, 0, scoreWidth, height, fmt.Sprintf(styleFillFmt, scoreColor))

	// Status section (right) - same color as score
	canvas.Rect(labelWidth+scoreWidth, 0, statusWidth, height, fmt.Sprintf(styleFillFmt, scoreColor))

	canvas.Gend()

	// Text styling
	textStyle := "font-family:Verdana,Geneva,DejaVu Sans,sans-serif;font-size:11px;fill:#fff"

	// Label text (with shadow for readability)
	// Shift text right if icon is present
	labelX := labelWidth / 2
	if opts.Icon != "" {
		labelX = iconWidth + (labelWidth-iconWidth)/2
	}
	textY := 14
	canvas.Text(labelX+1, textY+1, label, textStyle+styleShadowAnchor)
	canvas.Text(labelX, textY, label, textStyle+styleTextAnchor)

	// Render icon if provided
	if opts.Icon != "" {
		// Icon is rendered at 14x14, positioned with padding
		// Use a group with transform to scale from viewBox 0 0 14 14
		canvas.Group(`transform="translate(4,3)"`)
		canvas.Path(opts.Icon, "fill:#fff;fill-opacity:0.3", `transform="translate(0.5,0.5)"`)
		canvas.Path(opts.Icon, "fill:#fff")
		canvas.Gend()
	}

	// Score text
	scoreX := labelWidth + scoreWidth/2
	canvas.Text(scoreX+1, textY+1, scoreText, textStyle+styleShadowAnchor)
	canvas.Text(scoreX, textY, scoreText, textStyle+styleTextAnchor)

	// Status text
	statusX := labelWidth + scoreWidth + statusWidth/2
	canvas.Text(statusX+1, textY+1, statusText, textStyle+styleShadowAnchor)
	canvas.Text(statusX, textY, statusText, textStyle+styleTextAnchor)

	canvas.End()
	return nil
}
