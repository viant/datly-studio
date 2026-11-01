// Package change exposes an optional host notification for committed namespace writes.
package change

import (
	"context"
	"errors"
	"github.com/viant/datly-studio/studio/settings"
	"reflect"
)

type Identity struct {
	NamespaceID     string
	OwnerID         string
	Name            string
	SettingsChanged bool
	Revision        int64
	SecurityChanged bool
	Security        *settings.Security
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

// WithAdditionalObserver retains an embedding host's notification while adding
// another trusted post-commit consumer. Both run even if one reports failure.
func WithAdditionalObserver(ctx context.Context, observer Observer) context.Context {
	previous, _ := ctx.Value(observerKey{}).(Observer)
	if previous == nil {
		return WithObserver(ctx, observer)
	}
	if observer == nil {
		return ctx
	}
	if containsObserver(previous, observer) {
		return ctx
	}
	return WithObserver(ctx, &observerChain{observer, previous})
}

type observerChain struct{ first, next Observer }

func (c *observerChain) Committed(ctx context.Context, identities []Identity) error {
	return errors.Join(c.first.Committed(ctx, identities), c.next.Committed(ctx, identities))
}
func containsObserver(current, candidate Observer) bool {
	if chain, ok := current.(*observerChain); ok {
		return containsObserver(chain.first, candidate) || containsObserver(chain.next, candidate)
	}
	typeOf := reflect.TypeOf(candidate)
	return typeOf != nil && typeOf.Comparable() && reflect.TypeOf(current) == typeOf && current == candidate
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
