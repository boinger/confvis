package main

import "testing"

func TestResolveVersion_AlreadySet(t *testing.T) {
	// Non-"dev" passes through unchanged regardless of buildInfoFn.
	got := resolveVersion("v1.2.3", func() (string, bool) {
		return "v9.9.9", true
	})
	if got != "v1.2.3" {
		t.Errorf("resolveVersion(v1.2.3) = %q, want %q", got, "v1.2.3")
	}
}

func TestResolveVersion_Dev_WithBuildInfo(t *testing.T) {
	got := resolveVersion("dev", func() (string, bool) {
		return "v1.5.0", true
	})
	if got != "v1.5.0" {
		t.Errorf("resolveVersion(dev) = %q, want %q", got, "v1.5.0")
	}
}

func TestResolveVersion_Dev_NoBuildInfo(t *testing.T) {
	got := resolveVersion("dev", func() (string, bool) {
		return "", false
	})
	if got != "dev" {
		t.Errorf("resolveVersion(dev) = %q, want %q", got, "dev")
	}
}

func TestResolveVersion_Dev_EmptyVersion(t *testing.T) {
	got := resolveVersion("dev", func() (string, bool) {
		return "", true
	})
	if got != "dev" {
		t.Errorf("resolveVersion(dev) = %q, want %q", got, "dev")
	}
}

func TestDefaultBuildInfo(t *testing.T) {
	// debug.ReadBuildInfo() succeeds in test binaries, returning
	// the test binary's module info.
	ver, ok := defaultBuildInfo()
	if !ok {
		t.Fatal("defaultBuildInfo() returned ok=false; expected true in test binary")
	}
	// In test binaries the version is typically empty or "(devel)".
	// We just verify the function runs without error and returns ok=true.
	_ = ver
}
