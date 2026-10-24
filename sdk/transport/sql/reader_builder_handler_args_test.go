package sqltransport

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/transcribe/dql"
	"github.com/viant/datly/typecatalog"
	"github.com/viant/x"
	xpredicate "github.com/viant/xdatly/predicate"
)

// This handler supplies real method authority to the authoring compiler. The
// test verifies editing/persistence, not execution of a business predicate.
type persistedArgumentHandler struct{}

func (*persistedArgumentHandler) Compute(context.Context, any, ...any) (*xpredicate.Criteria, error) {
	return nil, nil
}

func TestReaderBuilderHandlerArgumentsPersistThroughSQLiteReload(t *testing.T) {
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"})
	path := filepath.Join(t.TempDir(), "studio.db")
	db, err := sql.Open("sqlite", "file:"+path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, schema.ApplySQLite(ctx, db, "studio"))
	types := typecatalog.NewCatalog()
	require.NoError(t, types.Register(typecatalog.TypeOriginPackage, x.NewType(reflect.TypeFor[persistedArgumentHandler](), x.WithPkgPath("example.com/filters"), x.WithName("Scope"))))
	newClient := func() sdk.Client {
		client, e := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}, RuntimeTypes: types})
		require.NoError(t, e)
		return client
	}
	client := newClient()
	connector, err := client.Connectors().Create(ctx, sdk.CreateConnectorInput{Name: "main", Driver: "sqlite"})
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE connectors SET status='active' WHERE name=?`, connector.Name)
	require.NoError(t, err)
	component, err := client.Components().Create(ctx, sdk.CreateComponentInput{Slug: "predicate-arguments", Title: "Predicate arguments", DefaultConnectorName: connector.Name})
	require.NoError(t, err)
	source := `#setting($_ = $route('/records','GET'))
#define($_ = $IDs<[]int>(query/ids).Optional())
SELECT records.* FROM (SELECT r.id,r.name FROM records r) records`
	version, err := client.Versions().Create(ctx, component.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: source})
	require.NoError(t, err)
	operation := json.RawMessage(`{"type":"addFieldPredicate","predicate":{"field":"IDs","view":"records","group":0,"name":"handler","args":["example.com/filters.Scope"],"handlerArgs":["alias","r","enabled",false,"limit",10,"nullable",null]}}`)
	result, err := client.Versions().ApplyReaderCommand(ctx, component.ID, version.VersionNo, sdk.ReaderBuilderCommand{ExpectedSourceRevision: version.SourceRevision, Operation: operation})
	require.NoError(t, err)
	require.True(t, result.Applied, "%+v", result.Inspection.Diagnostics)
	expected := []any{"alias", "r", "enabled", false, "limit", 10, "nullable", nil}
	assertArguments := func(text string, want []any) {
		t.Helper()
		declarations, e := dql.DeclarationOccurrences(text)
		require.NoError(t, e)
		require.Len(t, declarations[0].Predicates, 1)
		require.Equal(t, []string{"example.com/filters.Scope"}, declarations[0].Predicates[0].Predicate.Args)
		require.Equal(t, want, declarations[0].Predicates[0].Predicate.HandlerArgs)
	}
	assertArguments(result.Inspection.DQL, expected)
	require.NoError(t, db.Close())
	db, err = sql.Open("sqlite", "file:"+path)
	require.NoError(t, err)
	client = newClient()
	loaded, err := client.Versions().Get(ctx, component.ID, version.VersionNo)
	require.NoError(t, err)
	assertArguments(loaded.AuthoredDQL, expected)
	// Explicit [] clears persisted configuration through the same raw JSON path.
	clear := json.RawMessage(`{"type":"updateFieldPredicate","predicate":{"field":"IDs","occurrence":0,"view":"records","group":0,"name":"handler","args":["example.com/filters.Scope"],"handlerArgs":[]}}`)
	cleared, err := client.Versions().ApplyReaderCommand(ctx, component.ID, version.VersionNo, sdk.ReaderBuilderCommand{ExpectedSourceRevision: loaded.SourceRevision, Operation: clear})
	require.NoError(t, err)
	require.True(t, cleared.Applied, "%+v", cleared.Inspection.Diagnostics)
	assertArguments(cleared.Inspection.DQL, nil)
	stored, err := client.Versions().Get(ctx, component.ID, version.VersionNo)
	require.NoError(t, err)
	assertArguments(stored.AuthoredDQL, nil)
}
