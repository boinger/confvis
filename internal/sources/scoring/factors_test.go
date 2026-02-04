package scoring

import (
	"testing"
)

func TestVulnSeverityConfigs(t *testing.T) {
	counts := SeverityCounts{Critical: 1, High: 2, Medium: 3, Low: 4}
	penalties := [4]int{33, 20, 10, 5}
	weights := [4]int{40, 30, 20, 10}

	configs := VulnSeverityConfigs(counts, penalties, weights)

	if len(configs) != 4 {
		t.Fatalf("len(configs) = %d, want 4", len(configs))
	}

	tests := []struct {
		idx     int
		name    string
		count   int
		penalty int
		weight  int
	}{
		{0, "Critical Vulnerabilities", 1, 33, 40},
		{1, "High Vulnerabilities", 2, 20, 30},
		{2, "Medium Vulnerabilities", 3, 10, 20},
		{3, "Low Vulnerabilities", 4, 5, 10},
	}

	for _, tt := range tests {
		cfg := configs[tt.idx]
		if cfg.Name != tt.name {
			t.Errorf("configs[%d].Name = %q, want %q", tt.idx, cfg.Name, tt.name)
		}
		if cfg.Count != tt.count {
			t.Errorf("configs[%d].Count = %d, want %d", tt.idx, cfg.Count, tt.count)
		}
		if cfg.Penalty != tt.penalty {
			t.Errorf("configs[%d].Penalty = %d, want %d", tt.idx, cfg.Penalty, tt.penalty)
		}
		if cfg.Weight != tt.weight {
			t.Errorf("configs[%d].Weight = %d, want %d", tt.idx, cfg.Weight, tt.weight)
		}
	}
}

func TestBuildSeverityFactors(t *testing.T) {
	configs := []SeverityConfig{
		{Name: "Critical Vulnerabilities", Count: 1, Penalty: 33, Weight: 40},
		{Name: "High Vulnerabilities", Count: 0, Penalty: 20, Weight: 30},
	}

	factors := BuildSeverityFactors(configs, "")

	if len(factors) != 2 {
		t.Fatalf("len(factors) = %d, want 2", len(factors))
	}

	// Critical: 1 issue, penalty 33 -> score = 67
	if factors[0].Score != 67 {
		t.Errorf("factors[0].Score = %d, want 67", factors[0].Score)
	}
	if factors[0].Weight != 40 {
		t.Errorf("factors[0].Weight = %d, want 40", factors[0].Weight)
	}
	if factors[0].Description != "1 critical" {
		t.Errorf("factors[0].Description = %q, want %q", factors[0].Description, "1 critical")
	}

	// High: 0 issues -> score = 100
	if factors[1].Score != 100 {
		t.Errorf("factors[1].Score = %d, want 100", factors[1].Score)
	}
	if factors[1].Description != "0 high" {
		t.Errorf("factors[1].Description = %q, want %q", factors[1].Description, "0 high")
	}
}

func TestBuildSeverityFactors_WithURL(t *testing.T) {
	configs := []SeverityConfig{
		{Name: "Critical Vulnerabilities", Count: 0, Penalty: 33, Weight: 40},
	}

	url := "https://example.com/alerts"
	factors := BuildSeverityFactors(configs, url)

	if factors[0].URL != url {
		t.Errorf("factors[0].URL = %q, want %q", factors[0].URL, url)
	}
}

func TestBuildSeverityFactors_NoURL(t *testing.T) {
	configs := []SeverityConfig{
		{Name: "Critical Vulnerabilities", Count: 0, Penalty: 33, Weight: 40},
	}

	factors := BuildSeverityFactors(configs, "")

	if factors[0].URL != "" {
		t.Errorf("factors[0].URL = %q, want empty", factors[0].URL)
	}
}

func TestExtractSeverity(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Critical Vulnerabilities", "critical"},
		{"High Vulnerabilities", "high"},
		{"Medium Vulnerabilities", "medium"},
		{"Low Vulnerabilities", "low"},
		{"Error Findings", "errors"},
		{"Warning Findings", "warnings"},
		{"Info Findings", "info"},
		{"Unknown Name", "issues"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSeverity(tt.name)
			if got != tt.want {
				t.Errorf("extractSeverity(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
