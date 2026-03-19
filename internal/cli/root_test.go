package cli

import "testing"

func TestSetVersion(t *testing.T) {
	// Save and restore to avoid cross-test pollution.
	saved := rootCmd.Version
	defer func() { rootCmd.Version = saved }()

	SetVersion("1.2.3")
	if rootCmd.Version != "1.2.3" {
		t.Errorf("rootCmd.Version = %q, want %q", rootCmd.Version, "1.2.3")
	}
}
