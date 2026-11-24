// Package warmupschedule owns recurring warmup interval semantics.
package warmupschedule

import (
	"fmt"
	"math"
	"time"

	"github.com/viant/datly-studio/sdk"
)

func Validate(seconds int64, first time.Time) error {
	if seconds <= 0 || seconds > math.MaxInt64/int64(time.Second) {
		return &sdk.Error{Code: sdk.ErrorInvalidArgument, Field: "intervalSeconds", Message: "intervalSeconds must be positive and fit a native duration"}
	}
	stored := first.UTC().Truncate(time.Microsecond)
	if stored.IsZero() || first.Year() < 1 || stored.Year() < 1 {
		const message = "firstRunAt must be a complete, nonzero timestamp with a year of at least 1"
		return &sdk.Error{Code: sdk.ErrorInvalidArgument, Field: "firstRunAt", Message: message,
			Diagnostics: []sdk.Diagnostic{{Severity: "error", Code: "invalid_argument", Field: "firstRunAt", Message: message}}}
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
