package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
)

type schemaSearchFunc func(context.Context, *sdk.Connector, sdk.Principal, sdk.SchemaSearchInput) (*sdk.SchemaSearchResult, error)

func (f schemaSearchFunc) Search(ctx context.Context, c *sdk.Connector, p sdk.Principal, in sdk.SchemaSearchInput) (*sdk.SchemaSearchResult, error) {
	return f(ctx, c, p, in)
}

func TestSchemaSearchReauthorizesEveryInvocation(t *testing.T) {
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "reader"})
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "studio.db"))
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, schema.ApplySQLite(ctx, db, "studio"))
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}}
	client, err := sdk.NewClient(transport)
	require.NoError(t, err)
	_, err = client.Connectors().Create(ctx, sdk.CreateConnectorInput{Name: "warehouse", Driver: "sqlite", DSNTemplate: "file:fixture", OwnerID: "reader"})
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE connectors SET status='active' WHERE name='warehouse'`)
	require.NoError(t, err)
	calls, checks := 0, 0
	revoked := false
	denied := &sdk.Error{Code: sdk.ErrorForbidden, Message: "connector access revoked"}
	transport.Authorizer = AuthorizerFunc(func(ctx context.Context, r AuthorizationRequest) error {
		checks++
		require.Equal(t, sdk.OperationConnectorSearch, r.Operation)
		require.Equal(t, "warehouse", r.ConnectorName)
		require.Equal(t, "view", r.Permission)
		if revoked {
			return denied
		}
		return nil
	})
	transport.SchemaSearch = schemaSearchFunc(func(ctx context.Context, c *sdk.Connector, p sdk.Principal, in sdk.SchemaSearchInput) (*sdk.SchemaSearchResult, error) {
		calls++
		require.Equal(t, "reader", p.Subject)
		require.Equal(t, "warehouse", c.Name)
		require.Equal(t, "fixture", in.Schema)
		return &sdk.SchemaSearchResult{SnapshotID: "reused", MetadataComplete: true}, nil
	})
	for i := 0; i < 2; i++ {
		result, err := client.Catalog().Search(ctx, "warehouse", sdk.SchemaSearchInput{Schema: "fixture"})
		require.NoError(t, err)
		require.Equal(t, "reused", result.SnapshotID)
	}
	revoked = true
	_, err = client.Catalog().Search(ctx, "warehouse", sdk.SchemaSearchInput{Schema: "fixture"})
	require.True(t, errors.Is(err, denied))
	require.Equal(t, 3, checks)
	require.Equal(t, 2, calls, "revocation must prevent reaching even a warm schema snapshot")
}
