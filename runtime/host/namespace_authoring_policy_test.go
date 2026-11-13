package host

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/studio/namespaces/change"
	"github.com/viant/datly-studio/studio/settings"
	"github.com/viant/datly-studio/studio/sourcevalidation"
)

func TestAuthoringPredicateRequirementDoesNotInvalidatePublishedGeneration(t *testing.T) {
	dql := strings.Replace(unscopedRecordsDQL, "#define($_ = $Records", "#setting($_ = $mcp('records.list','List records'))\n#define($_ = $Records", 1)
	const pkg = "github.com/viant/datly-studio/studio/predicatecatalog/testdata/extension"
	fixture, err := newScopedHostConfiguredPackages(t, map[string]string{"records": dql}, map[string]authz.Policy{"records": {Mode: "public"}}, nil, []string{pkg})
	require.NoError(t, err)
	ctx := context.Background()
	token := fixture.token(t, "alice", []authz.Entity{})
	assertAvailable := func() {
		t.Helper()
		status, body := fixture.get(t, "/records", token, nil)
		require.Equal(t, http.StatusOK, status, body)
		require.Contains(t, body, "alpha-101")
		_, body = fixture.callTool(t, token, "records.list", nil)
		require.Contains(t, body, "alpha-101")
	}
	assertAvailable()
	_, err = fixture.service.studio.ExecContext(ctx, `UPDATE namespace_settings SET value_json='{"requireSecurityPredicate":true,"predicatePackages":["github.com/viant/datly-studio/studio/predicatecatalog/testdata/extension"]}' WHERE owner_id='owner' AND namespace='general' AND setting_key='security'`)
	require.NoError(t, err)
	require.NoError(t, fixture.service.ApplyNamespaceSettings(ctx, change.Identity{NamespaceID: namespaceaccess.ID("owner", "general"), OwnerID: "owner", Name: "general", SettingsChanged: true, SecurityChanged: true, Revision: 2, Security: &settings.Security{RequireSecurityPredicate: true, PredicatePackages: []string{pkg}}}))
	assertAvailable()
	hookCalls := 0
	fixture.service.config.SourceValidator = func(_ context.Context, input sourcevalidation.Input) error {
		hookCalls++
		require.False(t, input.RequireSecurityPredicate, "published loading must not rerun authoring admission")
		return nil
	}
	for attempt := 0; attempt < 2; attempt++ {
		require.NoError(t, fixture.service.Reload(ctx, 0))
		assertAvailable()
	}
	require.Positive(t, hookCalls)
}
