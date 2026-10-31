package nativestore_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/sdk/artifactrelease/nativestore"
	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/forge/backend/artifactcohort"
	"github.com/viant/forge/backend/artifactrelease"
	_ "modernc.org/sqlite"
)

func releaseFixture(t *testing.T) (*sql.DB, *nativestore.Store, artifactrelease.Channel, artifactrelease.Bundle) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	scope := releasevalue.Scope{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "demo"}
	store, err := datatest.NativeReleaseStore(t, db, scope, "verified-actor")
	if err != nil {
		t.Fatal(err)
	}
	key := artifactrelease.Key{Tenant: scope.Tenant, Account: scope.Account, Workspace: scope.Workspace, Namespace: scope.Namespace, Name: "main"}
	policy := authz.NamedReference{Resource: authz.Resource{Tenant: scope.Tenant, Kind: "policy", ID: "readers", Version: "logical"}, Revision: 1}
	channel, err := artifactrelease.NewChannel(key, "Release", []authz.NamedReference{policy})
	if err != nil {
		t.Fatal(err)
	}
	ref := artifactcohort.RootRef{URI: "window://demo/home", Kind: "window", Version: 1, Hash: strings.Repeat("a", 64)}
	bundle := artifactrelease.Bundle{Namespace: "demo", Roots: []artifactcohort.RootRef{ref}}
	bundle.Hash = artifactrelease.BundleHash(bundle)
	if err = store.PutBundle(ctx, bundle); err != nil {
		t.Fatal(err)
	}
	if err = store.CreateChannel(ctx, channel); err != nil {
		t.Fatal(err)
	}
	return db, store, channel, bundle
}

// A new channel has no active bundle. Its nullable composite FK must not
// require an unrelated bundle row merely to make the scope scalar exist.
func TestReleaseCreateUnpublishedChannelWithoutBundle(t *testing.T) {
	ctx := t.Context()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	scope := releasevalue.Scope{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "empty"}
	store, err := datatest.NativeReleaseStore(t, db, scope, "actor")
	if err != nil {
		t.Fatal(err)
	}
	channel, err := artifactrelease.NewChannel(artifactrelease.Key{Tenant: scope.Tenant, Account: scope.Account, Workspace: scope.Workspace, Namespace: scope.Namespace, Name: "main"}, "Unpublished", []authz.NamedReference{{Resource: authz.Resource{Tenant: scope.Tenant, Kind: "policy", ID: "readers", Version: "logical"}, Revision: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if err = store.CreateChannel(ctx, channel); err != nil {
		t.Fatal(err)
	}
	selected, err := store.ChannelSelections(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 1 || selected[0].Channel.Key != channel.Key || selected[0].Bundle.Hash != "" {
		t.Fatalf("unpublished selection: %+v", selected)
	}
	active, err := store.ActiveSelections(ctx)
	if err != nil || len(active) != 0 {
		t.Fatalf("unpublished channel became active: %+v %v", active, err)
	}
	var count int
	if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM artifact_release_bundles").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("unpublished channel created %d bundle rows", count)
	}
}

func TestReleaseStageRecoveryAndIdempotentComplete(t *testing.T) {
	ctx := context.Background()
	db, store, current, bundle := releaseFixture(t)
	next, err := artifactrelease.PreparePromotion(current, 0, bundle)
	if err != nil {
		t.Fatal(err)
	}
	operationID, err := store.StageRevision(ctx, current, next, bundle)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Channel(ctx, current.Key)
	if err != nil || !reflect.DeepEqual(got, current) {
		t.Fatalf("pending exposed active channel: %+v %v", got, err)
	}
	restarted, err := datatest.NativeReleaseStore(t, db, scopeFromChannel(current), "verified-actor")
	if err != nil {
		t.Fatal(err)
	}
	pending, err := restarted.Pending(ctx)
	if err != nil || len(pending) != 1 || !reflect.DeepEqual(pending[0].Next, next) || pending[0].BundleHash != bundle.Hash {
		t.Fatalf("lost replay intent: %+v %v", pending, err)
	}
	if pending[0].OperationID != operationID {
		t.Fatal("recovered operation identity changed")
	}
	if err = restarted.CompleteRevision(ctx, current.Key, 0, operationID); err != nil {
		t.Fatal(err)
	}
	if err = restarted.CompleteRevision(ctx, current.Key, 0, operationID); err != nil {
		t.Fatalf("completion not idempotent: %v", err)
	}
	got, err = restarted.Channel(ctx, current.Key)
	if err != nil || !reflect.DeepEqual(got, next) {
		t.Fatalf("activation mismatch: %+v %v", got, err)
	}
	pending, err = restarted.Pending(ctx)
	if err != nil || len(pending) != 0 {
		t.Fatalf("completed operation remained pending: %+v %v", pending, err)
	}
}

func TestReleaseFailedProjectionRetainsActiveAndAllowsRetry(t *testing.T) {
	ctx := context.Background()
	_, store, current, bundle := releaseFixture(t)
	next, err := artifactrelease.PreparePromotion(current, 0, bundle)
	if err != nil {
		t.Fatal(err)
	}
	operationID, err := store.StageRevision(ctx, current, next, bundle)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.FailRevision(ctx, current.Key, 0, operationID, "projection_failed"); err != nil {
		t.Fatal(err)
	}
	got, err := store.Channel(ctx, current.Key)
	if err != nil || !reflect.DeepEqual(got, current) {
		t.Fatalf("failure changed active channel: %+v %v", got, err)
	}
	pending, err := store.Pending(ctx)
	if err != nil || len(pending) != 0 {
		t.Fatalf("failure left pending: %+v %v", pending, err)
	}
	if _, err = store.StageRevision(ctx, current, next, bundle); err != nil {
		t.Fatalf("retry after failure: %v", err)
	}
}

func TestReleaseCASAndNativeRollback(t *testing.T) {
	ctx := context.Background()
	db, store, current, bundle := releaseFixture(t)
	next, err := artifactrelease.PreparePromotion(current, 0, bundle)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, `CREATE TRIGGER reject_native_operation BEFORE INSERT ON artifact_release_operations BEGIN SELECT RAISE(ABORT,'synthetic operation failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err = store.StageRevision(ctx, current, next, bundle); err == nil {
		t.Fatal("native child failure must roll back the channel reservation")
	}
	if _, err = db.ExecContext(ctx, `DROP TRIGGER reject_native_operation`); err != nil {
		t.Fatal(err)
	}
	unchanged, err := store.Channel(ctx, current.Key)
	if err != nil || !reflect.DeepEqual(unchanged, current) {
		t.Fatalf("native rollback changed channel: %+v %v", unchanged, err)
	}
	pending, err := store.Pending(ctx)
	if err != nil || len(pending) != 0 {
		t.Fatalf("rolled-back intent visible: %+v %v", pending, err)
	}
	operationID, err := store.StageRevision(ctx, current, next, bundle)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.StageRevision(ctx, current, next, bundle); !errors.Is(err, artifactrelease.ErrConflict) {
		t.Fatalf("stale/pending CAS=%v", err)
	}
	if err = store.CompleteRevision(ctx, current.Key, 0, operationID); err != nil {
		t.Fatal(err)
	}
	if _, err = store.StageRevision(ctx, current, next, bundle); !errors.Is(err, artifactrelease.ErrConflict) {
		t.Fatalf("stale completed CAS=%v", err)
	}
}

func TestReleaseConcurrentStageHasOneWinner(t *testing.T) {
	ctx := context.Background()
	_, store, current, bundle := releaseFixture(t)
	next, err := artifactrelease.PreparePromotion(current, 0, bundle)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for i := 0; i < 2; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			_, err := store.StageRevision(ctx, current, next, bundle)
			results <- err
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	success, conflict := 0, 0
	for result := range results {
		switch {
		case result == nil:
			success++
		case errors.Is(result, artifactrelease.ErrConflict):
			conflict++
		default:
			t.Fatalf("unexpected concurrent result: %v", result)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success=%d conflict=%d", success, conflict)
	}
	pending, err := store.Pending(ctx)
	if err != nil || len(pending) != 1 {
		t.Fatalf("one winner must leave one intent: %+v %v", pending, err)
	}
}

func TestReleasePolicyAndDescriptionChangeUsesSameProjection(t *testing.T) {
	ctx := context.Background()
	_, store, current, bundle := releaseFixture(t)
	policy := authz.NamedReference{Resource: authz.Resource{Tenant: "tenant", Kind: "policy", ID: "operators", Version: "logical"}, Revision: 2}
	next, err := artifactrelease.PrepareChannelUpdate(current, 0, "Updated release", []authz.NamedReference{policy}, bundle)
	if err != nil {
		t.Fatal(err)
	}
	operationID, err := store.StageRevision(ctx, current, next, bundle)
	if err != nil {
		t.Fatal(err)
	}
	before, err := store.Channel(ctx, current.Key)
	if err != nil || !reflect.DeepEqual(before, current) {
		t.Fatalf("policy edit leaked before projection: %+v %v", before, err)
	}
	if err = store.CompleteRevision(ctx, current.Key, 0, operationID); err != nil {
		t.Fatal(err)
	}
	after, err := store.Channel(ctx, current.Key)
	if err != nil || !reflect.DeepEqual(after, next) {
		t.Fatalf("policy edit activation mismatch: %+v %v", after, err)
	}
}

func TestReleaseStaleWorkerCannotCompleteOrFailReplacementIntent(t *testing.T) {
	ctx := context.Background()
	_, store, current, first := releaseFixture(t)
	firstNext, err := artifactrelease.PreparePromotion(current, 0, first)
	if err != nil {
		t.Fatal(err)
	}
	firstID, err := store.StageRevision(ctx, current, firstNext, first)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.FailRevision(ctx, current.Key, 0, firstID, "projection_failed"); err != nil {
		t.Fatal(err)
	}
	second := first
	second.Roots = append([]artifactcohort.RootRef(nil), first.Roots...)
	second.Roots[0].Version = 2
	second.Hash = artifactrelease.BundleHash(second)
	if err = store.PutBundle(ctx, second); err != nil {
		t.Fatal(err)
	}
	secondNext, err := artifactrelease.PreparePromotion(current, 0, second)
	if err != nil {
		t.Fatal(err)
	}
	secondID, err := store.StageRevision(ctx, current, secondNext, second)
	if err != nil {
		t.Fatal(err)
	}
	if firstID == secondID {
		t.Fatal("replacement intent reused operation ID")
	}
	if err = store.CompleteRevision(ctx, current.Key, 0, firstID); !errors.Is(err, artifactrelease.ErrConflict) {
		t.Fatalf("stale worker completed new intent: %v", err)
	}
	if err = store.FailRevision(ctx, current.Key, 0, firstID, "projection_failed"); !errors.Is(err, artifactrelease.ErrConflict) {
		t.Fatalf("stale worker failed new intent: %v", err)
	}
	before, err := store.Channel(ctx, current.Key)
	if err != nil || !reflect.DeepEqual(before, current) {
		t.Fatalf("stale worker changed active: %+v %v", before, err)
	}
	if err = store.CompleteRevision(ctx, current.Key, 0, secondID); err != nil {
		t.Fatal(err)
	}
	after, err := store.Channel(ctx, current.Key)
	if err != nil || !reflect.DeepEqual(after, secondNext) {
		t.Fatalf("replacement did not activate: %+v %v", after, err)
	}
}

func TestReleaseImmutableBundleAndScope(t *testing.T) {
	ctx := context.Background()
	db, store, _, bundle := releaseFixture(t)
	if err := store.PutBundle(ctx, bundle); err != nil {
		t.Fatalf("same immutable bundle should be idempotent: %v", err)
	}
	mutated := bundle
	mutated.Roots = append([]artifactcohort.RootRef(nil), bundle.Roots...)
	mutated.Roots[0].Version = 2
	if err := store.PutBundle(ctx, mutated); !errors.Is(err, artifactrelease.ErrInvalid) {
		t.Fatalf("hash/content mismatch accepted: %v", err)
	}
	other, err := datatest.NativeReleaseStore(t, db, releasevalue.Scope{Tenant: "tenant", Account: "other", Workspace: "workspace", Namespace: "demo"}, "verified-actor")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = other.Bundle(ctx, bundle.Hash); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-account bundle visible: %v", err)
	}
	if _, err = other.Channel(ctx, artifactrelease.Key{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "demo", Name: "main"}); !errors.Is(err, artifactrelease.ErrInvalid) {
		t.Fatalf("cross-account key accepted: %v", err)
	}
}

type actorKey struct{}

func TestReleaseScopedListHistoryAndRequestActor(t *testing.T) {
	ctx := context.Background()
	db, _, current, bundle := releaseFixture(t)
	store, err := datatest.NativeReleaseStoreWithActor(t, db, scopeFromChannel(current), func(ctx context.Context) (string, error) {
		actor, _ := ctx.Value(actorKey{}).(string)
		if actor == "" {
			return "", errors.New("verified actor missing")
		}
		return actor, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	next, err := artifactrelease.PreparePromotion(current, 0, bundle)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.StageRevision(ctx, current, next, bundle); err == nil {
		t.Fatal("stage accepted missing verified actor")
	}
	pending, err := store.Pending(ctx)
	if err != nil || len(pending) != 0 {
		t.Fatalf("failed actor resolver changed pending: %+v %v", pending, err)
	}
	alice := context.WithValue(ctx, actorKey{}, "alice")
	id, err := store.StageRevision(alice, current, next, bundle)
	if err != nil {
		t.Fatal(err)
	}
	channels, err := store.ListChannels(ctx)
	if err != nil || len(channels) != 1 || !reflect.DeepEqual(channels[0], current) {
		t.Fatalf("pending exposed via list: %+v %v", channels, err)
	}
	bob := context.WithValue(ctx, actorKey{}, "bob")
	if err = store.CompleteRevision(bob, current.Key, 0, id); err != nil {
		t.Fatal(err)
	}
	history, err := store.History(ctx, current.Key)
	if err != nil || len(history) != 1 || history[0].OperationID != id || history[0].Status != "applied" || history[0].CreatedBy != "alice" || history[0].CompletedAt == nil || !reflect.DeepEqual(history[0].Next, next) {
		t.Fatalf("inaccurate release history: %+v %v", history, err)
	}
	channels, err = store.ListChannels(ctx)
	if err != nil || len(channels) != 1 || !reflect.DeepEqual(channels[0], next) {
		t.Fatalf("list did not advance after commit: %+v %v", channels, err)
	}
	other, err := datatest.NativeReleaseStore(t, db, releasevalue.Scope{Tenant: "tenant", Account: "other", Workspace: "workspace", Namespace: "demo"}, "verified-actor")
	if err != nil {
		t.Fatal(err)
	}
	otherChannels, err := other.ListChannels(ctx)
	if err != nil || len(otherChannels) != 0 {
		t.Fatalf("cross-account list leaked: %+v %v", otherChannels, err)
	}
}

func scopeFromChannel(channel artifactrelease.Channel) releasevalue.Scope {
	key := channel.Key
	return releasevalue.Scope{Tenant: key.Tenant, Account: key.Account, Workspace: key.Workspace, Namespace: key.Namespace}
}
