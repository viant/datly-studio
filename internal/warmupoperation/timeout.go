// Package warmupoperation owns the timeout policy shared by Studio warmup hosts.
package warmupoperation

import (
	"fmt"
	"time"
)

// DefaultTimeout is also the historical budget of rows without a deadline.
const DefaultTimeout = 5 * time.Minute

// Timeout resolves trusted host configuration. Nil means omitted; an explicit
// empty, malformed, overflowing, zero, or negative duration is invalid.
func Timeout(configured *string) (time.Duration, error) {
	if configured == nil {
		return DefaultTimeout, nil
	}
	duration, err := time.ParseDuration(*configured)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("WarmupOperationTimeout must be a positive duration")
	}
	return duration, nil
}
