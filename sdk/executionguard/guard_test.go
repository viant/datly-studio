package executionguard

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCheckRetainsIndependentGuardsAndFailsClosed(t *testing.T) {
	denied := errors.New("revoked")
	count := 0
	parent := With(context.Background(), func(context.Context) error { count++; return nil })
	child := With(parent, func(context.Context) error { return denied })
	if err := Check(child); !errors.Is(err, denied) || count != 1 {
		t.Fatalf("child guard result: %v, checks: %d", err, count)
	}
	if err := Check(parent); err != nil || count != 2 {
		t.Fatalf("child registration changed parent: %v, checks: %d", err, count)
	}
	panicking := With(parent, func(context.Context) error { panic("private failure") })
	if err := Check(panicking); err == nil {
		t.Fatal("panicking completion guard allowed commit")
	}
	canceled, cancel := context.WithCancel(context.Background())
	canceled = With(canceled, func(context.Context) error { cancel(); return nil })
	if err := Check(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("guard cancellation allowed commit: %v", err)
	}
}

func TestCaptureRetainsChecksAndCapsDispatchDeadline(t *testing.T) {
	type privateKey struct{}
	deadline := time.Now().Add(time.Minute)
	source, cancel := context.WithDeadline(context.WithValue(context.Background(), privateKey{}, "source-only"), deadline)
	defer cancel()
	denied := errors.New("revoked after dispatch")
	source = With(source, func(context.Context) error { return denied })
	continuation, release := Capture(context.Background(), source)
	defer release()
	if actual, ok := continuation.Deadline(); !ok || !actual.Equal(deadline) {
		t.Fatalf("dispatch deadline was lost: %v, %v", actual, ok)
	}
	if continuation.Value(privateKey{}) != nil {
		t.Fatal("capture copied arbitrary source context")
	}
	if err := Check(continuation); !errors.Is(err, denied) {
		t.Fatalf("continuation lost completion check: %v", err)
	}
	release()
	if err := Check(continuation); !errors.Is(err, context.Canceled) {
		t.Fatalf("released continuation allowed completion: %v", err)
	}
}
