package main

import (
	"context"
	"testing"
	"time"
)

type observedSessionPruner struct{ calls chan time.Time }

func (p observedSessionPruner) DeleteExpired(ctx context.Context, cutoff time.Time) error {
	select {
	case p.calls <- cutoff:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestSessionPrunerRunsAndStopsWithHostContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pruner := observedSessionPruner{calls: make(chan time.Time, 4)}
	done := startSessionPruner(ctx, pruner, 10*time.Millisecond)
	for i := 0; i < 2; i++ {
		select {
		case cutoff := <-pruner.calls:
			if cutoff.IsZero() {
				t.Fatal("cleanup received a zero cutoff")
			}
		case <-time.After(time.Second):
			t.Fatal("session cleanup did not run")
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("session cleanup did not stop")
	}
}

func TestSessionPrunerCanBeDisabled(t *testing.T) {
	pruner := observedSessionPruner{calls: make(chan time.Time, 1)}
	select {
	case <-startSessionPruner(context.Background(), pruner, 0):
	case <-time.After(time.Second):
		t.Fatal("disabled cleanup did not return")
	}
	if len(pruner.calls) != 0 {
		t.Fatal("disabled cleanup ran")
	}
}
