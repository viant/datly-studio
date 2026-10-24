package sqlstore

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
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

func TestReleaseJSONColumnsValidateValuesAcrossMySQLFormatting(t *testing.T) {
	ctx := context.Background()
	db, store, current, bundle := releaseFixture(t)
	binding := ComponentBinding{Root: bundle.Roots[0], DataSourceID: "rows", ComponentKind: "dynamic", ComponentID: "reader", ComponentRevision: "7", Method: http.MethodGet, Route: "/rows", SchemaFingerprint: strings.Repeat("b", 64), ContentFingerprint: strings.Repeat("c", 64)}
	deps := []authz.NamedDependency{
		{Resource: authz.Resource{Tenant: "tenant", Kind: "policy", ID: "operators", Version: "logical"}, Revision: 1},
		{Resource: authz.Resource{Tenant: "tenant", Kind: "policy", ID: "readers", Version: "logical"}, Revision: 2},
	}
	next, err := contract.PreparePromotion(current, 0, bundle)
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.StageRevisionWithBindingsAndDependencies(ctx, current, next, bundle, []ComponentBinding{binding}, deps)
	if err != nil {
		t.Fatal(err)
	}
	// MySQL JSON reorders object keys and inserts spaces. Neither presentation
	// change alters the immutable binding hash or the named dependency values.
	formattedDeps := `[{"revision": 1, "resource": {"version": "logical", "tenant": "tenant", "kind": "policy", "id": "operators"}}, {"revision": 2, "resource": {"version": "logical", "tenant": "tenant", "kind": "policy", "id": "readers"}}]`
	if _, err = db.ExecContext(ctx, `UPDATE artifact_release_operations SET dependencies_json=? WHERE operation_id=?`, formattedDeps, id); err != nil {
		t.Fatal(err)
	}
	gotDeps, err := store.PendingDependencies(ctx, id)
	if err != nil || !reflect.DeepEqual(gotDeps, deps) {
		t.Fatalf("reformatted dependencies changed values: %+v %v", gotDeps, err)
	}
	var raw []byte
	if err = db.QueryRowContext(ctx, `SELECT bindings_json FROM artifact_release_bundle_bindings WHERE bundle_hash=?`, bundle.Hash).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var formatted bytes.Buffer
	if err = json.Indent(&formatted, raw, "", "  "); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, `UPDATE artifact_release_bundle_bindings SET bindings_json=? WHERE bundle_hash=?`, formatted.String(), bundle.Hash); err != nil {
		t.Fatal(err)
	}
	gotBindings, err := store.BundleBindings(ctx, bundle)
	if err != nil || !reflect.DeepEqual(gotBindings, []ComponentBinding{binding}) {
		t.Fatalf("reformatted immutable bindings changed values: %+v %v", gotBindings, err)
	}
	_, canonical, hash, err := validateBindings(bundle, []ComponentBinding{binding})
	if err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.putBindingsTx(ctx, tx, bundle, canonical, hash); err != nil {
		t.Fatalf("same immutable values rejected after SQL formatting: %v", err)
	}
	_ = tx.Rollback()
	changed := binding
	changed.ComponentID = "tampered"
	changedRaw, _ := json.Marshal([]ComponentBinding{changed})
	if _, err = db.ExecContext(ctx, `UPDATE artifact_release_bundle_bindings SET bindings_json=? WHERE bundle_hash=?`, changedRaw, bundle.Hash); err != nil {
		t.Fatal(err)
	}
	if _, err = store.BundleBindings(ctx, bundle); !errors.Is(err, contract.ErrInvalid) {
		t.Fatalf("changed binding bypassed canonical-value hash: %v", err)
	}
	unknownBinding := strings.Replace(string(canonical), `"componentId":"reader"`, `"componentId":"reader","extra":true`, 1)
	if _, err = db.ExecContext(ctx, `UPDATE artifact_release_bundle_bindings SET bindings_json=? WHERE bundle_hash=?`, unknownBinding, bundle.Hash); err != nil {
		t.Fatal(err)
	}
	if _, err = store.BundleBindings(ctx, bundle); !errors.Is(err, contract.ErrInvalid) {
		t.Fatalf("unknown binding field accepted: %v", err)
	}
	unknownDep := strings.Replace(formattedDeps, `"revision": 1`, `"revision": 1, "extra": true`, 1)
	if _, err = db.ExecContext(ctx, `UPDATE artifact_release_operations SET dependencies_json=? WHERE operation_id=?`, unknownDep, id); err != nil {
		t.Fatal(err)
	}
	if _, err = store.PendingDependencies(ctx, id); !errors.Is(err, contract.ErrInvalid) {
		t.Fatalf("unknown dependency field accepted: %v", err)
	}
	reversed, _ := json.Marshal([]authz.NamedDependency{deps[1], deps[0]})
	if _, err = db.ExecContext(ctx, `UPDATE artifact_release_operations SET dependencies_json=? WHERE operation_id=?`, reversed, id); err != nil {
		t.Fatal(err)
	}
	if _, err = store.PendingDependencies(ctx, id); !errors.Is(err, contract.ErrInvalid) {
		t.Fatalf("unsorted dependencies accepted: %v", err)
	}
	duplicate, _ := json.Marshal([]authz.NamedDependency{deps[0], deps[0]})
	if _, err = db.ExecContext(ctx, `UPDATE artifact_release_operations SET dependencies_json=? WHERE operation_id=?`, duplicate, id); err != nil {
		t.Fatal(err)
	}
	if _, err = store.PendingDependencies(ctx, id); !errors.Is(err, contract.ErrInvalid) {
		t.Fatalf("duplicate dependencies accepted: %v", err)
	}
	if _, err = db.ExecContext(ctx, `UPDATE artifact_release_operations SET dependencies_json=? WHERE operation_id=?`, formattedDeps, id); err != nil {
		t.Fatal(err)
	}
	if err = store.CompleteRevision(ctx, current.Key, 0, id); err != nil {
		t.Fatal(err)
	}
	gotDeps, err = store.ActiveDependencies(ctx, current.Key)
	if err != nil || !reflect.DeepEqual(gotDeps, deps) {
		t.Fatalf("active dependencies changed values: %+v %v", gotDeps, err)
	}
}
