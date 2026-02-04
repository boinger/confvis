package cmdrun

import (
	"context"
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
