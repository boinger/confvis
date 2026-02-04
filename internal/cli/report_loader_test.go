package cli

import (
	"strings"
	"testing"
)

func TestReportLoader_LoadReport_FromStdin(t *testing.T) {
	stdin := strings.NewReader(`{"title": "Test", "score": 85, "threshold": 75}`)

	loader := &ReportLoader{
		FS:     &MockFileSystem{},
		Stdin:  stdin,
		Config: "-",
	}

	report, err := loader.LoadReport()
	if err != nil {
		t.Fatalf("LoadReport() error = %v", err)
	}

	if report.Title != "Test" {
		t.Errorf("Title = %q, want %q", report.Title, "Test")
	}
	if report.Score != 85 {
		t.Errorf("Score = %d, want 85", report.Score)
	}
}

func TestReportLoader_LoadReport_FromFile(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.SetFileContent("report.json", `{"title": "File Report", "score": 90, "threshold": 75}`)

	loader := &ReportLoader{
		FS:     mockFS,
		Stdin:  strings.NewReader(""),
		Config: "report.json",
	}

	report, err := loader.LoadReport()
	if err != nil {
		t.Fatalf("LoadReport() error = %v", err)
	}

	if report.Title != "File Report" {
		t.Errorf("Title = %q, want %q", report.Title, "File Report")
	}
	if report.Score != 90 {
		t.Errorf("Score = %d, want 90", report.Score)
	}
}

func TestReportLoader_LoadReport_FromYAML(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.SetFileContent("report.yaml", `title: YAML Report
score: 88
threshold: 70`)

	loader := &ReportLoader{
		FS:     mockFS,
		Stdin:  strings.NewReader(""),
		Config: "report.yaml",
	}

	report, err := loader.LoadReport()
	if err != nil {
		t.Fatalf("LoadReport() error = %v", err)
	}

	if report.Title != "YAML Report" {
		t.Errorf("Title = %q, want %q", report.Title, "YAML Report")
	}
}

func TestReportLoader_LoadReport_FileNotFound(t *testing.T) {
	mockFS := NewMockFileSystem()

	loader := &ReportLoader{
		FS:     mockFS,
		Stdin:  strings.NewReader(""),
		Config: "nonexistent.json",
	}

	_, err := loader.LoadReport()
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestReportLoader_LoadReport_InvalidJSON(t *testing.T) {
	stdin := strings.NewReader(`not valid json`)

	loader := &ReportLoader{
		FS:     &MockFileSystem{},
		Stdin:  stdin,
		Config: "-",
	}

	_, err := loader.LoadReport()
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
