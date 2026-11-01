// Package change exposes an optional host notification for committed namespace writes.
package change

import "context"

type Identity struct {
	NamespaceID string
	OwnerID     string
	Name        string
}

type Observer interface {
	Committed(context.Context, []Identity) error
}

type ObserverFunc func(context.Context, []Identity) error

func (f ObserverFunc) Committed(ctx context.Context, identities []Identity) error {
	return f(ctx, identities)
}

type observerKey struct{}

// WithObserver supplies a trusted invocation capability; request fields cannot set it.
func WithObserver(ctx context.Context, observer Observer) context.Context {
	if observer == nil {
		return ctx
	}
	return context.WithValue(ctx, observerKey{}, observer)
}

// Notify is called by native outcome hooks only after confirmed transaction commit.
// An observer failure does not undo the committed namespace mutation.
func Notify(ctx context.Context, identities []Identity) error {
	observer, _ := ctx.Value(observerKey{}).(Observer)
	if observer == nil || len(identities) == 0 {
		return nil
	}
	return observer.Committed(ctx, identities)
}
