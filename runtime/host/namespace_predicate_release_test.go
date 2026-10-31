package host

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/internal/predicatepolicy"
	"github.com/viant/datly-studio/internal/settingstore"
	windowprotocol "github.com/viant/forge/backend/window/contract"
)

func TestReleaseBindingRechecksOwningNamespaceAuthorizationPredicate(t *testing.T) {
	fixture, err := newScopedHost(t, map[string]string{"records": unscopedRecordsDQL}, map[string]authz.Policy{"records": {Mode: "public"}})
	require.NoError(t, err)
	ctx := context.Background()
	service := fixture.service
	ref := ComponentReference{Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}
	manager, pin, err := service.componentRuntimeWithAuthority(ctx, ref, false)
	require.NoError(t, err)
	require.NoError(t, manager.Shutdown(ctx))
	artifact := releaseWindowFixture(t, pin)
	_, err = service.studio.ExecContext(ctx, "INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at) VALUES('owner','general','General','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)")
	require.NoError(t, err)
	require.NoError(t, settingstore.Write(ctx, service.studio, "owner", "general", 1, "owner", map[string]json.RawMessage{"security": json.RawMessage(`{"predicatePackages":["example.com/security"],"requireSecurityPredicate":true}`)}))
	var sourceError error
	_, err = CompileReleaseWindowBindings(ctx, artifact, map[string]ComponentReference{"records": ref}, func(ctx context.Context, ref ComponentReference) (windowprotocol.ComponentBinding, error) {
		runtime, binding, err := service.componentRuntimeWithAuthority(ctx, ref, false)
		if runtime != nil {
			defer runtime.Shutdown(ctx)
		}
		sourceError = err
		return binding, err
	})
	require.Error(t, err)
	var required *predicatepolicy.RequiredError
	require.True(t, errors.As(sourceError, &required), "native source gate was bypassed: %v", sourceError)
	var count int
	require.NoError(t, service.studio.QueryRowContext(ctx, "SELECT COUNT(*) FROM artifact_release_bundles").Scan(&count))
	require.Zero(t, count)
	require.NoError(t, service.studio.QueryRowContext(ctx, "SELECT COUNT(*) FROM artifact_release_channels").Scan(&count))
	require.Zero(t, count)
}

func TestReleaseBindingAcceptsDeclaredLinkedPredicateFromOwningNamespace(t *testing.T) {
	const pkg = "github.com/viant/datly-studio/studio/predicatecatalog/testdata/extension"
	t.Setenv("STUDIO_PREDICATE_PACKAGES", pkg)
	fixture, err := newScopedHost(t, map[string]string{"tasks": linkedExtensionDQL}, map[string]authz.Policy{"tasks": {Mode: "public"}})
	require.NoError(t, err)
	ctx := context.Background()
	service := fixture.service
	_, err = service.studio.ExecContext(ctx, "INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at) VALUES('owner','general','General','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)")
	require.NoError(t, err)
	raw, err := json.Marshal(map[string]any{"predicatePackages": []string{pkg}, "requireSecurityPredicate": true})
	require.NoError(t, err)
	require.NoError(t, settingstore.Write(ctx, service.studio, "owner", "general", 1, "owner", map[string]json.RawMessage{"security": raw}))
	manager, binding, err := service.componentRuntimeWithAuthority(ctx, ComponentReference{Kind: "dynamic", ID: "tasks", Revision: "1", Method: http.MethodGet, Route: "/tasks"}, false)
	require.NoError(t, err)
	defer manager.Shutdown(ctx)
	require.NotEmpty(t, binding.ContentFingerprint)
}
