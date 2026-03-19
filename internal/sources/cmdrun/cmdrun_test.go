package cmdrun

import (
	"context"
	"os/exec"
	"testing"
)

func TestRun(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		command     string
		args        []string
		toolName    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "simple command",
			command:  "echo",
			args:     []string{"hello"},
			toolName: "test",
			wantErr:  false,
		},
		{
			name:     "compound command",
			command:  "echo hello",
			args:     []string{"world"},
			toolName: "test",
			wantErr:  false,
		},
		{
			name:        "empty command",
			command:     "",
			args:        []string{},
			toolName:    "testtool",
			wantErr:     true,
			errContains: "empty testtool command",
		},
		{
			name:        "whitespace only command",
			command:     "   ",
			args:        []string{},
			toolName:    "testtool",
			wantErr:     true,
			errContains: "empty testtool command",
		},
		{
			name:     "command not found",
			command:  "nonexistent_command_12345",
			args:     []string{},
			toolName: "test",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Run(ctx, tt.command, tt.args, tt.toolName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.errContains != "" && err != nil && !containsString(err.Error(), tt.errContains) {
				t.Errorf("Run() error = %v, want error containing %q", err, tt.errContains)
			}
			// Result should never be nil - it's always returned for safe access
			if result == nil {
				t.Error("Run() returned nil result")
			}
		})
	}
}

func TestRunOutput(t *testing.T) {
	ctx := context.Background()

	result, err := Run(ctx, "echo", []string{"test output"}, "echo")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if string(result.Stdout) != "test output\n" {
		t.Errorf("Run() stdout = %q, want %q", string(result.Stdout), "test output\n")
	}
}

func TestFormatError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		stderr   []byte
		toolName string
		action   string
		want     string
	}{
		{
			name:     "with stderr",
			err:      context.DeadlineExceeded,
			stderr:   []byte("error details"),
			toolName: "grype",
			action:   "scan",
			want:     "grype scan failed: context deadline exceeded: error details",
		},
		{
			name:     "without stderr",
			err:      context.DeadlineExceeded,
			stderr:   []byte{},
			toolName: "trivy",
			action:   "scan",
			want:     "trivy scan failed: context deadline exceeded",
		},
		{
			name:     "stderr with whitespace",
			err:      context.DeadlineExceeded,
			stderr:   []byte("  trimmed  \n"),
			toolName: "semgrep",
			action:   "scan",
			want:     "semgrep scan failed: context deadline exceeded: trimmed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatError(tt.err, tt.stderr, tt.toolName, tt.action)
			if got.Error() != tt.want {
				t.Errorf("FormatError() = %q, want %q", got.Error(), tt.want)
			}
		})
	}
}

func TestCheckAcceptableExitCode_Accepted(t *testing.T) {
	// Run a command that exits with code 1 to get an *exec.ExitError
	err := exec.Command("sh", "-c", "exit 1").Run()
	if err == nil {
		t.Fatal("expected error from exit 1 command")
	}

	got := CheckAcceptableExitCode(err, []int{1}, nil, "tool", "scan")
	if got != nil {
		t.Errorf("CheckAcceptableExitCode() = %v, want nil for accepted code", got)
	}
}

func TestCheckAcceptableExitCode_Rejected(t *testing.T) {
	err := exec.Command("sh", "-c", "exit 2").Run()
	if err == nil {
		t.Fatal("expected error from exit 2 command")
	}

	got := CheckAcceptableExitCode(err, []int{1}, nil, "tool", "scan")
	if got == nil {
		t.Fatal("CheckAcceptableExitCode() = nil, want error for rejected code")
	}
	if !containsString(got.Error(), "tool scan failed") {
		t.Errorf("CheckAcceptableExitCode() = %q, want error containing %q", got.Error(), "tool scan failed")
	}
}

func TestCheckAcceptableExitCode_NotExitError(t *testing.T) {
	got := CheckAcceptableExitCode(context.DeadlineExceeded, []int{1}, nil, "tool", "scan")
	if got == nil {
		t.Fatal("CheckAcceptableExitCode() = nil, want error for non-ExitError")
	}
	if !containsString(got.Error(), "tool scan failed") {
		t.Errorf("CheckAcceptableExitCode() = %q, want error containing %q", got.Error(), "tool scan failed")
	}
	if !containsString(got.Error(), "context deadline exceeded") {
		t.Errorf("CheckAcceptableExitCode() = %q, want error containing %q", got.Error(), "context deadline exceeded")
	}
}

func TestParseJSONOutput_Valid(t *testing.T) {
	type testStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	input := []byte(`{"name":"test","value":42}`)
	got, err := ParseJSONOutput[testStruct](input, "tool")
	if err != nil {
		t.Fatalf("ParseJSONOutput() error = %v", err)
	}
	if got.Name != "test" {
		t.Errorf("ParseJSONOutput().Name = %q, want %q", got.Name, "test")
	}
	if got.Value != 42 {
		t.Errorf("ParseJSONOutput().Value = %d, want %d", got.Value, 42)
	}
}

func TestParseJSONOutput_Empty(t *testing.T) {
	type testStruct struct{}

	_, err := ParseJSONOutput[testStruct]([]byte("   "), "tool")
	if err == nil {
		t.Fatal("ParseJSONOutput() = nil, want error for empty input")
	}
	if !containsString(err.Error(), "produced no output") {
		t.Errorf("ParseJSONOutput() error = %q, want error containing %q", err.Error(), "produced no output")
	}
}

func TestParseJSONOutput_Invalid(t *testing.T) {
	type testStruct struct{}

	_, err := ParseJSONOutput[testStruct]([]byte("not json"), "tool")
	if err == nil {
		t.Fatal("ParseJSONOutput() = nil, want error for invalid JSON")
	}
	if !containsString(err.Error(), "parsing") {
		t.Errorf("ParseJSONOutput() error = %q, want error containing %q", err.Error(), "parsing")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
