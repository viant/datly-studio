package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
)

func TestImmutableBindingsStageWithChannelCAS(t *testing.T) {
	ctx := context.Background()
	db, store, current, bundle := releaseFixture(t)
	binding := ComponentBinding{Root: bundle.Roots[0], DataSourceID: "rows", ComponentKind: "dynamic", ComponentID: "reader", ComponentRevision: "7", Method: http.MethodGet, Route: "/rows", SchemaFingerprint: strings.Repeat("b", 64), ContentFingerprint: strings.Repeat("c", 64)}
	next, err := contract.PreparePromotion(current, 0, bundle)
	if err != nil {
		t.Fatal(err)
	}
	deps := []authz.NamedDependency{{Resource: authz.Resource{Tenant: "tenant", Kind: "policy", ID: "operators", Version: "logical"}, Revision: 1}}
	id, err := store.StageRevisionWithBindingsAndDependencies(ctx, current, next, bundle, []ComponentBinding{binding}, deps)
	if err != nil {
		t.Fatal(err)
	}
	pendingDeps, err := store.PendingDependencies(ctx, id)
	if err != nil || !reflect.DeepEqual(pendingDeps, deps) {
		t.Fatalf("pending dependency revision missing: %+v %v", pendingDeps, err)
	}
	stored, err := store.BundleBindings(ctx, bundle)
	if err != nil || !reflect.DeepEqual(stored, []ComponentBinding{binding}) {
		t.Fatalf("binding lost: %+v %v", stored, err)
	}
	if err = store.CompleteRevision(ctx, current.Key, 0, id); err != nil {
		t.Fatal(err)
	}
	activeDeps, err := store.ActiveDependencies(ctx, current.Key)
	if err != nil || !reflect.DeepEqual(activeDeps, deps) {
		t.Fatalf("active dependency revision missing: %+v %v", activeDeps, err)
	}
	changed := binding
	changed.ComponentID = "other"
	if _, err = store.StageRevisionWithBindings(ctx, next, next, bundle, []ComponentBinding{changed}); !errors.Is(err, contract.ErrConflict) {
		t.Fatalf("immutable companion overwritten: %v", err)
	}
	stored, err = store.BundleBindings(ctx, bundle)
	if err != nil || !reflect.DeepEqual(stored, []ComponentBinding{binding}) {
		t.Fatalf("changed companion persisted: %+v %v", stored, err)
	}
	other := bundle
	other.Roots = append([]artifactcohort.RootRef(nil), bundle.Roots...)
	other.Roots[0].Version = 2
	other.Hash = contract.BundleHash(other)
	if err = store.PutBundle(ctx, other); err != nil {
		t.Fatal(err)
	}
	otherBinding := binding
	otherBinding.Root = other.Roots[0]
	if _, err = store.StageRevisionWithBindings(ctx, current, next, other, []ComponentBinding{otherBinding}); !errors.Is(err, contract.ErrInvalid) && !errors.Is(err, contract.ErrConflict) {
		t.Fatalf("stale stage accepted: %v", err)
	}
	if _, err = store.BundleBindings(ctx, other); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("failed CAS left companion row: %v", err)
	}
	var count int
	if err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM artifact_release_bundle_bindings`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("companion count=%d err=%v", count, err)
	}
}
