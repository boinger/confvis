package grype

import (
	"testing"
)

func TestSeverityScore(t *testing.T) {
	tests := []struct {
		name    string
		count   int
		penalty int
		want    int
	}{
		{"no vulnerabilities", 0, 33, 100},
		{"one critical", 1, 33, 67},
		{"two critical", 2, 33, 34},
		{"three critical", 3, 33, 1},
		{"four critical (capped)", 4, 33, 0},
		{"one high", 1, 20, 80},
		{"one medium", 1, 10, 90},
		{"one low", 1, 5, 95},
		{"many low", 25, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SeverityScore(tt.count, tt.penalty)
			if got != tt.want {
				t.Errorf("SeverityScore(%d, %d) = %d, want %d", tt.count, tt.penalty, got, tt.want)
			}
		})
	}
}

func TestCountFromMatches(t *testing.T) {
	matches := []Match{
		{Vulnerability: Vulnerability{Severity: "Critical"}},
		{Vulnerability: Vulnerability{Severity: "Critical"}},
		{Vulnerability: Vulnerability{Severity: "High"}},
		{Vulnerability: Vulnerability{Severity: "Medium"}},
		{Vulnerability: Vulnerability{Severity: "Medium"}},
		{Vulnerability: Vulnerability{Severity: "Medium"}},
		{Vulnerability: Vulnerability{Severity: "Low"}},
		{Vulnerability: Vulnerability{Severity: "Negligible"}},
		{Vulnerability: Vulnerability{Severity: "Unknown"}},
	}

	counts := CountFromMatches(matches)

	if counts.Critical != 2 {
		t.Errorf("Critical = %d, want 2", counts.Critical)
	}
	if counts.High != 1 {
		t.Errorf("High = %d, want 1", counts.High)
	}
	if counts.Medium != 3 {
		t.Errorf("Medium = %d, want 3", counts.Medium)
	}
	if counts.Low != 2 { // Low + Negligible
		t.Errorf("Low = %d, want 2", counts.Low)
	}
	if counts.Unknown != 1 {
		t.Errorf("Unknown = %d, want 1", counts.Unknown)
	}
}

func TestDeriveTitle(t *testing.T) {
	tests := []struct {
		target string
		want   string
	}{
		// For paths, derive to base name
		{"./src", "src"},
		// For container images, preserve the full name
		{"alpine:3.18", "alpine:3.18"},
		{"nginx:latest", "nginx:latest"},
		{"myrepo/myimage:v1.0", "myrepo/myimage:v1.0"},
		{"myimage", "myimage"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			got := deriveTitle(tt.target)
			if got != tt.want {
				t.Errorf("deriveTitle(%q) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}

func TestLooksLikeContainerImage(t *testing.T) {
	tests := []struct {
		target string
		want   bool
	}{
		{".", false},
		{"..", false},
		{"./src", false},
		{"/absolute/path", false},
		{"alpine:3.18", true},
		{"nginx:latest", true},
		{"myimage", true},            // Single word could be an image
		{"myrepo/myimage:v1.0", true}, // Image with registry/repo and tag
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			got := looksLikeContainerImage(tt.target)
			if got != tt.want {
				t.Errorf("looksLikeContainerImage(%q) = %v, want %v", tt.target, got, tt.want)
			}
		})
	}
}

func TestSource_Name(t *testing.T) {
	s := &Source{}
	if s.Name() != sourceName {
		t.Errorf("Name() = %q, want %q", s.Name(), sourceName)
	}
}
