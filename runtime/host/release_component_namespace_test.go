package host

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/viant/authz"
	accessstore "github.com/viant/authz/component/store/sql"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/namespacepolicy"
	"github.com/viant/datly-studio/runtime/namespacemcp"
	"github.com/viant/datly/application"
	identity "github.com/viant/forge/backend/resource"
	windowprotocol "github.com/viant/forge/backend/window/contract"
)

func TestPrivateNamespaceNamedReleaseMCPWindowFreshRowsAndRollback(t *testing.T) {
	testNamedReleaseMCPWindowFreshRowsAndRollback(t, true)
}

type privateReleaseDependencies struct{ revision, calls, closed atomic.Int64 }

func configurePrivateReleaseNamespace(ctx context.Context, t *testing.T, host *Service) *privateReleaseDependencies {
	t.Helper()
	// Match the native release-store SQLite fixtures: one connection serializes
	// SQLite reads/writes while the real background reconciler remains active.
	host.studio.SetMaxOpenConns(1)
	id := namespaceaccess.ID("owner", "demo")
	if _, err := host.studio.ExecContext(ctx, `INSERT INTO namespaces(namespace_id,owner_id,name,title,status,visibility,allowed_roles_json,mcp_enabled,etag,created_at,updated_at) VALUES(?,'owner','demo','Private release','active','private','["studio_viewer"]',1,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := host.studio.ExecContext(ctx, `UPDATE components SET namespace='demo',namespace_id=? WHERE id='records'`, id); err != nil {
		t.Fatal(err)
	}
	host.config.NamespaceID = id
	host.config.namespaceDefinitions = &namespacemcp.DefinitionIndex{}
	host.config.namespaceDefinitions.Replace([]namespacemcp.Definition{{NamespaceID: id, Name: "demo", OwnerID: "owner", Active: true, Enabled: true, Visibility: "private", AllowedRoles: []string{"studio_viewer"}}})
	state := &privateReleaseDependencies{}
	state.revision.Store(1)
	host.config.ExactBuildDependencyResolver = func(context.Context, string, ComponentReference) (*ExactBuildDependencies, error) {
		state.calls.Add(1)
		return &ExactBuildDependencies{ContentFingerprint: identity.ContentFingerprint([]byte("support-" + strconv.FormatInt(state.revision.Load(), 10))), Shutdown: func(context.Context) error { state.closed.Add(1); return nil }}, nil
	}
	return state
}

func installPrivateReleaseSourcePolicy(ctx context.Context, t *testing.T, host *Service, doc authz.Document) {
	t.Helper()
	store := host.resourceAccess.Store.(*namespacepolicy.Store).Store.(*accessstore.Store)
	doc.Revision = 0
	if _, err := store.Provision(ctx, doc, "fixture"); err != nil {
		t.Fatal(err)
	}
}

func checkPrivateReleaseBoundaries(ctx context.Context, t *testing.T, host *Service, recipient, visitor context.Context, ref ComponentReference, pin windowprotocol.ComponentBinding, dependencies *privateReleaseDependencies) {
	t.Helper()
	dependencies.revision.Store(2)
	if _, err := host.ResolveComponentBinding(recipient, ref); err == nil {
		t.Fatal("stale dependency generation described")
	}
	if _, err := host.ExecuteComponentJSON(recipient, ref, pin, nil); err == nil {
		t.Fatal("stale dependency generation executed")
	}
	dependencies.revision.Store(1)
	if dependencies.calls.Load() != dependencies.closed.Load() {
		t.Fatal("exact metadata retained temporary support")
	}
	described, err := host.ResolveComponentBinding(recipient, ref)
	if err != nil || described != pin {
		t.Fatalf("exact released description: %+v %v", described, err)
	}
	if _, err := host.ResolveComponentBinding(visitor, ref); err == nil {
		t.Fatal("unauthorized recipient described source")
	}
	lease, err := host.admitReleasedComponent(recipient, ref, pin)
	if err != nil {
		t.Fatal(err)
	}
	resource := authz.Resource{Kind: "component", ID: ref.ID, Version: ref.Revision, Tenant: host.config.Access.Tenant}
	admitted := context.WithValue(recipient, releaseComponentContextKey{}, lease)
	if err := host.authorizeNamespace(recipient); err == nil {
		t.Fatal("named audience acquired Studio viewer access")
	}
	if err := host.authorizeNamespace(admitted); err == nil {
		t.Fatal("component receipt became namespace-wide access")
	}
	if err := host.authorizeNamespaceResource(admitted, resource); err != nil {
		t.Fatal(err)
	}
	for _, wrong := range []authz.Resource{{Kind: "component", ID: "other", Version: ref.Revision, Tenant: resource.Tenant}, {Kind: "component", ID: ref.ID, Version: "2", Tenant: resource.Tenant}, {Kind: "component", ID: ref.ID, Version: ref.Revision, Tenant: "other"}, {Kind: "skill", ID: ref.ID, Version: ref.Revision, Tenant: resource.Tenant}} {
		if err := host.authorizeNamespaceResource(admitted, wrong); err == nil {
			t.Fatalf("receipt crossed resource boundary: %+v", wrong)
		}
	}
	// A caller cannot manufacture source admission by supplying a receipt from a
	// different identity or root: the execution entry establishes its own lease.
	forged := lease
	forged.Root.URI = "window://demo/unreleased"
	if _, err := host.ExecuteComponentJSON(context.WithValue(visitor, releaseComponentContextKey{}, forged), ref, pin, nil); err == nil {
		t.Fatal("forged recipient/root receipt executed")
	}
	if err := host.fenceReleasedComponent(admitted, forged); err == nil {
		t.Fatal("wrong release root passed the completion fence")
	}
	wrongPin := pin
	wrongPin.ContentFingerprint = "wrong"
	if err := host.authorizeReleasedComponentDescription(recipient, ref, wrongPin); err == nil {
		t.Fatal("wrong source pin described")
	}
	if _, err := host.ExecuteComponentJSON(recipient, ref, wrongPin, nil); err == nil {
		t.Fatal("wrong source pin executed")
	}
	row, _ := host.config.namespaceDefinitions.Lookup(host.config.NamespaceID)
	for _, invalid := range []namespacemcp.Definition{
		{NamespaceID: row.NamespaceID, Active: true, Visibility: "private"},
		{NamespaceID: row.NamespaceID, OwnerID: row.OwnerID, Active: true, Visibility: "invalid"},
	} {
		host.config.namespaceDefinitions.Update(invalid)
		if err := host.authorizeNamespaceResource(admitted, resource); err == nil {
			t.Fatal("receipt bypassed invalid namespace identity/visibility")
		}
	}
	host.config.namespaceDefinitions.Update(row)
	inactive := row
	inactive.Active = false
	host.config.namespaceDefinitions.Update(inactive)
	if _, err := host.ExecuteComponentJSON(recipient, ref, pin, nil); err == nil {
		t.Fatal("inactive namespace executed through release")
	}
	if err := host.fenceReleasedComponent(admitted, lease); err == nil {
		t.Fatal("namespace withdrawal missed completion fence")
	}
	host.config.namespaceDefinitions.Update(row)
	_, err = host.executeExactComponent(recipient, ref, pin, func(callCtx context.Context, manager *application.Manager, _ *exactComponentMetadata) (any, error) {
		rows, readErr := executeRuntimeJSON(callCtx, host.authenticated(manager), ref.Method, ref.Route, nil)
		if readErr != nil || len(rows) == 0 {
			t.Fatalf("native execution before namespace withdrawal: %v", readErr)
		}
		host.config.namespaceDefinitions.Update(inactive)
		return rows, readErr
	}, nil)
	host.config.namespaceDefinitions.Update(row)
	if err == nil {
		t.Fatal("namespace withdrawn during native execution released rows")
	}
	// Named release admission substitutes only plain creator ownership. Explicit
	// denials, entity bounds, OAuth scopes, and composite rules remain mandatory.
	previous := host.resourceAccess.Store
	original, err := previous.Get(ctx, resource)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { host.resourceAccess.Store = previous }()
	if _, err = host.studio.ExecContext(ctx, `INSERT INTO components(id,slug,title,owner_id,namespace,namespace_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) SELECT 'unreleased','unreleased','Unreleased',owner_id,namespace,namespace_id,status,default_connector_name,component_scope,'unreleased',etag,created_at,updated_at FROM components WHERE id='records'`); err != nil {
		t.Fatal(err)
	}
	if _, err = host.studio.ExecContext(ctx, `INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) SELECT 'unreleased',version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at FROM component_versions WHERE report_id='records' AND version_no=1`); err != nil {
		t.Fatal(err)
	}
	otherDoc := original
	otherDoc.Resource.ID = "unreleased"
	both, err := authz.NewStaticStore([]authz.Document{original, otherDoc})
	if err != nil {
		t.Fatal(err)
	}
	host.resourceAccess.Store = both
	otherRef := ref
	otherRef.ID = "unreleased"
	manager, otherPin, err := host.componentRuntimeWithAuthority(ctx, otherRef, false)
	if err != nil {
		t.Fatal(err)
	}
	if err = manager.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = host.ExecuteComponentJSON(admitted, otherRef, otherPin, nil); err == nil {
		t.Fatal("release receipt authorized an unrelated native component")
	}
	// Ordinary policy-less publication keeps its existing namespace protection.
	noPolicies := &accessstore.Store{DB: host.studio}
	defer noPolicies.Close(ctx)
	host.resourceAccess.Store = noPolicies
	otherResource := otherDoc.Resource
	if _, _, _, err := host.publishedAuthorizer(map[string]int{"unreleased": 1}).resourceAccess.AuthorizeWithStatus(recipient, authz.Request{Resource: otherResource, Action: "execute"}); err != nil {
		t.Fatalf("fixture did not exercise ordinary public default: %v", err)
	}
	if _, err := host.ExecuteComponentJSON(recipient, otherRef, otherPin, nil); err == nil {
		t.Fatal("private policy-less component became publicly executable")
	}
	host.resourceAccess.Store = previous
	current := original
	defer func() {
		restore := original
		restore.Revision = current.Revision
		if _, err := previous.Replace(ctx, restore, restore.Revision, "fixture"); err != nil {
			t.Error(err)
		}
	}()
	// Present metadata restrictions are independent of the joined release grant.
	for _, policy := range []authz.Policy{{Mode: "deny"}, {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "owner"}, RequiredScopes: []string{"metadata:read"}}, {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "owner"}, EntityType: "project"}, {Mode: "protected", Rule: &authz.Rule{Kind: "all", Rules: []authz.Rule{{Kind: "subject", Value: "owner"}, {Kind: "role", Value: "restricted"}}}}} {
		doc := original
		doc.Revision = current.Revision
		doc.Policies = map[string]authz.Policy{"execute": original.Policies["execute"], "describe": original.Policies["describe"]}
		doc.Policies["describe"] = policy
		current, err = previous.Replace(ctx, doc, doc.Revision, "fixture")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := host.ResolveComponentBinding(recipient, ref); err == nil {
			t.Fatal("release bypassed explicit source metadata restriction")
		}
	}
	for _, policy := range []authz.Policy{{Mode: "deny"}, {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "owner"}, EntityType: "project"}, {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "owner"}, RequiredScopes: []string{"records:read"}}, {Mode: "protected", Rule: &authz.Rule{Kind: "all", Rules: []authz.Rule{{Kind: "subject", Value: "owner"}, {Kind: "role", Value: "restricted"}}}}} {
		doc := original
		doc.Revision = current.Revision
		doc.Policies = map[string]authz.Policy{"describe": original.Policies["describe"], "execute": policy}
		current, err = previous.Replace(ctx, doc, doc.Revision, "fixture")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := host.ExecuteComponentJSON(recipient, ref, pin, nil); err == nil {
			t.Fatal("release bypassed explicit source constraints")
		}
	}
	host.resourceAccess.Store = previous
}
