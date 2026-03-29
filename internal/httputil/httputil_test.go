package httputil

import (
	"context"
	"testing"
	"time"
)

func TestParseRetryAfter_Seconds(t *testing.T) {
	d := ParseRetryAfter("2")
	if d != 2*time.Second {
		t.Errorf("expected 2s, got %v", d)
	}
}

func TestParseRetryAfter_Zero(t *testing.T) {
	d := ParseRetryAfter("0")
	if d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestParseRetryAfter_Negative(t *testing.T) {
	d := ParseRetryAfter("-5")
	if d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestParseRetryAfter_Invalid(t *testing.T) {
	d := ParseRetryAfter("not-a-number")
	if d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}

func TestParseRetryAfter_CapsAbsurdValue(t *testing.T) {
	d := ParseRetryAfter("99999")
	if d != MaxRetryAfterDuration {
		t.Errorf("expected %v (capped), got %v", MaxRetryAfterDuration, d)
	}
}

func TestParseRetryAfter_ExactlyAtCap(t *testing.T) {
	d := ParseRetryAfter("60")
	if d != 60*time.Second {
		t.Errorf("expected 60s, got %v", d)
	}
}

func TestParseRetryAfter_JustOverCap(t *testing.T) {
	d := ParseRetryAfter("61")
	if d != MaxRetryAfterDuration {
		t.Errorf("expected %v (capped), got %v", MaxRetryAfterDuration, d)
	}
}

func TestSleepWithContext_NormalSleep(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	err := SleepWithContext(ctx, 50*time.Millisecond)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("sleep too short: %v", elapsed)
	}
}

func TestSleepWithContext_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	start := time.Now()
	err := SleepWithContext(ctx, 10*time.Second)
	elapsed := time.Since(start)

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("should have returned immediately, took %v", elapsed)
	}
}

func TestSleepWithContext_DeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := SleepWithContext(ctx, 10*time.Second)
	elapsed := time.Since(start)

	if err != context.DeadlineExceeded {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("should have returned near deadline, took %v", elapsed)
	}
}
