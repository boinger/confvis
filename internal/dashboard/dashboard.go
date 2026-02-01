// Package dashboard provides HTML dashboard generation for confidence reports.
package dashboard

import (
	"embed"
	"html/template"
	"io"

	"github.com/boinger/confvis/internal/confidence"
	"github.com/boinger/confvis/internal/gauge"
)

//go:embed templates/*.html
var templateFS embed.FS

// Options configures dashboard generation.
type Options struct {
	DarkMode bool
}

// TemplateData holds data passed to the dashboard template.
type TemplateData struct {
	Report   *confidence.Report
	GaugeSVG template.HTML
	DarkMode bool
}

// Generate creates an HTML dashboard for the given report.
func Generate(w io.Writer, report *confidence.Report, opts Options) error {
	// Generate the gauge SVG
	gaugeOpts := gauge.Options{
		Width:    200,
		Height:   120,
		DarkMode: opts.DarkMode,
	}
	gaugeSVG, err := gauge.GenerateToString(report, gaugeOpts)
	if err != nil {
		return err
	}

	// Parse and execute template
	tmpl, err := template.ParseFS(templateFS, "templates/dashboard.html")
	if err != nil {
		return err
	}

	data := TemplateData{
		Report:   report,
		GaugeSVG: template.HTML(gaugeSVG),
		DarkMode: opts.DarkMode,
	}

	return tmpl.Execute(w, data)
}
