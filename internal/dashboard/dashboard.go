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

// MultiOptions configures multi-report dashboard generation.
type MultiOptions struct {
	DarkMode bool
}

// TemplateData holds data passed to the dashboard template.
type TemplateData struct {
	Report   *confidence.Report
	GaugeSVG template.HTML
	DarkMode bool
}

// ReportSummary holds a report with its weight and source path.
type ReportSummary struct {
	Report   *confidence.Report
	Weight   int
	Path     string
	GaugeSVG template.HTML
}

// MultiTemplateData holds data passed to the multi-dashboard template.
type MultiTemplateData struct {
	Reports      []ReportSummary
	Aggregate    *confidence.Report
	AggregateSVG template.HTML
	DarkMode     bool
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

	// Parse and execute template (include base.html for shared styles)
	tmpl, err := template.ParseFS(templateFS, "templates/base.html", "templates/dashboard.html")
	if err != nil {
		return err
	}

	data := TemplateData{
		Report:   report,
		GaugeSVG: template.HTML(gaugeSVG),
		DarkMode: opts.DarkMode,
	}

	return tmpl.ExecuteTemplate(w, "dashboard.html", data)
}

// GenerateMulti creates an HTML dashboard for multiple reports with an aggregate view.
func GenerateMulti(w io.Writer, reports []ReportSummary, aggregate *confidence.Report, opts MultiOptions) error {
	// Generate gauge SVGs for each report
	gaugeOpts := gauge.Options{
		Width:    200,
		Height:   120,
		DarkMode: opts.DarkMode,
	}

	for i := range reports {
		svg, err := gauge.GenerateToString(reports[i].Report, gaugeOpts)
		if err != nil {
			return err
		}
		reports[i].GaugeSVG = template.HTML(svg)
	}

	// Generate aggregate gauge
	aggregateSVG, err := gauge.GenerateToString(aggregate, gaugeOpts)
	if err != nil {
		return err
	}

	// Parse and execute template (include base.html for shared styles)
	tmpl, err := template.ParseFS(templateFS, "templates/base.html", "templates/multi-dashboard.html")
	if err != nil {
		return err
	}

	data := MultiTemplateData{
		Reports:      reports,
		Aggregate:    aggregate,
		AggregateSVG: template.HTML(aggregateSVG),
		DarkMode:     opts.DarkMode,
	}

	return tmpl.ExecuteTemplate(w, "multi-dashboard.html", data)
}
