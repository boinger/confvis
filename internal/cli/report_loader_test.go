package cli

import (
	"strings"
	"testing"

	"github.com/boinger/confvis/internal/confidence"
)

func TestReportLoader_LoadReport(t *testing.T) {
	tests := []struct {
		name       string
		config     string
		fileContent map[string]string
		stdin      string
		wantTitle  string
		wantScore  int
		wantErr    bool
	}{
		{
			name:      "from stdin",
			config:    "-",
			stdin:     `{"title": "Test", "score": 85, "threshold": 75}`,
			wantTitle: "Test",
			wantScore: 85,
		},
		{
			name:        "from JSON file",
			config:      "report.json",
			fileContent: map[string]string{"report.json": `{"title": "File Report", "score": 90, "threshold": 75}`},
			wantTitle:   "File Report",
			wantScore:   90,
		},
		{
			name:        "from YAML file",
			config:      "report.yaml",
			fileContent: map[string]string{"report.yaml": "title: YAML Report\nscore: 88\nthreshold: 70"},
			wantTitle:   "YAML Report",
			wantScore:   88,
		},
		{
			name:    "file not found",
			config:  "nonexistent.json",
			wantErr: true,
		},
		{
			name:    "invalid JSON from stdin",
			config:  "-",
			stdin:   "not valid json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			for path, content := range tt.fileContent {
				mockFS.SetFileContent(path, content)
			}

			loader := &ReportLoader{
				FS:     mockFS,
				Stdin:  strings.NewReader(tt.stdin),
				Config: tt.config,
			}

			report, err := loader.LoadReport()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadReport() error = %v", err)
			}
			if report.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", report.Title, tt.wantTitle)
			}
			if report.ScoreValue() != tt.wantScore {
				t.Errorf("Score = %d, want %d", report.ScoreValue(), tt.wantScore)
			}
		})
	}
}

func TestParseInputFormat(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantFormat confidence.Format
		wantErr    bool
	}{
		{
			name:       "auto",
			input:      "auto",
			wantFormat: confidence.FormatAuto,
		},
		{
			name:       "json",
			input:      "json",
			wantFormat: confidence.FormatJSON,
		},
		{
			name:       "yaml",
			input:      "yaml",
			wantFormat: confidence.FormatYAML,
		},
		{
			name:    "invalid format",
			input:   "xml",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, err := ParseInputFormat(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseInputFormat() error = %v", err)
			}
			if format != tt.wantFormat {
				t.Errorf("ParseInputFormat() = %v, want %v", format, tt.wantFormat)
			}
		})
	}
}
