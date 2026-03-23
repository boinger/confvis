package gosec

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"
)

func TestClient_NewClient(t *testing.T) {
	// Default command
	c := NewClient("")
	if c.command != DefaultCommand {
		t.Errorf("NewClient(\"\").command = %q, want %q", c.command, DefaultCommand)
	}

	// Custom command
	c = NewClient("custom-gosec")
	if c.command != "custom-gosec" {
		t.Errorf("NewClient(custom).command = %q, want %q", c.command, "custom-gosec")
	}
}

func TestCheckGosecError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		stdout  []byte
		stderr  []byte
		wantNil bool
	}{
		{
			name:    "exit code 1 with valid JSON stdout",
			err:     &exec.ExitError{ProcessState: newExitState(1)},
			stdout:  []byte(`{"Issues": [], "Stats": {}}`),
			stderr:  nil,
			wantNil: true,
		},
		{
			name:    "exit code 1 without valid JSON stdout",
			err:     &exec.ExitError{ProcessState: newExitState(1)},
			stdout:  []byte("some error output"),
			stderr:  []byte("gosec failed"),
			wantNil: false,
		},
		{
			name:    "exit code 2",
			err:     &exec.ExitError{ProcessState: newExitState(2)},
			stdout:  nil,
			stderr:  []byte("fatal error"),
			wantNil: false,
		},
		{
			name:    "non-ExitError",
			err:     fmt.Errorf("command not found"),
			stdout:  nil,
			stderr:  nil,
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkGosecError(tt.err, tt.stdout, tt.stderr)
			if tt.wantNil && got != nil {
				t.Errorf("checkGosecError() = %v, want nil", got)
			}
			if !tt.wantNil && got == nil {
				t.Error("checkGosecError() = nil, want error")
			}
		})
	}
}

// newExitState creates a *os.ProcessState with the given exit code for testing.
// ProcessState is not directly constructable, so we run a real command.
func newExitState(code int) *os.ProcessState {
	var cmd *exec.Cmd
	if code == 0 {
		cmd = exec.Command("true")
	} else {
		cmd = exec.Command("sh", "-c", "exit "+strconv.Itoa(code))
	}
	_ = cmd.Run()
	return cmd.ProcessState
}
