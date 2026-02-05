package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/boinger/confvis/internal/baseline"
	"github.com/boinger/confvis/internal/confidence"
)

// intPtr is a test helper that returns a pointer to an int.
func intPtr(i int) *int { return &i }

func TestParseBaselineConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		fileContent map[string]string
		stdin       string
		wantScore   int
	}{
		{
			name:      "from stdin",
			config:    "-",
			stdin:     `{"title": "Test", "score": 85, "threshold": 75}`,
			wantScore: 85,
		},
		{
			name:        "from file",
			config:      "report.json",
			fileContent: map[string]string{"report.json": `{"title": "File", "score": 90, "threshold": 75}`},
			wantScore:   90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			for path, content := range tt.fileContent {
				mockFS.SetFileContent(path, content)
			}

			deps := &BaselineDeps{
				FS:     mockFS,
				Stdin:  strings.NewReader(tt.stdin),
				Config: tt.config,
			}

			report, err := parseBaselineConfig(deps)
			if err != nil {
				t.Fatalf("parseBaselineConfig() error = %v", err)
			}
			if report.ScoreValue() != tt.wantScore {
				t.Errorf("Score = %d, want %d", report.ScoreValue(), tt.wantScore)
			}
		})
	}
}

func TestTruncateCommit(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"abcdefghij", "abcdefg"},
		{"abc", "abc"},
		{"", ""},
		{"1234567", "1234567"},
		{"12345678", "1234567"},
	}

	for _, tt := range tests {
		if got := truncateCommit(tt.input); got != tt.want {
			t.Errorf("truncateCommit(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGetBaselineRef(t *testing.T) {
	viper.Reset()
	if got := getBaselineRef(); got != baseline.DefaultBaselineRef {
		t.Errorf("getBaselineRef() = %q, want %q", got, baseline.DefaultBaselineRef)
	}

	viper.Set("baseline.ref", "refs/custom/baseline")
	if got := getBaselineRef(); got != "refs/custom/baseline" {
		t.Errorf("getBaselineRef() = %q, want %q", got, "refs/custom/baseline")
	}
}

func TestGetBaselineFile(t *testing.T) {
	viper.Reset()
	if got := getBaselineFile(); got != "" {
		t.Errorf("getBaselineFile() = %q, want empty", got)
	}

	viper.Set("baseline.file", "baseline.json")
	if got := getBaselineFile(); got != "baseline.json" {
		t.Errorf("getBaselineFile() = %q, want %q", got, "baseline.json")
	}
}

func TestOutputBaseline(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		baseline *baseline.Baseline
		contains string
	}{
		{
			name:   "text format",
			format: "text",
			baseline: &baseline.Baseline{
				Report:  confidence.Report{Score: intPtr(85)},
				SavedAt: "2024-01-01T00:00:00Z",
				Commit:  "abc1234567890",
				Branch:  "main",
			},
			contains: "85%",
		},
		{
			name:     "json format",
			format:   "json",
			baseline: &baseline.Baseline{Report: confidence.Report{Score: intPtr(85)}},
			contains: `"score": 85`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			deps := &BaselineDeps{Stdout: &buf, Format: tt.format}

			if err := outputBaseline(deps, tt.baseline); err != nil {
				t.Fatalf("outputBaseline() error = %v", err)
			}
			if !strings.Contains(buf.String(), tt.contains) {
				t.Errorf("output should contain %q", tt.contains)
			}
		})
	}
}

func TestBaselineShowImpl_InvalidFormat(t *testing.T) {
	deps := &BaselineDeps{Format: "invalid"}
	if err := baselineShowImpl(deps); err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestLoadBaseline(t *testing.T) {
	tests := []struct {
		name         string
		file         string
		ref          string
		isGitRepo    func() bool
		fileReader   func(string) (*baseline.Baseline, error)
		gitRefReader func(string) (*baseline.Baseline, error)
		wantScore    int
		wantSource   string
		wantErr      bool
	}{
		{
			name:      "not in git repo",
			isGitRepo: func() bool { return false },
			wantErr:   true,
		},
		{
			name: "from file",
			file: "baseline.json",
			fileReader: func(string) (*baseline.Baseline, error) {
				return &baseline.Baseline{Report: confidence.Report{Score: intPtr(75)}}, nil
			},
			wantScore:  75,
			wantSource: "baseline.json",
		},
		{
			name:      "from git ref",
			ref:       "refs/confvis/baseline",
			isGitRepo: func() bool { return true },
			gitRefReader: func(string) (*baseline.Baseline, error) {
				return &baseline.Baseline{Report: confidence.Report{Score: intPtr(80)}}, nil
			},
			wantScore:  80,
			wantSource: "refs/confvis/baseline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := &BaselineDeps{
				File:         tt.file,
				Ref:          tt.ref,
				IsGitRepo:    tt.isGitRepo,
				FileReader:   tt.fileReader,
				GitRefReader: tt.gitRefReader,
			}

			b, source, err := loadBaseline(deps)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("loadBaseline() error = %v", err)
			}
			if b.ScoreValue() != tt.wantScore {
				t.Errorf("Score = %d, want %d", b.Score, tt.wantScore)
			}
			if source != tt.wantSource {
				t.Errorf("source = %q, want %q", source, tt.wantSource)
			}
		})
	}
}

func TestSaveBaseline(t *testing.T) {
	tests := []struct {
		name       string
		dryRun     bool
		file       string
		ref        string
		verbose    bool
		fileWriter func(string, *baseline.Baseline) error
		gitWriter  func(string, *baseline.Baseline) error
		baseline   *baseline.Baseline
		contains   string
		checkSaved func(t *testing.T, path, ref string)
	}{
		{
			name:     "dry run",
			dryRun:   true,
			file:     "baseline.json",
			baseline: &baseline.Baseline{Report: confidence.Report{Score: intPtr(85), Title: "Test"}},
			contains: "DRY RUN",
		},
		{
			name:    "to file",
			file:    "baseline.json",
			verbose: true,
			baseline: &baseline.Baseline{
				Report: confidence.Report{Score: intPtr(85)},
				Commit: "abc123",
			},
		},
		{
			name:     "to git ref",
			ref:      "refs/confvis/baseline",
			verbose:  true,
			baseline: &baseline.Baseline{Report: confidence.Report{Score: intPtr(85)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			var savedPath, savedRef string

			deps := &BaselineDeps{
				Stdout:  &buf,
				DryRun:  tt.dryRun,
				File:    tt.file,
				Ref:     tt.ref,
				Verbose: tt.verbose,
				FileWriter: func(path string, b *baseline.Baseline) error {
					savedPath = path
					return nil
				},
				GitRefWriter: func(ref string, b *baseline.Baseline) error {
					savedRef = ref
					return nil
				},
			}

			if err := saveBaseline(deps, tt.baseline, tt.file != ""); err != nil {
				t.Fatalf("saveBaseline() error = %v", err)
			}

			if tt.contains != "" && !strings.Contains(buf.String(), tt.contains) {
				t.Errorf("output should contain %q", tt.contains)
			}
			if tt.file != "" && !tt.dryRun && savedPath != tt.file {
				t.Errorf("saved to %q, want %q", savedPath, tt.file)
			}
			if tt.ref != "" && !tt.dryRun && savedRef != tt.ref {
				t.Errorf("saved ref %q, want %q", savedRef, tt.ref)
			}
		})
	}
}
