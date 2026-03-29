// Package httputil provides shared HTTP utility functions used by API clients.
package httputil

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// MaxRetryAfterDuration is the upper bound for parsed Retry-After values.
// A server sending Retry-After: 86400 would otherwise block for 24 hours.
const MaxRetryAfterDuration = 60 * time.Second

// ParseRetryAfter parses a Retry-After header value (seconds or HTTP-date).
// Returns 0 if the value is unparseable or non-positive. The result is capped
// at MaxRetryAfterDuration to prevent unbounded sleep on malicious or
// misconfigured servers.
func ParseRetryAfter(value string) time.Duration {
	// Try parsing as seconds first.
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		d := time.Duration(seconds) * time.Second
		if d > MaxRetryAfterDuration {
			return MaxRetryAfterDuration
		}
		return d
	}

	// Try parsing as HTTP-date.
	if t, err := http.ParseTime(value); err == nil {
		if delay := time.Until(t); delay > 0 {
			if delay > MaxRetryAfterDuration {
				return MaxRetryAfterDuration
			}
			return delay
		}
	}

	return 0
}

// SleepWithContext sleeps for the given duration, returning early if the
// context is cancelled. Returns ctx.Err() on cancellation, nil otherwise.
func SleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
