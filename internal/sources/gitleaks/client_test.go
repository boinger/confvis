package gitleaks

import "testing"

func TestClient_NewClient(t *testing.T) {
	// Default command
	c := NewClient("")
	if c.command != DefaultCommand {
		t.Errorf("NewClient(\"\").command = %q, want %q", c.command, DefaultCommand)
	}

	// Custom command
	c = NewClient("custom-gitleaks")
	if c.command != "custom-gitleaks" {
		t.Errorf("NewClient(custom).command = %q, want %q", c.command, "custom-gitleaks")
	}
}
