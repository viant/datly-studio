package sqltransport

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/studio/settings"
	"testing"
)

func TestNamespaceInternalPolicyPersistsAndRejectsPublicPublication(t *testing.T) {
	f := newImportFixture(t)
	v, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
	require.NoError(t, err)
	validation, err := f.client.Versions().Validate(f.ctx, f.report.ID, v.VersionNo, v.SourceRevision)
	require.NoError(t, err)
	require.True(t, validation.Valid)
	before, err := f.client.Namespaces().GetSettings(f.ctx, f.report.Namespace)
	require.NoError(t, err)
	input := sdk.UpdateNamespaceSettingsInput{ETag: before.ETag, Settings: settings.Namespace{Security: &settings.Security{RequireInternalComponents: true}}}
	_, err = f.client.Namespaces().UpdateSettings(f.ctx, f.report.Namespace, input)
	require.Error(t, err, "ordinary owner cannot change namespace policy")
	admin := settingsAdminContext(t, f.ctx)
	saved, err := f.client.Namespaces().UpdateSettings(admin, f.report.Namespace, input)
	require.NoError(t, err)
	require.True(t, saved.Settings.Security.RequireInternalComponents)
	_, err = f.client.Namespaces().UpdateSettings(admin, f.report.Namespace, input)
	require.Error(t, err, "stale ETag must not overwrite the restriction")
	_, err = f.client.Publications().Publish(f.ctx, f.report.ID, v.VersionNo, sdk.PublishInput{ExpectedSourceRevision: v.SourceRevision})
	require.ErrorContains(t, err, "internal DQL components")
	var count int
	require.NoError(t, f.db.QueryRowContext(f.ctx, "SELECT COUNT(*) FROM runtime_generations").Scan(&count))
	require.Zero(t, count)
	readback, err := f.client.Namespaces().GetSettings(f.ctx, f.report.Namespace)
	require.NoError(t, err)
	require.True(t, readback.Settings.Security.RequireInternalComponents)
	// A policy in one namespace never restricts another namespace implicitly.
	other, err := f.client.Namespaces().Create(admin, sdk.CreateNamespaceInput{Name: "unrestricted", Title: "Unrestricted"})
	require.NoError(t, err)
	c, err := f.client.Components().Create(f.ctx, sdk.CreateComponentInput{Namespace: other.Name, Slug: "reader", Title: "Reader", DefaultConnectorName: f.report.DefaultConnectorName})
	require.NoError(t, err)
	otherVersion, err := f.client.Versions().Create(f.ctx, c.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
	require.NoError(t, err)
	require.NoError(t, f.transport.requireNamespaceAuthorizationPredicate(f.ctx, c.ID, otherVersion))
}
