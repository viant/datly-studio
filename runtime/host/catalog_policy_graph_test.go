package host

import (
	"context"
	"database/sql"
	accessstore "github.com/viant/authz/component/store/sql"
	"github.com/viant/datly-studio/schema"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/studio/resource_policy/discovery"
)

func TestCatalogGraphKeepsVersionTenantAndCorruptHeadsDistinct(t *testing.T) {
	policy := `{"discover":{"mode":"public"},"describe":{"mode":"deny"}}`
	g, err := newCatalogPolicyGraph("tenant", "scope", []*discovery.PolicyHead{
		{ResourceKind: "namespace", ResourceId: "scope"},
		{ResourceKind: "component", ResourceId: "reader", ResourceVersion: "1", Revision: 7, PoliciesJson: &policy},
		{ResourceKind: "component", ResourceId: "reader", ResourceVersion: "2", Revision: 1}, // missing revision row
		{ResourceKind: "component", ResourceId: "unrestricted"},
	})
	require.NoError(t, err)
	require.True(t, g.components["reader"])
	require.Contains(t, g.components, "unrestricted")
	require.False(t, g.components["unrestricted"])
	key := authz.Resource{Tenant: "tenant", Kind: "component", ID: "reader", Version: "1"}
	doc, err := g.get(key)
	require.NoError(t, err)
	require.EqualValues(t, 7, doc.Revision)
	require.Equal(t, "deny", doc.Policies["describe"].Mode)
	key.Version = "2"
	_, err = g.get(key)
	require.ErrorIs(t, err, authz.ErrUnavailable)
	key.Version = "3"
	_, err = g.get(key)
	require.ErrorIs(t, err, sql.ErrNoRows)
	key.Version = "1"
	key.Tenant = "another"
	_, err = g.get(key)
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = newCatalogPolicyGraph("tenant", "scope", nil)
	require.ErrorIs(t, err, authz.ErrDenied)
}

func TestNativeCatalogGraphTracksCommittedPolicyRevision(t *testing.T) {
	ctx := context.Background()
	path := "file:" + filepath.Join(t.TempDir(), "catalog.db")
	reader, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer reader.Close()
	require.NoError(t, schema.ApplySQLite(ctx, reader, "studio"))
	_, err = reader.ExecContext(ctx, `INSERT INTO namespaces(namespace_id,owner_id,name,title,status,visibility,etag,created_at,updated_at) VALUES('scope','owner','scope','Scope','active','private',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	graphStore, err := newPublishedDefinitionStore(reader)
	require.NoError(t, err)
	defer graphStore.Close(ctx)
	writer, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer writer.Close()
	store := &accessstore.Store{DB: writer}
	defer store.Close(ctx)
	key := authz.Resource{Kind: "namespace", ID: "scope", Version: "1", Tenant: "tenant"}
	graph, err := graphStore.readCatalogPolicyGraph(ctx, "tenant", "scope")
	require.NoError(t, err)
	_, err = graph.get(key)
	require.ErrorIs(t, err, sql.ErrNoRows)
	doc, err := store.Provision(ctx, authz.Document{Resource: key, Policies: map[string]authz.Policy{"discover": {Mode: "public"}}}, "owner")
	require.NoError(t, err)
	graph, err = graphStore.readCatalogPolicyGraph(ctx, "tenant", "scope")
	require.NoError(t, err)
	got, err := graph.get(key)
	require.NoError(t, err)
	require.Equal(t, doc.Revision, got.Revision)
	unchanged, err := graphStore.readCatalogPolicyGraph(ctx, "tenant", "scope")
	require.NoError(t, err)
	require.Same(t, graph, unchanged)
	doc.Policies["discover"] = authz.Policy{Mode: "deny"}
	changed, err := store.Replace(ctx, doc, doc.Revision, "owner")
	require.NoError(t, err)
	graph, err = graphStore.readCatalogPolicyGraph(ctx, "tenant", "scope")
	require.NoError(t, err)
	got, err = graph.get(key)
	require.NoError(t, err)
	require.NotSame(t, unchanged, graph)
	require.Equal(t, changed.Revision, got.Revision)
	require.Equal(t, "deny", got.Policies["discover"].Mode)
}
