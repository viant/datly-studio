// Package releasegate coordinates durable release activation with the host's
// artifact manifest. Every release-aware serving read and Publisher mutation
// must use the same Gate; it never locks a long-running business tool call.
package releasegate

import (
	"context"
	"errors"
	"sync"

	"github.com/viant/forge/backend/artifactrelease"
)

var ErrNotReady = errors.New("release projection is not ready")

// Projection operates on complete manifest snapshots. Project must atomically
// replace the Publisher's visible manifest; Restore must reinstall the prior
// complete manifest if DB completion fails. Both run under Gate's write lock.
type Projection struct {
	// Reconcile rebuilds one complete visible manifest from committed store
	// state before any pending operations are replayed. This also handles an
	// indeterminate DB commit that succeeded despite returning an error.
	Reconcile func(context.Context) error
	Snapshot  func(context.Context) (any, error)
	Project   func(context.Context, artifactrelease.PendingRevision) error
	Restore   func(context.Context, any) error
}

type Store interface {
	artifactrelease.ChannelStore
	FailRevision(context.Context, artifactrelease.Key, uint64, string, string) error
}

type Gate struct {
	mu         sync.RWMutex
	ready      bool
	store      Store
	projection Projection
}

func New(store Store, projection Projection) (*Gate, error) {
	if store == nil || projection.Reconcile == nil || projection.Snapshot == nil || projection.Project == nil || projection.Restore == nil {
		return nil, artifactrelease.ErrInvalid
	}
	return &Gate{store: store, projection: projection}, nil
}

// Read holds the gate only while taking or checking the authorized release
// snapshot. Callers release it before any business execution or network I/O.
func (g *Gate) Read(read func() error) error {
	if g == nil || read == nil {
		return artifactrelease.ErrInvalid
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	if !g.ready {
		return ErrNotReady
	}
	return read()
}

// Update serializes any other mutation of the same Publisher with releases.
// The host must route non-release publication changes through this gate too.
func (g *Gate) Update(update func() error) error {
	if g == nil || update == nil {
		return artifactrelease.ErrInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.ready {
		return ErrNotReady
	}
	return update()
}

// UpdateOrBlock serializes a fail-closed withdrawal. If the producer cannot
// remove an invalid cohort, release reads stay unavailable until Recover
// reconciles the committed policy/catalog state.
func (g *Gate) UpdateOrBlock(update func() error) error {
	if g == nil || update == nil {
		return artifactrelease.ErrInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.ready {
		return ErrNotReady
	}
	if err := update(); err != nil {
		g.ready = false
		return err
	}
	return nil
}

// Recover must run before release serving starts. A crash between Project and
// Complete leaves a pending intent; replay projects it again idempotently and
// acknowledges that exact operation ID. On failure the prior manifest is
// restored and release serving remains unready.
func (g *Gate) Recover(ctx context.Context) error {
	if g == nil || ctx == nil || ctx.Err() != nil {
		return artifactrelease.ErrInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.ready = false
	if err := g.projection.Reconcile(ctx); err != nil {
		return err
	}
	pending, err := g.store.Pending(ctx)
	if err != nil {
		return err
	}
	for _, item := range pending {
		if err = g.projectAndComplete(ctx, item); err != nil {
			return err
		}
	}
	g.ready = true
	return nil
}

// Promote stages one immutable release intent before projecting it. A failed
// projection is explicitly failed after restoring the prior manifest. A DB
// completion failure retains the pending intent for restart reconciliation.
func (g *Gate) Promote(ctx context.Context, current, next artifactrelease.Channel, bundle artifactrelease.Bundle) error {
	return g.PromoteWithPreflight(ctx, current, next, bundle, nil)
}

// PromoteWithPreflight rechecks the editor/lease under the write lock just
// before durable Stage. Recovery finishes an already accepted operation under
// producer authority without relying on the initiating user's session.
func (g *Gate) PromoteWithPreflight(ctx context.Context, current, next artifactrelease.Channel, bundle artifactrelease.Bundle, preflight func(context.Context) error) error {
	return g.PromoteWithStage(ctx, current, next, bundle, preflight, nil)
}

// PromoteWithStage lets a producer atomically persist immutable companion
// bindings and the outbox operation in its existing SQL transaction.
func (g *Gate) PromoteWithStage(ctx context.Context, current, next artifactrelease.Channel, bundle artifactrelease.Bundle, preflight func(context.Context) error, stage func(context.Context, artifactrelease.Channel, artifactrelease.Channel, artifactrelease.Bundle) (string, error)) error {
	if g == nil || ctx == nil || ctx.Err() != nil {
		return artifactrelease.ErrInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.ready {
		return ErrNotReady
	}
	if preflight != nil {
		if err := preflight(ctx); err != nil {
			return err
		}
	}
	if stage == nil {
		stage = g.store.StageRevision
	}
	operationID, err := stage(ctx, current, next, bundle)
	if err != nil {
		return err
	}
	item := artifactrelease.PendingRevision{OperationID: operationID, Current: current, Next: next, BundleHash: bundle.Hash}
	prior, err := g.projection.Snapshot(ctx)
	if err != nil {
		g.ready = false
		return err
	}
	if err = g.projection.Project(ctx, item); err != nil {
		restoreErr := g.projection.Restore(ctx, prior)
		if restoreErr != nil {
			g.ready = false
			return errors.Join(err, restoreErr)
		}
		failErr := g.store.FailRevision(ctx, current.Key, current.Revision, operationID, "projection_failed")
		if failErr != nil {
			g.ready = false
		}
		return errors.Join(err, failErr)
	}
	if err = g.store.CompleteRevision(ctx, current.Key, current.Revision, operationID); err != nil {
		restoreErr := g.projection.Restore(ctx, prior)
		g.ready = false
		return errors.Join(err, restoreErr)
	}
	return nil
}

func (g *Gate) projectAndComplete(ctx context.Context, item artifactrelease.PendingRevision) error {
	prior, err := g.projection.Snapshot(ctx)
	if err != nil {
		return err
	}
	if err = g.projection.Project(ctx, item); err == nil {
		err = g.store.CompleteRevision(ctx, item.Current.Key, item.Current.Revision, item.OperationID)
	}
	if err != nil {
		return errors.Join(err, g.projection.Restore(ctx, prior))
	}
	return nil
}
