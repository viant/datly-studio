package host

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/internal/predicatepolicy"
	"github.com/viant/datly-studio/internal/settingstore"
	"github.com/viant/datly-studio/runtime/preview"
	windowprotocol "github.com/viant/forge/backend/window/contract"
)

func TestReleaseBindingRetainsAcceptedPublicationAfterAuthoringPolicyChange(t *testing.T) {
	fixture, err := newScopedHost(t, map[string]string{"records": unscopedRecordsDQL}, map[string]authz.Policy{"records": {Mode: "public"}})
	require.NoError(t, err)
	ctx := oauth.WithBearer(context.Background(), fixture.token(t, "alice", []authz.Entity{}))
	service := fixture.service
	reader := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}
	store, err := authz.NewStaticStore([]authz.Document{{Resource: authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: scopeTestTenant}, Revision: 1, Policies: map[string]authz.Policy{"describe": reader, "execute": reader}}})
	require.NoError(t, err)
	service.resourceAccess.Store = store
	ref := ComponentReference{Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}
	manager, pin, err := service.componentRuntimeWithAuthority(ctx, ref, false)
	require.NoError(t, err)
	require.NoError(t, manager.Shutdown(ctx))
	artifact := releaseWindowFixture(t, pin)
	_, err = service.studio.ExecContext(ctx, "INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at) VALUES('owner','general','General','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)")
	require.NoError(t, err)
	require.NoError(t, settingstore.Write(ctx, service.studio, "owner", "general", 1, "owner", map[string]json.RawMessage{"security": json.RawMessage(`{"predicatePackages":["example.com/security"],"requireSecurityPredicate":true}`)}))
	bindings, err := CompileReleaseWindowBindings(ctx, artifact, map[string]ComponentReference{"records": ref}, func(ctx context.Context, ref ComponentReference) (windowprotocol.ComponentBinding, error) {
		runtime, binding, err := service.componentRuntimeWithAuthority(ctx, ref, false)
		if runtime != nil {
			defer runtime.Shutdown(ctx)
		}
		return binding, err
	})
	require.NoError(t, err)
	require.Len(t, bindings, 1)
	require.Equal(t, pin, bindings[0].Binding)
	raw, err := service.ExecuteComponentJSON(ctx, ref, bindings[0].Binding, nil)
	require.NoError(t, err)
	require.Contains(t, string(raw), "alpha-101")

	// Explicit authoring validation still enforces the current namespace policy.
	validation := preview.Dynamic{StudioDB: service.studio, RootDir: service.config.RootDir}
	err = validation.Validate(ctx, ref.ID, 1)
	var required *predicatepolicy.RequiredError
	require.True(t, errors.As(err, &required), "authoring requirement was not enforced: %v", err)
	var count int
	require.NoError(t, service.studio.QueryRowContext(ctx, "SELECT COUNT(*) FROM artifact_release_bundles").Scan(&count))
	require.Zero(t, count)
	require.NoError(t, service.studio.QueryRowContext(ctx, "SELECT COUNT(*) FROM artifact_release_channels").Scan(&count))
	require.Zero(t, count)
}

func TestReleaseBindingAcceptsDeclaredLinkedPredicateFromOwningNamespace(t *testing.T) {
	const pkg = "github.com/viant/datly-studio/studio/predicatecatalog/testdata/extension"
	t.Setenv("STUDIO_PREDICATE_PACKAGES", pkg)
	fixture, err := newScopedHost(t, map[string]string{"tasks": unscopedRecordsDQL}, map[string]authz.Policy{"tasks": {Mode: "public"}})
	require.NoError(t, err)
	ctx := context.Background()
	service := fixture.service
	_, err = service.studio.ExecContext(ctx, "INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at) VALUES('owner','general','General','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)")
	require.NoError(t, err)
	raw, err := json.Marshal(map[string]any{"predicatePackages": []string{pkg}, "requireSecurityPredicate": true})
	require.NoError(t, err)
	require.NoError(t, settingstore.Write(ctx, service.studio, "owner", "general", 1, "owner", map[string]json.RawMessage{"security": raw}))
	_, err = service.studio.ExecContext(ctx, "UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id='tasks'", linkedExtensionDQL, linkedExtensionDQL)
	require.NoError(t, err)
	manager, binding, err := service.componentRuntimeWithAuthority(ctx, ComponentReference{Kind: "dynamic", ID: "tasks", Revision: "1", Method: http.MethodGet, Route: "/tasks"}, false)
	require.NoError(t, err)
	defer manager.Shutdown(ctx)
	require.NotEmpty(t, binding.ContentFingerprint)
}
