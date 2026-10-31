package stage_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	bundlereader "github.com/viant/datly-studio/studio/artifact_release/bundle"
	channels "github.com/viant/datly-studio/studio/artifact_release/channels"
	complete "github.com/viant/datly-studio/studio/artifact_release/complete"
	create_channel "github.com/viant/datly-studio/studio/artifact_release/create_channel"
	fail "github.com/viant/datly-studio/studio/artifact_release/fail"
	history "github.com/viant/datly-studio/studio/artifact_release/history"
	intent "github.com/viant/datly-studio/studio/artifact_release/intent"
	put_bundle "github.com/viant/datly-studio/studio/artifact_release/put_bundle"
	snapshot "github.com/viant/datly-studio/studio/artifact_release/snapshot"
	. "github.com/viant/datly-studio/studio/artifact_release/stage"
	xhandler "github.com/viant/xdatly/handler"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/schema"
	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
	"github.com/viant/datly/bootstrap/connector"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/standalone"
	"github.com/viant/datly/standalone/config"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
	"github.com/viant/x"
	_ "modernc.org/sqlite"
)

// Application data access is supplied solely by the stock-generated graph.
// SQL below is isolated fixture setup/assertion, never a production fallback.
func newStageCase(ctx context.Context, t *testing.T, setupSQL string) (*sql.DB, *standalone.Server, *Input) {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "studio.db")
	db, err := sql.Open("sqlite", filename)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	serverScope := releasevalue.Scope{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "namespace"}
	scope, opID := serverScope.ID(), strings.Repeat("c", 32)
	bundle := contract.Bundle{Namespace: "namespace", Roots: []artifactcohort.RootRef{{URI: "window://namespace/one", Kind: "window", Version: 1, Hash: strings.Repeat("d", 64)}}}
	bundle.Hash = contract.BundleHash(bundle)
	hash := bundle.Hash
	bundleJSON, _ := json.Marshal(bundle)
	policies := []authz.NamedReference{{Resource: authz.Resource{Tenant: "tenant", Kind: "policy", ID: "account/account/qa", Version: "logical"}, Revision: 1}}
	policyJSON, _ := json.Marshal(policies)
	current := contract.Channel{Key: contract.Key{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "namespace", Name: "qa"}, Description: "description", Policies: policies}
	nextChannel := current
	nextChannel.Revision = 1
	nextChannel.ActiveBundleHash = hash
	nextJSON, _ := json.Marshal(nextChannel)
	now := time.Now().UTC().Truncate(time.Second)
	if _, err = db.ExecContext(ctx, `INSERT INTO artifact_release_bundles(scope_id,tenant_id,account_id,workspace_id,namespace,bundle_hash,bundle_json,created_at,created_by,updated_at,updated_by) VALUES(?,?,?,?,?,?,?,?,'actor',?,'actor')`, scope, "tenant", "account", "workspace", "namespace", hash, bundleJSON, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, `INSERT INTO artifact_release_channels(scope_id,tenant_id,account_id,workspace_id,namespace,name,description,policy_refs_json,revision,created_at,created_by,updated_at,updated_by) VALUES(?,?,?,?,?,?,'description',?,0,?,'actor',?,'actor')`, scope, "tenant", "account", "workspace", "namespace", "qa", policyJSON, now, now); err != nil {
		t.Fatal(err)
	}
	if setupSQL != "" {
		if _, err = db.ExecContext(ctx, setupSQL); err != nil {
			t.Fatal(err)
		}
	}
	exports := x.NewRegistry()
	for _, typ := range []reflect.Type{reflect.TypeFor[StageComponent](), reflect.TypeFor[channels.Input](), reflect.TypeFor[channels.InputHas](), reflect.TypeFor[channels.Output](), reflect.TypeFor[channels.ReleaseChannel](), reflect.TypeFor[channels.ChannelsComponent](), reflect.TypeFor[history.Input](), reflect.TypeFor[history.InputHas](), reflect.TypeFor[history.Output](), reflect.TypeFor[history.ReleaseOperation](), reflect.TypeFor[history.HistoryComponent](), reflect.TypeFor[create_channel.Rules](), reflect.TypeFor[create_channel.Input](), reflect.TypeFor[create_channel.InputHas](), reflect.TypeFor[create_channel.Output](), reflect.TypeFor[create_channel.NewChannel](), reflect.TypeFor[create_channel.NewChannelHas](), reflect.TypeFor[create_channel.CreateChannelComponent](), reflect.TypeFor[put_bundle.Rules](), reflect.TypeFor[put_bundle.Input](), reflect.TypeFor[put_bundle.InputHas](), reflect.TypeFor[put_bundle.Output](), reflect.TypeFor[put_bundle.NewBundle](), reflect.TypeFor[put_bundle.NewBundleHas](), reflect.TypeFor[put_bundle.PutBundleComponent](), reflect.TypeFor[intent.Input](), reflect.TypeFor[intent.InputHas](), reflect.TypeFor[intent.Output](), reflect.TypeFor[intent.ReleaseOperation](), reflect.TypeFor[intent.IntentComponent](), reflect.TypeFor[snapshot.Input](), reflect.TypeFor[snapshot.InputHas](), reflect.TypeFor[snapshot.Output](), reflect.TypeFor[snapshot.ReleaseChannel](), reflect.TypeFor[snapshot.StoredBundle](), reflect.TypeFor[snapshot.BundleBinding](), reflect.TypeFor[snapshot.ReleaseOperation](), reflect.TypeFor[snapshot.SnapshotComponent](), reflect.TypeFor[bundlereader.Input](), reflect.TypeFor[bundlereader.InputHas](), reflect.TypeFor[bundlereader.Output](), reflect.TypeFor[bundlereader.StoredBundle](), reflect.TypeFor[bundlereader.BundleBinding](), reflect.TypeFor[bundlereader.BundleComponent](), reflect.TypeFor[complete.Rules](), reflect.TypeFor[complete.Input](), reflect.TypeFor[complete.InputHas](), reflect.TypeFor[complete.Output](), reflect.TypeFor[complete.CompleteChannel](), reflect.TypeFor[complete.CompleteChannelHas](), reflect.TypeFor[complete.CompleteOperation](), reflect.TypeFor[complete.CompleteOperationHas](), reflect.TypeFor[complete.ChannelSnapshot](), reflect.TypeFor[complete.Intent](), reflect.TypeFor[complete.CurrentCompleteView](), reflect.TypeFor[complete.CurrentOperationView](), reflect.TypeFor[complete.CompleteKeysRow](), reflect.TypeFor[complete.CompleteComponent](), reflect.TypeFor[fail.Rules](), reflect.TypeFor[fail.Input](), reflect.TypeFor[fail.InputHas](), reflect.TypeFor[fail.Output](), reflect.TypeFor[fail.FailChannel](), reflect.TypeFor[fail.FailChannelHas](), reflect.TypeFor[fail.FailOperation](), reflect.TypeFor[fail.FailOperationHas](), reflect.TypeFor[fail.ChannelSnapshot](), reflect.TypeFor[fail.Intent](), reflect.TypeFor[fail.CurrentFailView](), reflect.TypeFor[fail.CurrentOperationView](), reflect.TypeFor[fail.FailKeysRow](), reflect.TypeFor[fail.FailComponent](), reflect.TypeFor[Input](), reflect.TypeFor[InputHas](), reflect.TypeFor[Output](), reflect.TypeFor[StageChannel](), reflect.TypeFor[StageChannelHas](), reflect.TypeFor[StageOperation](), reflect.TypeFor[StageOperationHas](), reflect.TypeFor[BundleBinding](), reflect.TypeFor[BundleBindingHas](), reflect.TypeFor[StoredBundle](), reflect.TypeFor[StoredBundleHas](), reflect.TypeFor[CurrentStageView](), reflect.TypeFor[CurrentOperationView](), reflect.TypeFor[CurrentBindingView](), reflect.TypeFor[CurrentBundleView](), reflect.TypeFor[StageRules](), reflect.TypeFor[BindingRules](), reflect.TypeFor[ChannelSnapshot]()} {
		exports.Register(x.NewType(typ))
	}
	pkg := reflect.TypeFor[StageComponent]().PkgPath()
	server, err := standalone.New(ctx, standalone.Options{Config: &config.Config{Connector: "studio", GoBootstrap: &config.Packages{Packages: []string{pkg, reflect.TypeFor[put_bundle.PutBundleComponent]().PkgPath(), reflect.TypeFor[create_channel.CreateChannelComponent]().PkgPath(), reflect.TypeFor[history.HistoryComponent]().PkgPath(), reflect.TypeFor[channels.ChannelsComponent]().PkgPath(), reflect.TypeFor[intent.IntentComponent]().PkgPath(), reflect.TypeFor[snapshot.SnapshotComponent]().PkgPath(), reflect.TypeFor[bundlereader.BundleComponent]().PkgPath(), reflect.TypeFor[complete.CompleteComponent]().PkgPath(), reflect.TypeFor[fail.FailComponent]().PkgPath()}, LinkedOnly: true}, Connectors: []connector.Config{{Name: "studio", Driver: "sqlite", DSN: filename, MaxOpenConns: 1}}}, Registry: exports, Holders: []any{StageComponent{}, put_bundle.PutBundleComponent{}, create_channel.CreateChannelComponent{}, history.HistoryComponent{}, channels.ChannelsComponent{}, intent.IntentComponent{}, snapshot.SnapshotComponent{}, bundlereader.BundleComponent{}, complete.CompleteComponent{}, fail.FailComponent{}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Shutdown(ctx) })
	if err = server.Reload(ctx, 1); err != nil {
		t.Fatal(err)
	}
	parent := &StageChannel{}
	parent.SetScopeId(&scope)
	name := "qa"
	parent.SetName(&name)
	parent.SetRevision(0)
	tenant, account, workspace, namespace, description := "tenant", "account", "workspace", "namespace", "description"
	parent.SetExpectedTenantId(&tenant)
	parent.SetExpectedAccountId(&account)
	parent.SetExpectedWorkspaceId(&workspace)
	parent.SetExpectedNamespace(&namespace)
	parent.SetExpectedDescription(&description)
	refs := json.RawMessage(policyJSON)
	parent.SetExpectedPolicyRefsJson(&refs)
	parent.SetExpectedActiveBundleHash(nil)
	parent.SetPendingOperationId(&opID)
	parent.SetUpdatedAt(&now)
	actor := "actor"
	parent.SetUpdatedBy(&actor)
	parent.SetTargetBundleHash(hash)
	operation := &StageOperation{}
	operation.SetOperationId(&opID)
	operation.SetScopeId(&scope)
	operation.SetChannelName(&name)
	expected, target := int64(0), int64(1)
	operation.SetExpectedRevision(expected)
	operation.SetTargetRevision(target)
	operation.SetTargetBundleHash(&hash)
	next := json.RawMessage(nextJSON)
	operation.SetNextChannelJson(&next)
	deps := json.RawMessage(`[]`)
	operation.SetDependenciesJson(&deps)
	pending := "pending"
	operation.SetStatus(&pending)
	operation.SetCreatedAt(&now)
	operation.SetUpdatedAt(&now)
	operation.SetCreatedBy(&actor)
	operation.SetUpdatedBy(&actor)
	parent.SetOperation([]*StageOperation{operation})
	binding := &BundleBinding{}
	binding.SetScopeId(&scope)
	binding.SetBundleHash(&hash)
	raw := json.RawMessage(`[]`)
	sum := sha256.Sum256(raw)
	bindingHash := hex.EncodeToString(sum[:])
	binding.SetBindingsJson(&raw)
	binding.SetBindingsHash(&bindingHash)
	binding.SetCreatedAt(&now)
	binding.SetUpdatedAt(&now)
	binding.SetCreatedBy(&actor)
	binding.SetUpdatedBy(&actor)
	parent.SetBinding(binding)
	input := &Input{}
	input.SetScopeId(scope)
	input.SetScopeTenant(tenant)
	input.SetScopeAccount(account)
	input.SetScopeWorkspace(workspace)
	input.SetScopeNamespace(namespace)
	input.SetActor(actor)
	input.SetName(name)
	input.SetTargetBundleHash(hash)
	input.SetBindingsHash(bindingHash)
	input.SetExpectedRevision(0)
	input.SetNoPending(true)
	input.SetStage([]*StageChannel{parent})
	return db, server, input
}

func invokeStage(ctx context.Context, server *standalone.Server, input *Input) error {
	pkg := reflect.TypeFor[StageComponent]().PkgPath()
	value, err := server.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: pkg, Name: "stage"}, Route: spec.RouteRef{Method: "PATCH", Path: "/_studio/artifact-release-store/stage"}}, Input: input})
	if err == nil {
		if _, ok := value.(*Output); !ok {
			return errors.New("unexpected native output")
		}
	}
	return err
}
func assertNoStage(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()
	var pending sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT pending_operation_id FROM artifact_release_channels`).Scan(&pending); err != nil {
		t.Fatal(err)
	}
	if pending.Valid {
		t.Fatal("reservation survived failure")
	}
	for _, table := range []string{"artifact_release_operations", "artifact_release_bundle_bindings"} {
		var n int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&n); err != nil || n != 0 {
			t.Fatalf("partial %s count=%d err=%v", table, n, err)
		}
	}
}
func TestNativeStageGraphTransaction(t *testing.T) {
	ctx := context.Background()
	db, server, input := newStageCase(ctx, t, "")
	if err := invokeStage(ctx, server, input); err != nil {
		t.Fatal(err)
	}
	var got string
	if err := db.QueryRowContext(ctx, `SELECT pending_operation_id FROM artifact_release_channels`).Scan(&got); err != nil || got != text(input.Stage[0].PendingOperationId) {
		t.Fatalf("pending=%q err=%v", got, err)
	}
	for _, table := range []string{"artifact_release_operations", "artifact_release_bundle_bindings"} {
		var n int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&n); err != nil || n != 1 {
			t.Fatalf("%s count=%d err=%v", table, n, err)
		}
	}
}
func TestNativeStageRejectsStaleAndForeignFacts(t *testing.T) {
	for _, tc := range []struct {
		name, setup string
		mutate      func(*Input)
		conflict    bool
	}{
		{name: "stale revision", setup: `UPDATE artifact_release_channels SET revision=1`, conflict: true},
		{name: "foreign trusted scope", mutate: func(input *Input) { input.SetScopeAccount("other") }},
		{name: "wrong bundle", mutate: func(input *Input) { input.SetTargetBundleHash(strings.Repeat("f", 64)) }},
		{name: "invalid next channel", mutate: func(input *Input) { bad := json.RawMessage(`{}`); input.Stage[0].Operation[0].SetNextChannelJson(&bad) }},
		{name: "invalid binding fingerprint", mutate: func(input *Input) { bad := strings.Repeat("f", 64); input.Stage[0].Binding.SetBindingsHash(&bad) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, server, input := newStageCase(ctx, t, tc.setup)
			if tc.mutate != nil {
				tc.mutate(input)
			}
			err := invokeStage(ctx, server, input)
			if err == nil {
				t.Fatal("invalid stage accepted")
			}
			if tc.conflict {
				var native *xhandler.Conflict
				if !errors.Is(err, contract.ErrConflict) && !errors.As(err, &native) {
					t.Fatalf("stale error changed: %v", err)
				}
			}
			assertNoStage(ctx, t, db)
		})
	}
}
func TestNativeStageRollsBackLateFailure(t *testing.T) {
	ctx := context.Background()
	db, server, input := newStageCase(ctx, t, `CREATE TRIGGER stop_stage BEFORE INSERT ON artifact_release_operations BEGIN SELECT RAISE(ABORT,'synthetic late stage failure');END;`)
	if err := invokeStage(ctx, server, input); err == nil {
		t.Fatal("late fixture failure accepted")
	}
	assertNoStage(ctx, t, db)
}
func TestNativeStageDoesNotMutateAuxiliaryOrChannelPayload(t *testing.T) {
	ctx := context.Background()
	db, server, input := newStageCase(ctx, t, `CREATE TRIGGER immutable_bundle BEFORE UPDATE ON artifact_release_bundles BEGIN SELECT RAISE(ABORT,'auxiliary write');END;
CREATE TRIGGER only_reserve BEFORE UPDATE OF tenant_id,account_id,workspace_id,namespace,description,policy_refs_json,active_bundle_hash,active_dependencies_json,created_at,created_by ON artifact_release_channels BEGIN SELECT RAISE(ABORT,'non-stage column updated');END;`)
	if err := invokeStage(ctx, server, input); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM artifact_release_operations`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("operations=%d err=%v", count, err)
	}
}

func TestNativeStageRejectsExistingPendingReservation(t *testing.T) {
	ctx := context.Background()
	prior := strings.Repeat("e", 32)
	db, server, input := newStageCase(ctx, t, `UPDATE artifact_release_channels SET pending_operation_id='`+prior+`'`)
	if err := invokeStage(ctx, server, input); !errors.Is(err, contract.ErrConflict) {
		t.Fatalf("pending guard error=%v", err)
	}
	var value string
	if err := db.QueryRowContext(ctx, `SELECT pending_operation_id FROM artifact_release_channels`).Scan(&value); err != nil || value != prior {
		t.Fatalf("prior reservation changed: %s %v", value, err)
	}
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM artifact_release_operations`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("partial operation %d %v", n, err)
	}
}
func TestNativeStageReusesImmutableBindingWithoutUpdate(t *testing.T) {
	ctx := context.Background()
	db, server, input := newStageCase(ctx, t, "")
	if err := invokeStage(ctx, server, input); err != nil {
		t.Fatal(err)
	}
	// Fixture advances the prior attempt to a failed/unreserved state; production
	// failure completion will be a separate generated graph.
	if _, err := db.ExecContext(ctx, `UPDATE artifact_release_channels SET pending_operation_id=NULL;UPDATE artifact_release_operations SET status='failed';CREATE TRIGGER no_binding_update BEFORE UPDATE ON artifact_release_bundle_bindings BEGIN SELECT RAISE(ABORT,'immutable binding update');END;`); err != nil {
		t.Fatal(err)
	}
	row := *input.Stage[0]
	flags := *row.Has
	row.Has = &flags
	operation := *row.Operation[0]
	opFlags := *operation.Has
	operation.Has = &opFlags
	binding := *row.Binding
	bindFlags := *binding.Has
	binding.Has = &bindFlags
	actor, id := "second-actor", strings.Repeat("f", 32)
	now := time.Now().UTC()
	row.SetUpdatedBy(&actor)
	row.SetUpdatedAt(&now)
	row.SetPendingOperationId(&id)
	operation.SetOperationId(&id)
	operation.SetUpdatedBy(&actor)
	operation.SetCreatedBy(&actor)
	operation.SetCreatedAt(&now)
	operation.SetUpdatedAt(&now)
	binding.SetCreatedBy(&actor)
	binding.SetUpdatedBy(&actor)
	binding.SetCreatedAt(&now)
	binding.SetUpdatedAt(&now)
	row.SetOperation([]*StageOperation{&operation})
	row.SetBinding(nil)
	next := &Input{}
	next.SetScopeId(input.ScopeId)
	next.SetScopeTenant(input.ScopeTenant)
	next.SetScopeAccount(input.ScopeAccount)
	next.SetScopeWorkspace(input.ScopeWorkspace)
	next.SetScopeNamespace(input.ScopeNamespace)
	next.SetActor(actor)
	next.SetName(input.Name)
	next.SetTargetBundleHash(input.TargetBundleHash)
	next.SetBindingsHash(input.BindingsHash)
	next.SetExpectedRevision(0)
	next.SetNoPending(true)
	next.SetStage([]*StageChannel{&row})
	if err := invokeStage(ctx, server, next); err != nil {
		t.Fatal(err)
	}
	var creator string
	if err := db.QueryRowContext(ctx, `SELECT created_by FROM artifact_release_bundle_bindings`).Scan(&creator); err != nil || creator != "actor" {
		t.Fatalf("binding audit changed %q %v", creator, err)
	}
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM artifact_release_operations`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("operation count=%d %v", n, err)
	}
}

func text(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
