package scoring

import "testing"

func TestSeverityScore(t *testing.T) {
	tests := []struct {
		name    string
		count   int
		penalty int
		want    int
	}{
		{"no issues", 0, 20, 100},
		{"one issue with penalty 20", 1, 20, 80},
		{"three issues with penalty 20", 3, 20, 40},
		{"five issues with penalty 20", 5, 20, 0},
		{"six issues (capped at 0)", 6, 20, 0},
		{"one high (penalty 20)", 1, 20, 80},
		{"two high (penalty 20)", 2, 20, 60},
		{"one critical (penalty 33)", 1, 33, 67},
		{"three critical (penalty 33)", 3, 33, 1},
		{"four critical (penalty 33, capped)", 4, 33, 0},
		{"one medium (penalty 10)", 1, 10, 90},
		{"five medium (penalty 10)", 5, 10, 50},
		{"one low (penalty 5)", 1, 5, 95},
		{"many low (penalty 5)", 25, 5, 0},
		{"one info (penalty 2)", 1, 2, 98},
		{"fifty info (penalty 2)", 50, 2, 0},
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

func TestCountSeverity(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		wantC    int
		wantH    int
		wantM    int
		wantL    int
	}{
		{"critical", "critical", 1, 0, 0, 0},
		{"high", "high", 0, 1, 0, 0},
		{"medium", "medium", 0, 0, 1, 0},
		{"low", "low", 0, 0, 0, 1},
		{"CRITICAL uppercase", "CRITICAL", 1, 0, 0, 0},
		{"High mixed case", "High", 0, 1, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counts := &SeverityCounts{}
			CountSeverity(counts, tt.severity, "test", false)
			if counts.Critical != tt.wantC {
				t.Errorf("Critical = %d, want %d", counts.Critical, tt.wantC)
			}
			if counts.High != tt.wantH {
				t.Errorf("High = %d, want %d", counts.High, tt.wantH)
			}
			if counts.Medium != tt.wantM {
				t.Errorf("Medium = %d, want %d", counts.Medium, tt.wantM)
			}
			if counts.Low != tt.wantL {
				t.Errorf("Low = %d, want %d", counts.Low, tt.wantL)
			}
		})
	}
}

func TestCountSeverity_UnknownSeverity(t *testing.T) {
	counts := &SeverityCounts{}
	CountSeverity(counts, "unknown", "test", true)

	if counts.Critical != 0 || counts.High != 0 || counts.Medium != 0 || counts.Low != 0 {
		t.Errorf("unknown severity should not increment any counter, got %+v", counts)
	}
}

func TestCountSeverity_EmptySeverity(t *testing.T) {
	counts := &SeverityCounts{}
	CountSeverity(counts, "", "test", true)

	if counts.Critical != 0 || counts.High != 0 || counts.Medium != 0 || counts.Low != 0 {
		t.Errorf("empty severity should not increment any counter, got %+v", counts)
	}
}

func TestDefaultPenalties(t *testing.T) {
	got := DefaultPenalties()
	want := [4]int{33, 20, 10, 5}
	if got != want {
		t.Errorf("DefaultPenalties() = %v, want %v", got, want)
	}
}

func TestDefaultWeights(t *testing.T) {
	got := DefaultWeights()
	want := [4]int{40, 30, 20, 10}
	if got != want {
		t.Errorf("DefaultWeights() = %v, want %v", got, want)
	}
}
