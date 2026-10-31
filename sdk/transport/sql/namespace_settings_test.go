package sqltransport

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/internal/settingstore"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"github.com/viant/datly-studio/studio/settings"
	"reflect"
	"testing"
)

func TestNamespaceSettingsAuthorizationCASAndSourcePreservation(t *testing.T) {
	f := newImportFixture(t)
	before, err := f.client.Namespaces().GetSettings(f.ctx, f.report.Namespace)
	require.NoError(t, err)
	bytes := int64(100)
	saved, err := f.client.Namespaces().UpdateSettings(f.ctx, f.report.Namespace, sdk.UpdateNamespaceSettingsInput{ETag: before.ETag, Settings: settings.Namespace{ResourceLimits: &settings.ResourceLimits{MaxBytesProcessed: &bytes}}})
	require.NoError(t, err)
	require.Equal(t, before.ETag+1, saved.ETag)
	_, err = f.client.Namespaces().UpdateSettings(f.ctx, f.report.Namespace, sdk.UpdateNamespaceSettingsInput{ETag: before.ETag, Settings: settings.Namespace{Security: &settings.Security{}}})
	require.Error(t, err)
	other := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "other"})
	_, err = f.client.Namespaces().GetSettings(other, f.report.Namespace)
	require.Error(t, err)
	// Ordinary namespace edits preserve policy rows.
	title := "Updated namespace"
	_, err = f.client.Namespaces().Update(f.ctx, f.report.Namespace, sdk.UpdateNamespaceInput{ETag: saved.ETag, Title: &title})
	require.NoError(t, err)
	got, err := f.client.Namespaces().GetSettings(f.ctx, f.report.Namespace)
	require.NoError(t, err)
	require.EqualValues(t, 100, *got.Settings.ResourceLimits.MaxBytesProcessed)
}
func TestResourceCheckEditInvalidatesValidationAndPreservesSource(t *testing.T) {
	f := newImportFixture(t)
	v, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
	require.NoError(t, err)
	require.False(t, v.ResourceCheckEnabled)
	result, err := f.client.Versions().Apply(f.ctx, f.report.ID, v.VersionNo, sdk.EditCommand{Kind: "set_resource_check", ExpectedSourceRevision: v.SourceRevision, Payload: json.RawMessage(`{"resourceCheckEnabled":true}`)})
	require.NoError(t, err)
	require.True(t, result.Version.ResourceCheckEnabled)
	require.Equal(t, v.SourceRevision+1, result.Version.SourceRevision)
	require.Equal(t, "pending", result.Version.CompileStatus)
	require.Nil(t, result.Version.ValidatedAt)
	require.Equal(t, v.AuthoredDQL, result.Version.AuthoredDQL)
	_, err = f.client.Versions().Apply(f.ctx, f.report.ID, v.VersionNo, sdk.EditCommand{Kind: "set_resource_check", ExpectedSourceRevision: v.SourceRevision, Payload: json.RawMessage(`{"resourceCheckEnabled":false}`)})
	require.Error(t, err)
	result, err = f.client.Versions().Apply(f.ctx, f.report.ID, v.VersionNo, sdk.EditCommand{Kind: "set_resource_check", ExpectedSourceRevision: result.Version.SourceRevision, Payload: json.RawMessage(`{"resourceCheckEnabled":false}`)})
	require.NoError(t, err)
	require.False(t, result.Version.ResourceCheckEnabled)
}

func TestPredicateDiscoveryUsesTheRequestedNamespacePackages(t *testing.T) {
	f := newImportFixture(t)
	catalog, err := predicatecatalog.New(predicatecatalog.Package{Path: "example.com/security/alpha", Types: []reflect.Type{reflect.TypeFor[dynamicPredicateFixture]()}}, predicatecatalog.Package{Path: "example.com/security/beta", Types: []reflect.Type{reflect.TypeFor[dynamicPredicateFixture]()}})
	require.NoError(t, err)
	f.transport.Predicates = catalog
	current, err := f.client.Namespaces().GetSettings(f.ctx, f.report.Namespace)
	require.NoError(t, err)
	_, err = f.client.Namespaces().UpdateSettings(f.ctx, f.report.Namespace, sdk.UpdateNamespaceSettingsInput{ETag: current.ETag, Settings: settings.Namespace{Security: &settings.Security{PredicatePackages: []string{"example.com/security/alpha"}}}})
	require.NoError(t, err)
	namespace, err := f.client.Namespaces().Get(f.ctx, f.report.Namespace)
	require.NoError(t, err)
	page, err := f.client.AuthorizationPredicates().Types(sdk.WithNamespaceSelection(f.ctx, namespace.NamespaceID))
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, "example.com/security/alpha", page.Items[0].PackagePath)
	global, err := f.client.AuthorizationPredicates().Types(f.ctx)
	require.NoError(t, err)
	require.Len(t, global.Items, 2)
	other := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "other"})
	_, err = f.client.AuthorizationPredicates().Types(sdk.WithNamespaceSelection(other, namespace.NamespaceID))
	require.Error(t, err)
}

func TestPublishRechecksNamespacePredicateRequirementBeforeStaging(t *testing.T) {
	f := newImportFixture(t)
	v, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
	require.NoError(t, err)
	validation, err := f.client.Versions().Validate(f.ctx, f.report.ID, v.VersionNo, v.SourceRevision)
	require.NoError(t, err)
	require.True(t, validation.Valid)
	current, err := f.client.Namespaces().GetSettings(f.ctx, f.report.Namespace)
	require.NoError(t, err)
	require.NoError(t, settingstore.Write(f.ctx, f.db, f.report.OwnerID, f.report.Namespace, current.ETag, "owner", map[string]json.RawMessage{"security": json.RawMessage(`{"predicatePackages":["example.com/security"],"requireSecurityPredicate":true}`)}))
	_, err = f.client.Publications().Publish(f.ctx, f.report.ID, v.VersionNo, sdk.PublishInput{ExpectedSourceRevision: v.SourceRevision})
	require.ErrorContains(t, err, "authorization predicate")
	var count int
	require.NoError(t, f.db.QueryRowContext(f.ctx, "SELECT COUNT(*) FROM runtime_generations").Scan(&count))
	require.Zero(t, count)
	require.NoError(t, f.db.QueryRowContext(f.ctx, "SELECT COUNT(*) FROM component_publications").Scan(&count))
	require.Zero(t, count)
	// The owning namespace flag is independent; another namespace's reader is not blocked by it.
	other, err := f.client.Namespaces().Create(f.ctx, sdk.CreateNamespaceInput{Name: "beta", Title: "Beta"})
	require.NoError(t, err)
	otherComponent, err := f.client.Components().Create(f.ctx, sdk.CreateComponentInput{Namespace: other.Name, Slug: "other", Title: "Other", DefaultConnectorName: f.report.DefaultConnectorName})
	require.NoError(t, err)
	otherVersion, err := f.client.Versions().Create(f.ctx, otherComponent.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
	require.NoError(t, err)
	require.NoError(t, f.transport.requireNamespaceAuthorizationPredicate(f.ctx, otherComponent.ID, otherVersion))
}
