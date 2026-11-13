package schemasearch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/sdk"
	catalog "github.com/viant/mcp/client/catalog/index"
)

func TestSchemaSearchFieldsAndBounds(t *testing.T) {
	tables := make([]sdk.TableDetail, 10001)
	for i := range tables {
		tables[i] = sdk.TableDetail{Table: sdk.DatabaseTable{Catalog: "source", Schema: "data", Name: fmt.Sprintf("records_%05d", i)}, Columns: []sdk.DatabaseColumn{{Name: "id", Description: "record identity"}}}
	}
	tables[5000].Table.Name = "shipments"
	tables[5000].Columns = append(tables[5000].Columns, sdk.DatabaseColumn{Name: "delivery_zone", Description: "Geographical destination region for dispatched parcels"})
	index := New("revision-one", tables)
	for _, q := range []Query{{Search: "parcels"}, {ColumnTerms: []string{"delivery_zone"}}, {DescriptionTerms: []string{"destination region"}}, {Table: &tables[5000].Table, ColumnTerms: []string{"delivery_zone"}, DescriptionTerms: []string{"parcels"}}} {
		page, err := index.Search(context.Background(), q)
		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		require.Equal(t, "shipments", page.Items[0].Table.Name)
		require.Equal(t, "delivery_zone", page.Items[0].Column)
	}
	page, err := index.Search(context.Background(), Query{})
	require.NoError(t, err)
	require.Len(t, page.Items, 7)
	require.NotEmpty(t, page.NextCursor)
	body, err := json.Marshal(page)
	require.NoError(t, err)
	require.Less(t, len(body), 4096)
	_, err = index.Search(context.Background(), Query{Search: "changed", Cursor: page.NextCursor})
	require.ErrorIs(t, err, catalog.ErrCatalogCursor)
	_, err = New("revision-two", tables).Search(context.Background(), Query{Cursor: page.NextCursor})
	require.ErrorIs(t, err, catalog.ErrCatalogCursor)
	page, err = index.Search(context.Background(), Query{ColumnTerms: []string{"shipments"}})
	require.NoError(t, err)
	require.Empty(t, page.Items, "table name is not a column-name match")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = index.Search(ctx, Query{})
	require.ErrorIs(t, err, context.Canceled)
	require.LessOrEqual(t, len([]rune(snippet(strings.Repeat("é", 400)))), 240)
}
