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
