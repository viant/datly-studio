package nativestore_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/viant/datly-studio/internal/datatest"
)

func TestPublicAudiencePersistsThroughNativeStageRecoveryAndCompletion(t *testing.T) {
	ctx := context.Background()
	db, store, current, bundle := releaseFixture(t)
	next := current
	next.Public = true
	next.Policies = nil
	next.Revision++
	next.ActiveBundleHash = bundle.Hash
	id, err := store.StageRevision(ctx, current, next, bundle)
	if err != nil {
		t.Fatal(err)
	}
	before, err := store.Channel(ctx, current.Key)
	if err != nil || !reflect.DeepEqual(before, current) {
		t.Fatalf("public intent exposed before activation: %+v %v", before, err)
	}
	restarted, err := datatest.NativeReleaseStore(t, db, scopeFromChannel(current), "verified-actor")
	if err != nil {
		t.Fatal(err)
	}
	pending, err := restarted.Pending(ctx)
	if err != nil || len(pending) != 1 || !reflect.DeepEqual(pending[0].Next, next) {
		t.Fatalf("public recovery lost: %+v %v", pending, err)
	}
	if err = restarted.CompleteRevision(ctx, current.Key, current.Revision, id); err != nil {
		t.Fatal(err)
	}
	after, err := restarted.Channel(ctx, current.Key)
	if err != nil || !reflect.DeepEqual(after, next) {
		t.Fatalf("public activation mismatch: %+v %v", after, err)
	}
	private := after
	private.Public = false
	private.Policies = current.Policies
	private.Revision++
	id, err = restarted.StageRevision(ctx, after, private, bundle)
	if err != nil {
		t.Fatal(err)
	}
	if err = restarted.CompleteRevision(ctx, after.Key, after.Revision, id); err != nil {
		t.Fatal(err)
	}
	final, err := restarted.Channel(ctx, current.Key)
	if err != nil || !reflect.DeepEqual(final, private) {
		t.Fatalf("private restoration mismatch: %+v %v", final, err)
	}
}
