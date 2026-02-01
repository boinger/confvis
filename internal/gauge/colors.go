// Package gauge provides SVG gauge generation for confidence scores.
package gauge

// ColorScheme defines colors for gauge rendering.
type ColorScheme struct {
	Background  string
	TrackColor  string
	TextPrimary string
	TextMuted   string
	Success     string
	Warning     string
	Danger      string
	Border      string
}

// GitHubLight returns the light mode color scheme inspired by GitHub.
func GitHubLight() ColorScheme {
	return ColorScheme{
		Background:  "#ffffff",
		TrackColor:  "#e1e4e8",
		TextPrimary: "#24292f",
		TextMuted:   "#57606a",
		Success:     "#1a7f37",
		Warning:     "#9a6700",
		Danger:      "#cf222e",
		Border:      "#d0d7de",
	}
}

// GitHubDark returns the dark mode color scheme inspired by GitHub.
func GitHubDark() ColorScheme {
	return ColorScheme{
		Background:  "#0d1117",
		TrackColor:  "#30363d",
		TextPrimary: "#e6edf3",
		TextMuted:   "#8b949e",
		Success:     "#3fb950",
		Warning:     "#d29922",
		Danger:      "#f85149",
		Border:      "#30363d",
	}
}

// ScoreColor returns the appropriate color for a given score.
func (cs ColorScheme) ScoreColor(score int) string {
	switch {
	case score >= 75:
		return cs.Success
	case score >= 50:
		return cs.Warning
	default:
		return cs.Danger
	}
}
