// Package warmupschedule owns recurring warmup interval semantics.
package warmupschedule

import (
	"fmt"
	"math"
	"time"
)

func Validate(seconds int64, first time.Time) error {
	if seconds <= 0 || seconds > math.MaxInt64/int64(time.Second) {
		return fmt.Errorf("intervalSeconds must be positive and fit a native duration")
	}
	if first.IsZero() {
		return fmt.Errorf("firstRunAt is required")
	}
	return nil
}

// NextDue coalesces missed intervals while retaining the original cadence.
func NextDue(due, now time.Time, seconds int64) (time.Time, error) {
	if e := Validate(seconds, due); e != nil {
		return time.Time{}, e
	}
	if due.After(now) {
		return due.UTC(), nil
	}
	interval := time.Duration(seconds) * time.Second
	// Work backwards from now's remainder to avoid overflowing n*interval.
	elapsed := now.Sub(due)
	if elapsed == time.Duration(math.MaxInt64) {
		return time.Time{}, fmt.Errorf("schedule occurrence is outside the supported duration range")
	}
	return now.Add(interval - elapsed%interval).UTC().Truncate(time.Microsecond), nil
}
