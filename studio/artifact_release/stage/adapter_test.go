package stage_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	native "github.com/viant/datly-studio/sdk/artifactrelease/nativestore"
	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
	contract "github.com/viant/forge/backend/artifactrelease"
	"testing"
)

func TestNativeStoreStageCompleteReplay(t *testing.T) {
	ctx := context.Background()
	db, server, input := newStageCase(ctx, t, "")
	scope := releasevalue.Scope{Tenant: input.ScopeTenant, Account: input.ScopeAccount, Workspace: input.ScopeWorkspace, Namespace: input.ScopeNamespace}
	store, err := native.New(server, scope, func(context.Context) (string, error) { return "verified-actor", nil })
	if err != nil {
		t.Fatal(err)
	}
	var next contract.Channel
	if err := json.Unmarshal(*input.Stage[0].Operation[0].NextChannelJson, &next); err != nil {
		t.Fatal(err)
	}
	current, err := store.Channel(ctx, next.Key)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := store.Bundle(ctx, input.TargetBundleHash)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutBundle(ctx, bundle); err != nil {
		t.Fatalf("bundle replay: %v", err)
	}
	fresh := current
	fresh.Key.Name = "second-channel"
	if err := store.CreateChannel(ctx, fresh); err != nil {
		t.Fatalf("native create: %v", err)
	}
	if err := store.CreateChannel(ctx, fresh); err != nil {
		t.Fatalf("native create replay: %v", err)
	}
	channels, err := store.ListChannels(ctx)
	if err != nil || len(channels) != 2 {
		t.Fatalf("channels=%d err=%v", len(channels), err)
	}
	id, err := store.StageRevisionWithBindingsAndDependencies(ctx, current, next, bundle, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	pending, err := store.Pending(ctx)
	if err != nil || len(pending) != 1 || pending[0].OperationID != id {
		t.Fatalf("pending=%+v err=%v", pending, err)
	}
	if _, err := store.StageRevision(ctx, current, next, bundle); err == nil {
		t.Fatal("duplicate pending stage accepted")
	}
	if err := store.CompleteRevision(ctx, next.Key, 0, id); err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteRevision(ctx, next.Key, 0, id); err != nil {
		t.Fatalf("applied replay: %v", err)
	}
	got, err := store.Channel(ctx, next.Key)
	if err != nil || got.Revision != 1 || got.ActiveBundleHash != bundle.Hash {
		t.Fatalf("channel=%+v err=%v", got, err)
	}
	if _, err = store.PendingDependencies(ctx, id); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("completed intent pending deps: %v", err)
	}
	history, err := store.History(ctx, next.Key)
	if err != nil || len(history) != 1 || history[0].Status != "applied" {
		t.Fatalf("history=%+v err=%v", history, err)
	}
	var count int
	if err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM artifact_release_operations`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("operations=%d err=%v", count, err)
	}
}

func TestNativeStoreFailureRecoveryAndScope(t *testing.T) {
	ctx := context.Background()
	db, server, input := newStageCase(ctx, t, "")
	scope := releasevalue.Scope{Tenant: input.ScopeTenant, Account: input.ScopeAccount, Workspace: input.ScopeWorkspace, Namespace: input.ScopeNamespace}
	store, err := native.New(server, scope, func(context.Context) (string, error) { return "verified-actor", nil })
	if err != nil {
		t.Fatal(err)
	}
	var next contract.Channel
	if err = json.Unmarshal(*input.Stage[0].Operation[0].NextChannelJson, &next); err != nil {
		t.Fatal(err)
	}
	current, err := store.Channel(ctx, next.Key)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := store.Bundle(ctx, input.TargetBundleHash)
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.StageRevision(ctx, current, next, bundle)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.FailRevision(ctx, next.Key, 1, id, "projection_failed"); err == nil {
		t.Fatal("stale failure accepted")
	}
	foreign := next.Key
	foreign.Account = "foreign"
	if err = store.CompleteRevision(ctx, foreign, 0, id); !errors.Is(err, contract.ErrInvalid) {
		t.Fatalf("foreign completion=%v", err)
	}
	if err = store.FailRevision(ctx, next.Key, 0, id, "projection_failed"); err != nil {
		t.Fatal(err)
	}
	if err = store.CompleteRevision(ctx, next.Key, 0, id); err == nil {
		t.Fatal("failed intent completed")
	}
	got, err := store.Channel(ctx, next.Key)
	if err != nil || got.Revision != 0 || got.ActiveBundleHash != "" {
		t.Fatalf("failure promoted channel %+v %v", got, err)
	}
	pending, err := store.Pending(ctx)
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending=%+v err=%v", pending, err)
	}
	// Immutable existing binding reuse must issue neither update nor replacement insert.
	if _, err = db.ExecContext(ctx, `CREATE TRIGGER preserve_binding BEFORE UPDATE ON artifact_release_bundle_bindings BEGIN SELECT RAISE(ABORT,'binding changed');END;`); err != nil {
		t.Fatal(err)
	}
	if _, err = store.StageRevision(ctx, current, next, bundle); err != nil {
		t.Fatalf("reuse after failed attempt: %v", err)
	}
	history, err := store.History(ctx, next.Key)
	if err != nil || len(history) != 2 {
		t.Fatalf("history=%+v err=%v", history, err)
	}
}
