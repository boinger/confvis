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

// ScoreColor returns the appropriate color for a given score using the specified thresholds.
func (cs ColorScheme) ScoreColor(score, greenAbove, yellowAbove int) string {
	switch {
	case score >= greenAbove:
		return cs.Success
	case score >= yellowAbove:
		return cs.Warning
	default:
		return cs.Danger
	}
}

// Minimal returns a clean, simple color scheme with subtle colors.
func Minimal() ColorScheme {
	return ColorScheme{
		Background:  "#fafafa",
		TrackColor:  "#e0e0e0",
		TextPrimary: "#333333",
		TextMuted:   "#666666",
		Success:     "#4caf50",
		Warning:     "#ff9800",
		Danger:      "#f44336",
		Border:      "#e0e0e0",
	}
}

// MinimalDark returns a minimal color scheme for dark mode.
func MinimalDark() ColorScheme {
	return ColorScheme{
		Background:  "#1a1a1a",
		TrackColor:  "#333333",
		TextPrimary: "#e0e0e0",
		TextMuted:   "#999999",
		Success:     "#66bb6a",
		Warning:     "#ffb74d",
		Danger:      "#ef5350",
		Border:      "#333333",
	}
}

// Corporate returns a professional, muted color scheme.
func Corporate() ColorScheme {
	return ColorScheme{
		Background:  "#f5f5f5",
		TrackColor:  "#d9d9d9",
		TextPrimary: "#262626",
		TextMuted:   "#595959",
		Success:     "#52c41a",
		Warning:     "#faad14",
		Danger:      "#ff4d4f",
		Border:      "#d9d9d9",
	}
}

// CorporateDark returns a corporate color scheme for dark mode.
func CorporateDark() ColorScheme {
	return ColorScheme{
		Background:  "#141414",
		TrackColor:  "#303030",
		TextPrimary: "#f0f0f0",
		TextMuted:   "#8c8c8c",
		Success:     "#73d13d",
		Warning:     "#ffc53d",
		Danger:      "#ff7875",
		Border:      "#303030",
	}
}

// HighContrast returns an accessibility-focused high contrast color scheme.
func HighContrast() ColorScheme {
	return ColorScheme{
		Background:  "#ffffff",
		TrackColor:  "#767676",
		TextPrimary: "#000000",
		TextMuted:   "#333333",
		Success:     "#008000",
		Warning:     "#806600",
		Danger:      "#cc0000",
		Border:      "#000000",
	}
}

// HighContrastDark returns a high contrast color scheme for dark mode.
func HighContrastDark() ColorScheme {
	return ColorScheme{
		Background:  "#000000",
		TrackColor:  "#767676",
		TextPrimary: "#ffffff",
		TextMuted:   "#cccccc",
		Success:     "#00ff00",
		Warning:     "#ffff00",
		Danger:      "#ff0000",
		Border:      "#ffffff",
	}
}

// StyleNames returns the list of available style names.
func StyleNames() []string {
	return []string{"github", "minimal", "corporate", "high-contrast"}
}

// GetColorScheme returns the color scheme for the given style name and dark mode setting.
// Returns GitHubLight/GitHubDark for unknown style names.
func GetColorScheme(style string, darkMode bool) ColorScheme {
	switch style {
	case "minimal":
		if darkMode {
			return MinimalDark()
		}
		return Minimal()
	case "corporate":
		if darkMode {
			return CorporateDark()
		}
		return Corporate()
	case "high-contrast":
		if darkMode {
			return HighContrastDark()
		}
		return HighContrast()
	default: // "github" or unknown
		if darkMode {
			return GitHubDark()
		}
		return GitHubLight()
	}
}
