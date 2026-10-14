// Package executionguard retains trusted embedding-server checks for adapters
// which own their transaction completion. It is not a wire authorization API.
package executionguard

import (
	"context"
	"fmt"
)

type key struct{}
type CheckFunc func(context.Context) error

// With registers a side-effect-free Go check. No request field or MCP metadata
// can install a guard; the adapter's existing completion owner invokes Check.
func With(ctx context.Context, check CheckFunc) context.Context {
	if check == nil {
		return ctx
	}
	checks, _ := ctx.Value(key{}).([]CheckFunc)
	retained := append(append([]CheckFunc(nil), checks...), check)
	return context.WithValue(ctx, key{}, retained)
}

// Check fails closed on cancellation, revocation or a panicking guard. It never
// starts, commits or rolls back a transaction.
func Check(ctx context.Context) (err error) {
	defer func() {
		if recover() != nil {
			err = fmt.Errorf("execution guard failed")
		}
	}()
	checks, _ := ctx.Value(key{}).([]CheckFunc)
	for _, check := range checks {
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = check(ctx); err != nil {
			return err
		}
	}
	return ctx.Err()
}

// Capture carries only registered checks into an explicitly owned continuation
// and caps it to the original dispatch deadline. It does not copy credentials,
// claims, component input, transaction ownership or arbitrary request metadata.
func Capture(base, source context.Context) (context.Context, context.CancelFunc) {
	checks, _ := source.Value(key{}).([]CheckFunc)
	if len(checks) == 0 {
		return base, func() {}
	}
	for _, check := range checks {
		base = With(base, check)
	}
	if deadline, ok := source.Deadline(); ok {
		return context.WithDeadline(base, deadline)
	}
	return base, func() {}
}
