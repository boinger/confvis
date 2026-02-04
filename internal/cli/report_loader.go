package cli

import (
	"fmt"
	"io"

	"github.com/boinger/confvis/internal/confidence"
)

// ReportLoader loads confidence reports from stdin or files.
type ReportLoader struct {
	FS     FileSystem
	Stdin  io.Reader
	Config string
	Format confidence.Format // Optional: FormatAuto (default), FormatJSON, FormatYAML
}

// ParseInputFormat converts a string input format to confidence.Format.
func ParseInputFormat(s string) (confidence.Format, error) {
	switch s {
	case "auto":
		return confidence.FormatAuto, nil
	case "json":
		return confidence.FormatJSON, nil
	case "yaml":
		return confidence.FormatYAML, nil
	default:
		return confidence.FormatAuto, fmt.Errorf("invalid input-format %q: must be auto, json, or yaml", s)
	}
}

// LoadReport loads a confidence report from stdin or file path.
func (l *ReportLoader) LoadReport() (*confidence.Report, error) {
	format := l.Format
	if format == "" {
		format = confidence.FormatAuto
	}
	if l.Config == "-" {
		if format == confidence.FormatAuto {
			format = confidence.FormatJSON
		}
		report, err := confidence.ParseWithFormat(l.Stdin, format)
		if err != nil {
			return nil, fmt.Errorf("parsing config: %w", err)
		}
		return report, nil
	}
	reader, detectedFormat, err := openConfigFile(l.FS, l.Config, format)
	if err != nil {
		return nil, fmt.Errorf("opening config: %w", err)
	}
	report, err := confidence.ParseWithFormat(reader, detectedFormat)
	if err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return report, nil
}
