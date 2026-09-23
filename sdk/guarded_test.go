package sdk

import (
	"context"
	"errors"
	"testing"
)

type testGuard func(context.Context, string, any) error

func (g testGuard) CheckOperation(c context.Context, o string, i any) error { return g(c, o, i) }

type testTransport func(context.Context, string, any, any) error

func (f testTransport) Invoke(c context.Context, o string, i, o2 any) error { return f(c, o, i, o2) }

func TestGuardPrecedesMutation(t *testing.T) {
	called := false
	next := testTransport(func(context.Context, string, any, any) error { called = true; return nil })
	denied := errors.New("denied")
	for _, g := range []OperationGuard{nil, testGuard(func(context.Context, string, any) error { return denied })} {
		transport := GuardedTransport{Next: next, Guard: g}
		if err := transport.Invoke(context.Background(), "resources.write", nil, nil); err == nil || called {
			t.Fatal("denied operation reached transport")
		}
	}
	transport := GuardedTransport{Next: next, Guard: testGuard(func(context.Context, string, any) error { return nil })}
	if err := transport.Invoke(context.Background(), "resources.write", nil, nil); err != nil || !called {
		t.Fatal("authorized operation failed")
	}
}
