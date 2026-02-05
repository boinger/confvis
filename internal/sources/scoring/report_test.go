package scoring

import (
	"testing"
	"time"

	"github.com/boinger/confvis/internal/confidence"
)

func TestBuildReport(t *testing.T) {
	factors := []confidence.Factor{
		{Name: "Critical", Score: 67, Weight: 40},
		{Name: "High", Score: 100, Weight: 30},
		{Name: "Medium", Score: 100, Weight: 20},
		{Name: "Low", Score: 100, Weight: 10},
	}

	report := BuildReport("Test Project", "testsource", 75, factors)

	if report.Title != "Test Project" {
		t.Errorf("Title = %q, want %q", report.Title, "Test Project")
	}
	if report.Source != "testsource" {
		t.Errorf("Source = %q, want %q", report.Source, "testsource")
	}
	if report.Threshold != 75 {
		t.Errorf("Threshold = %d, want 75", report.Threshold)
	}
	if len(report.Factors) != 4 {
		t.Errorf("len(Factors) = %d, want 4", len(report.Factors))
	}

	// Expected weighted score: (67*40 + 100*30 + 100*20 + 100*10) = 8680
	// With rounding: (8680 + 50) / 100 = 87
	expectedScore := 87
	if report.ScoreValue() != expectedScore {
		t.Errorf("Score = %d, want %d", report.Score, expectedScore)
	}

	// GeneratedAt should be recent (within last minute)
	generatedAt, err := time.Parse(time.RFC3339, report.GeneratedAt)
	if err != nil {
		t.Errorf("GeneratedAt parse error: %v", err)
	}
	if time.Since(generatedAt) > time.Minute {
		t.Error("GeneratedAt is too old")
	}
}

func TestBuildReport_EmptyFactors(t *testing.T) {
	report := BuildReport("Empty", "testsource", 50, []confidence.Factor{})

	if report.ScoreValue() != 0 {
		t.Errorf("Score = %d, want 0 for empty factors", report.Score)
	}
}

func TestBuildReport_PerfectScore(t *testing.T) {
	factors := []confidence.Factor{
		{Name: "Factor1", Score: 100, Weight: 50},
		{Name: "Factor2", Score: 100, Weight: 50},
	}

	report := BuildReport("Perfect", "testsource", 75, factors)

	if report.ScoreValue() != 100 {
		t.Errorf("Score = %d, want 100", report.Score)
	}
}
