package cli

import (
	"fmt"
	"io"

	"github.com/boinger/confvis/internal/confidence"
)

// ReportLoader provides dependencies for loading confidence reports.
type ReportLoader struct {
	FS     FileSystem
	Stdin  io.Reader
	Config string
}

// LoadReport loads a confidence report from stdin or file path.
// If Config is "-", reads JSON from stdin.
// Otherwise, opens the file and auto-detects format from extension.
func (l *ReportLoader) LoadReport() (*confidence.Report, error) {
	if l.Config == "-" {
		report, err := confidence.ParseWithFormat(l.Stdin, confidence.FormatJSON)
		if err != nil {
			return nil, fmt.Errorf("parsing config: %w", err)
		}
		return report, nil
	}

	reader, format, err := openConfigFile(l.FS, l.Config, confidence.FormatAuto)
	if err != nil {
		return nil, fmt.Errorf("opening config: %w", err)
	}
	report, err := confidence.ParseWithFormat(reader, format)
	if err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return report, nil
}
