package cli

import (
	"testing"

	"github.com/boinger/confvis/internal/confidence"
)

func intPtrT(i int) *int { return &i }

func TestCheckThresholds_AllPass(t *testing.T) {
	report := &confidence.Report{Score: intPtrT(85), Title: "Test"}
	result := CheckThresholds(report, nil, 0, ThresholdConfig{FailUnder: 80})
	if !result.Passed() {
		t.Error("expected all checks to pass")
	}
	if !result.ScorePassed {
		t.Error("score should pass")
	}
}

func TestCheckThresholds_ScoreFails(t *testing.T) {
	report := &confidence.Report{Score: intPtrT(70), Title: "Test"}
	result := CheckThresholds(report, nil, 0, ThresholdConfig{FailUnder: 80})
	if result.Passed() {
		t.Error("expected checks to fail")
	}
	if result.ScorePassed {
		t.Error("score should fail")
	}
	if !result.BaselinePassed {
		t.Error("baseline should pass when not configured")
	}
}

func TestCheckThresholds_RegressionFails(t *testing.T) {
	report := &confidence.Report{Score: intPtrT(70), Title: "Test"}
	baseline := &confidence.Report{Score: intPtrT(85), Title: "Baseline"}
	result := CheckThresholds(report, baseline, -15, ThresholdConfig{FailOnRegression: true})
	if result.Passed() {
		t.Error("expected checks to fail")
	}
	if !result.BaselinePassed {
		// Expected: regression detected
	} else {
		t.Error("baseline should fail for regression")
	}
}

func TestCheckThresholds_RegressionPasses_ScoreImproved(t *testing.T) {
	report := &confidence.Report{Score: intPtrT(90), Title: "Test"}
	baseline := &confidence.Report{Score: intPtrT(85), Title: "Baseline"}
	result := CheckThresholds(report, baseline, 5, ThresholdConfig{FailOnRegression: true})
	if !result.Passed() {
		t.Error("expected all checks to pass for improved score")
	}
}

func TestCheckThresholds_RegressionIgnoredWhenNotConfigured(t *testing.T) {
	report := &confidence.Report{Score: intPtrT(70), Title: "Test"}
	baseline := &confidence.Report{Score: intPtrT(85), Title: "Baseline"}
	result := CheckThresholds(report, baseline, -15, ThresholdConfig{FailUnder: 50})
	if !result.BaselinePassed {
		t.Error("baseline check should pass when FailOnRegression is false")
	}
}

func TestCheckThresholds_RegressionIgnoredWhenNoBaseline(t *testing.T) {
	report := &confidence.Report{Score: intPtrT(70), Title: "Test"}
	result := CheckThresholds(report, nil, 0, ThresholdConfig{FailOnRegression: true})
	if !result.BaselinePassed {
		t.Error("baseline check should pass when no baseline provided")
	}
}

func TestCheckThresholds_FactorFails(t *testing.T) {
	report := &confidence.Report{
		Score: intPtrT(85),
		Title: "Test",
		Factors: []confidence.Factor{
			{Name: "Coverage", Score: 70, Weight: 50},
			{Name: "Security", Score: 95, Weight: 50},
		},
	}
	result := CheckThresholds(report, nil, 0, ThresholdConfig{
		FactorThresholds: map[string]int{"Coverage": 80},
	})
	if result.Passed() {
		t.Error("expected checks to fail")
	}
	if result.FactorsPassed {
		t.Error("factor check should fail")
	}
	if len(result.FactorFailures) != 1 {
		t.Errorf("expected 1 factor failure, got %d", len(result.FactorFailures))
	}
}

func TestCheckThresholds_FailUnderZero_NoCheck(t *testing.T) {
	report := &confidence.Report{Score: intPtrT(0), Title: "Test"}
	result := CheckThresholds(report, nil, 0, ThresholdConfig{FailUnder: 0})
	if !result.ScorePassed {
		t.Error("score check should pass when FailUnder is 0")
	}
}

func TestCheckThresholds_Combined_AllFail(t *testing.T) {
	report := &confidence.Report{
		Score: intPtrT(50),
		Title: "Test",
		Factors: []confidence.Factor{
			{Name: "Coverage", Score: 30, Weight: 100},
		},
	}
	baseline := &confidence.Report{Score: intPtrT(80), Title: "Baseline"}
	result := CheckThresholds(report, baseline, -30, ThresholdConfig{
		FailUnder:        75,
		FailOnRegression: true,
		FactorThresholds: map[string]int{"Coverage": 50},
	})
	if result.Passed() {
		t.Error("expected all checks to fail")
	}
	if result.ScorePassed {
		t.Error("score should fail")
	}
	if result.BaselinePassed {
		t.Error("baseline should fail")
	}
	if result.FactorsPassed {
		t.Error("factors should fail")
	}
}

func TestThresholdResult_Passed(t *testing.T) {
	tests := []struct {
		name   string
		result ThresholdResult
		want   bool
	}{
		{"all pass", ThresholdResult{ScorePassed: true, BaselinePassed: true, FactorsPassed: true}, true},
		{"score fails", ThresholdResult{ScorePassed: false, BaselinePassed: true, FactorsPassed: true}, false},
		{"baseline fails", ThresholdResult{ScorePassed: true, BaselinePassed: false, FactorsPassed: true}, false},
		{"factors fail", ThresholdResult{ScorePassed: true, BaselinePassed: true, FactorsPassed: false}, false},
		{"all fail", ThresholdResult{ScorePassed: false, BaselinePassed: false, FactorsPassed: false}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Passed(); got != tt.want {
				t.Errorf("Passed() = %v, want %v", got, tt.want)
			}
		})
	}
}
