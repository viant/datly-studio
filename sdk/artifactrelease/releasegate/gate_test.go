package releasegate_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk/artifactrelease/releasegate"
	"github.com/viant/datly-studio/sdk/artifactrelease/sqlstore"
	"github.com/viant/forge/backend/artifactcohort"
	"github.com/viant/forge/backend/artifactrelease"
	_ "modernc.org/sqlite"
)

func fixture(t *testing.T, projection releasegate.Projection) (*sqlstore.Store, *releasegate.Gate, artifactrelease.Channel, artifactrelease.Bundle) {
	t.Helper()
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	scope := sqlstore.Scope{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "demo"}
	store, err := sqlstore.New(db, scope, "verified-actor")
	if err != nil {
		t.Fatal(err)
	}
	key := artifactrelease.Key{Tenant: scope.Tenant, Account: scope.Account, Workspace: scope.Workspace, Namespace: scope.Namespace, Name: "main"}
	policy := authz.NamedReference{Resource: authz.Resource{Tenant: "tenant", Kind: "policy", ID: "readers", Version: "logical"}, Revision: 1}
	channel, err := artifactrelease.NewChannel(key, "", []authz.NamedReference{policy})
	if err != nil {
		t.Fatal(err)
	}
	bundle := artifactrelease.Bundle{Namespace: "demo", Roots: []artifactcohort.RootRef{{URI: "window://demo/home", Kind: "window", Version: 1, Hash: strings.Repeat("a", 64)}}}
	bundle.Hash = artifactrelease.BundleHash(bundle)
	if err = store.PutBundle(ctx, bundle); err != nil {
		t.Fatal(err)
	}
	if err = store.CreateChannel(ctx, channel); err != nil {
		t.Fatal(err)
	}
	gate, err := releasegate.New(store, projection)
	if err != nil {
		t.Fatal(err)
	}
	return store, gate, channel, bundle
}

func TestProjectionGateBlocksReadersUntilDBActivation(t *testing.T) {
	ctx := context.Background()
	state := "old"
	entered := make(chan struct{})
	release := make(chan struct{})
	projection := releasegate.Projection{
		Reconcile: func(context.Context) error { return nil },
		Snapshot:  func(context.Context) (any, error) { return state, nil },
		Project: func(context.Context, artifactrelease.PendingRevision) error {
			state = "new"
			close(entered)
			<-release
			return nil
		},
		Restore: func(_ context.Context, prior any) error { state = prior.(string); return nil },
	}
	store, gate, current, bundle := fixture(t, projection)
	if err := gate.Read(func() error { return nil }); !errors.Is(err, releasegate.ErrNotReady) {
		t.Fatalf("served before recovery: %v", err)
	}
	if err := gate.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	next, err := artifactrelease.PreparePromotion(current, 0, bundle)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- gate.Promote(ctx, current, next, bundle) }()
	<-entered
	readDone := make(chan error, 1)
	go func() {
		readDone <- gate.Read(func() error {
			if state != "new" {
				return errors.New("wrong state")
			}
			return nil
		})
	}()
	select {
	case err = <-readDone:
		t.Fatalf("reader saw uncommitted projection: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	committed, err := store.Channel(ctx, current.Key)
	if err != nil || committed.Revision != 0 {
		t.Fatalf("DB activated before projection completed: %+v %v", committed, err)
	}
	close(release)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	if err = <-readDone; err != nil {
		t.Fatal(err)
	}
	committed, err = store.Channel(ctx, current.Key)
	if err != nil || committed.Revision != 1 {
		t.Fatalf("DB did not activate: %+v %v", committed, err)
	}
}

func TestProjectionGateReplaysCrashPendingBeforeServing(t *testing.T) {
	ctx := context.Background()
	state := "old"
	projection := releasegate.Projection{
		Reconcile: func(context.Context) error { return nil },
		Snapshot:  func(context.Context) (any, error) { return state, nil },
		Project: func(_ context.Context, item artifactrelease.PendingRevision) error {
			state = item.BundleHash
			return nil
		},
		Restore: func(_ context.Context, prior any) error { state = prior.(string); return nil },
	}
	store, gate, current, bundle := fixture(t, projection)
	next, err := artifactrelease.PreparePromotion(current, 0, bundle)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.StageRevision(ctx, current, next, bundle); err != nil {
		t.Fatal(err)
	}
	state = bundle.Hash // process crashed after Publisher projection, before DB completion
	if err = gate.Read(func() error { return nil }); !errors.Is(err, releasegate.ErrNotReady) {
		t.Fatalf("served pending projection: %v", err)
	}
	if err = gate.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	if err = gate.Read(func() error {
		if state != bundle.Hash {
			return errors.New("wrong bundle")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	committed, err := store.Channel(ctx, current.Key)
	if err != nil || committed.Revision != 1 {
		t.Fatalf("replay did not commit: %+v %v", committed, err)
	}
}

func TestProjectionFailureRestoresOldManifestAndChannel(t *testing.T) {
	ctx := context.Background()
	state := "old"
	projection := releasegate.Projection{
		Reconcile: func(context.Context) error { return nil },
		Snapshot:  func(context.Context) (any, error) { return state, nil },
		Project: func(context.Context, artifactrelease.PendingRevision) error {
			state = "partial"
			return errors.New("projection failed")
		},
		Restore: func(_ context.Context, prior any) error { state = prior.(string); return nil },
	}
	store, gate, current, bundle := fixture(t, projection)
	if err := gate.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	next, err := artifactrelease.PreparePromotion(current, 0, bundle)
	if err != nil {
		t.Fatal(err)
	}
	if err = gate.Promote(ctx, current, next, bundle); err == nil {
		t.Fatal("projection failure accepted")
	}
	if err = gate.Read(func() error {
		if state != "old" {
			return errors.New("partial projection visible")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	committed, err := store.Channel(ctx, current.Key)
	if err != nil || committed.Revision != 0 {
		t.Fatalf("failed projection activated channel: %+v %v", committed, err)
	}
	pending, err := store.Pending(ctx)
	if err != nil || len(pending) != 0 {
		t.Fatalf("failed projection left intent: %+v %v", pending, err)
	}
}
