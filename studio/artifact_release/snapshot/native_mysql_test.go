package snapshot_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk/artifactrelease/nativestore"
	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
	"github.com/viant/datly-studio/studio/artifact_release/snapshot"
	"github.com/viant/datly/bootstrap/connector"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/standalone"
	"github.com/viant/datly/standalone/config"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
	"github.com/viant/x"
)

// STUDIO_RELEASE_SNAPSHOT_MYSQL_DSN must name an empty database in an owned
// disposable MySQL instance. Fixtures intentionally share the bundle hash across
// scopes to verify every child relation retains its composite scope identity.
func TestNativeSnapshotGraphMySQL(t *testing.T) {
	dsn := os.Getenv("STUDIO_RELEASE_SNAPSHOT_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set STUDIO_RELEASE_SNAPSHOT_MYSQL_DSN to a disposable MySQL database")
	}
	ctx := context.Background()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	var tables int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()`).Scan(&tables))
	require.Zero(t, tables, "snapshot regression requires an empty disposable schema")
	if err = schema.Initialize(ctx, db, "mysql"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	for _, scope := range []string{"selected", "other"} {
		_, err = db.ExecContext(ctx, `INSERT INTO artifact_release_bundles(scope_id,tenant_id,account_id,workspace_id,namespace,bundle_hash,bundle_json,created_at,created_by,updated_at,updated_by) VALUES(?,'tenant','account','workspace','namespace',?,'{"roots":[]}',?,'actor',?,'actor')`, scope, strings.Repeat("a", 64), now, now)
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.ExecContext(ctx, `INSERT INTO artifact_release_bundle_bindings(scope_id,bundle_hash,bindings_json,bindings_hash,created_at,created_by,updated_at,updated_by) VALUES(?,?,?, ?,?,'actor',?,'actor')`, scope, strings.Repeat("a", 64), `[{"scope":"`+scope+`"}]`, strings.Repeat("b", 64), now, now)
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.ExecContext(ctx, `INSERT INTO artifact_release_channels(scope_id,tenant_id,account_id,workspace_id,namespace,name,description,policy_refs_json,active_bundle_hash,revision,created_at,created_by,updated_at,updated_by) VALUES(?,'tenant','account','workspace','namespace','qa','description','[]',?,1,?,'actor',?,'actor')`, scope, strings.Repeat("a", 64), now, now)
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.ExecContext(ctx, `INSERT INTO artifact_release_operations(operation_id,scope_id,channel_name,expected_revision,target_revision,target_bundle_hash,next_channel_json,status,created_at,created_by,updated_at,updated_by) VALUES(?,?,'qa',1,2,?,'{}','pending',?,'actor',?,'actor')`, scope+"-operation", scope, strings.Repeat("a", 64), now, now)
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.ExecContext(ctx, `UPDATE artifact_release_channels SET pending_operation_id=? WHERE scope_id=? AND name='qa'`, scope+"-operation", scope)
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = db.ExecContext(ctx, `INSERT INTO artifact_release_channels(scope_id,tenant_id,account_id,workspace_id,namespace,name,description,policy_refs_json,active_bundle_hash,revision,created_at,created_by,updated_at,updated_by) VALUES('selected','tenant','account','workspace','namespace','qa-two','Second','[]',?,1,?,'actor',?,'actor'),('selected','tenant','account','workspace','namespace','inactive','Inactive','[]',NULL,0,?,'actor',?,'actor')`, strings.Repeat("a", 64), now, now, now, now)
	require.NoError(t, err)
	registry := x.NewRegistry()
	for _, typ := range []reflect.Type{reflect.TypeFor[snapshot.Input](), reflect.TypeFor[snapshot.InputHas](), reflect.TypeFor[snapshot.Output](), reflect.TypeFor[snapshot.ReleaseChannel](), reflect.TypeFor[snapshot.StoredBundle](), reflect.TypeFor[snapshot.BundleBinding](), reflect.TypeFor[snapshot.ReleaseOperation](), reflect.TypeFor[snapshot.SnapshotComponent]()} {
		registry.Register(x.NewType(typ))
	}
	pkg := reflect.TypeFor[snapshot.SnapshotComponent]().PkgPath()
	server, err := standalone.New(ctx, standalone.Options{Config: &config.Config{Connector: "studio", GoBootstrap: &config.Packages{Packages: []string{pkg}, LinkedOnly: true}, Connectors: []connector.Config{{Name: "studio", Driver: "mysql", DSN: dsn}}}, Registry: registry, Holders: []any{snapshot.SnapshotComponent{}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Shutdown(ctx) })
	if err = server.Reload(ctx, 1); err != nil {
		t.Fatal(err)
	}
	selectCount := func() int64 {
		var key string
		var count int64
		// The runtime opens its own connector. This instance is exclusively
		// owned by the test, so global counters include those native reads.
		require.NoError(t, db.QueryRowContext(ctx, "SHOW GLOBAL STATUS LIKE 'Com_select'").Scan(&key, &count))
		return count
	}
	var singleSelects int64
	for _, scope := range []string{"selected", "absent"} {
		input := &snapshot.Input{}
		input.SetScopeId(scope)
		input.SetName("qa")
		before := selectCount()
		value, err := server.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: pkg, Name: "snapshot"}, Route: spec.RouteRef{Method: "GET", Path: "/_studio/artifact-release-store/snapshot"}}, Input: input})
		if err != nil {
			t.Fatal(err)
		}
		output := value.(*snapshot.Output)
		expected := 0
		if scope == "selected" {
			expected = 1
		}
		if len(output.Data) != expected {
			t.Fatalf("scope %q: rows=%d want=%d", scope, len(output.Data), expected)
		}
		if expected == 1 {
			singleSelects = selectCount() - before
			row := output.Data[0]
			require.NotNil(t, row.Bundle)
			require.NotNil(t, row.Binding)
			require.NotNil(t, row.Operation)
			require.Equal(t, scope, *row.Bundle.ScopeId)
			require.Equal(t, scope, *row.Binding.ScopeId)
			require.Equal(t, scope, *row.Operation.ScopeId)
			require.Equal(t, scope+"-operation", *row.Operation.OperationId)
			require.JSONEq(t, `[{"scope":"`+scope+`"}]`, string(*row.Binding.BindingsJson))
		}

		if expected == 1 && (output.Data[0].ScopeId == nil || *output.Data[0].ScopeId != scope) {
			t.Fatalf("scope predicate leaked foreign row: %+v", output.Data[0])
		}
	}
	input := new(snapshot.Input)
	input.SetScopeId("selected")
	input.SetActiveOnly(true)
	before := selectCount()
	value, err := server.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: pkg, Name: "snapshot"}, Route: spec.RouteRef{Method: "GET", Path: "/_studio/artifact-release-store/snapshot"}}, Input: input})
	require.NoError(t, err)
	batchSelects := selectCount() - before
	rows := value.(*snapshot.Output).Data
	require.Len(t, rows, 2, "active-only predicate must exclude the inactive channel in SQL")
	for _, row := range rows {
		require.Equal(t, "selected", *row.ScopeId)
		require.NotNil(t, row.Bundle)
		require.NotNil(t, row.Binding)
		require.Equal(t, "selected", *row.Bundle.ScopeId)
		require.Equal(t, "selected", *row.Binding.ScopeId)
	}
	require.Positive(t, singleSelects)
	require.Equal(t, singleSelects, batchSelects, "related reads must batch across parents")
	t.Logf("single_channel_selects=%d two_channel_selects=%d", singleSelects, batchSelects)
	// Exercise the production adapter with valid domain metadata as well.
	scope := releasevalue.Scope{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "namespace"}
	bundle := contract.Bundle{Namespace: scope.Namespace, Roots: []artifactcohort.RootRef{{URI: "prompt://namespace/selected", Kind: "prompt", Version: 1, Hash: strings.Repeat("a", 64)}}}
	bundle.Hash = contract.BundleHash(bundle)
	document, err := json.Marshal(bundle)
	require.NoError(t, err)
	_, bindingBytes, bindingHash, err := releasevalue.ValidateBindings(bundle, nil)
	require.NoError(t, err)
	policies, err := json.Marshal([]authz.NamedReference{{Resource: authz.Resource{Tenant: scope.Tenant, Kind: "policy", ID: "readers", Version: "logical"}, Revision: 1}})
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO artifact_release_bundles(scope_id,tenant_id,account_id,workspace_id,namespace,bundle_hash,bundle_json,created_at,created_by,updated_at,updated_by) VALUES(?,'tenant','account','workspace','namespace',?,?,?,'actor',?,'actor')`, scope.ID(), bundle.Hash, document, now, now)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO artifact_release_bundle_bindings(scope_id,bundle_hash,bindings_json,bindings_hash,created_at,created_by,updated_at,updated_by) VALUES(?,?,?,?,?,'actor',?,'actor')`, scope.ID(), bundle.Hash, bindingBytes, bindingHash, now, now)
	require.NoError(t, err)
	for _, name := range []string{"current", "next"} {
		_, err = db.ExecContext(ctx, `INSERT INTO artifact_release_channels(scope_id,tenant_id,account_id,workspace_id,namespace,name,description,policy_refs_json,active_bundle_hash,revision,created_at,created_by,updated_at,updated_by) VALUES(?,'tenant','account','workspace','namespace',?,'Selected',?,?,1,?,'actor',?,'actor')`, scope.ID(), name, policies, bundle.Hash, now, now)
		require.NoError(t, err)
	}
	store, err := nativestore.New(server, scope, func(context.Context) (string, error) { return "actor", nil })
	require.NoError(t, err)
	before = selectCount()
	selections, err := store.ActiveSelections(ctx)
	require.NoError(t, err)
	require.Len(t, selections, 2)
	for _, selected := range selections {
		require.Equal(t, bundle, selected.Bundle)
		require.True(t, scope.Owns(selected.Channel.Key))
	}
	adapterSelects := selectCount() - before
	require.EqualValues(t, 3, adapterSelects, "no pending operation means its relation query is omitted")
	t.Logf("native_store_channels=%d native_store_selects=%d", len(selections), adapterSelects)
}
