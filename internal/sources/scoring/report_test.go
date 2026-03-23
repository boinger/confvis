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

	report, err := BuildReport("Test Project", "testsource", 75, factors)
	if err != nil {
		t.Fatalf("BuildReport returned unexpected error: %v", err)
	}

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
	report, err := BuildReport("Empty", "testsource", 50, []confidence.Factor{})
	if err != nil {
		t.Fatalf("BuildReport returned unexpected error: %v", err)
	}

	if report.ScoreValue() != 0 {
		t.Errorf("Score = %d, want 0 for empty factors", report.Score)
	}
}

func TestBuildReport_ErrorOnInvalidReport(t *testing.T) {
	// Factor with score > 100 should trigger validation error
	factors := []confidence.Factor{
		{Name: "Bad Factor", Score: 150, Weight: 50},
	}
	report, err := BuildReport("Test", "testsource", 75, factors)
	if err == nil {
		t.Error("BuildReport should return error on invalid report")
	}
	if report != nil {
		t.Error("BuildReport should return nil report on error")
	}
}

func TestBuildReport_PerfectScore(t *testing.T) {
	factors := []confidence.Factor{
		{Name: "Factor1", Score: 100, Weight: 50},
		{Name: "Factor2", Score: 100, Weight: 50},
	}

	report, err := BuildReport("Perfect", "testsource", 75, factors)
	if err != nil {
		t.Fatalf("BuildReport returned unexpected error: %v", err)
	}

	if report.ScoreValue() != 100 {
		t.Errorf("Score = %d, want 100", report.Score)
	}
}

func TestBuildReport_ScoreCalculation(t *testing.T) {
	factors := []confidence.Factor{
		{Name: "Critical", Score: 34, Weight: 40},
		{Name: "High", Score: 60, Weight: 30},
		{Name: "Medium", Score: 80, Weight: 20},
		{Name: "Low", Score: 95, Weight: 10},
	}

	report, err := BuildReport("Score Test", "testsource", 80, factors)
	if err != nil {
		t.Fatalf("BuildReport returned unexpected error: %v", err)
	}

	// Expected weighted score: (34*40 + 60*30 + 80*20 + 95*10) = 1360 + 1800 + 1600 + 950 = 5710
	// With rounding: (5710 + 50) / 100 = 57
	expectedScore := 57
	if report.ScoreValue() != expectedScore {
		t.Errorf("Score = %d, want %d", report.ScoreValue(), expectedScore)
	}
	if report.Threshold != 80 {
		t.Errorf("Threshold = %d, want 80", report.Threshold)
	}
	if report.Source != "testsource" {
		t.Errorf("Source = %q, want %q", report.Source, "testsource")
	}
	if len(report.Factors) != 4 {
		t.Errorf("len(Factors) = %d, want 4", len(report.Factors))
	}
}
