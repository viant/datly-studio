// Package schemasearch projects provider metadata into the shared MCP catalog.
// Authorization is performed by the connector owner before access to this index.
package schemasearch

import (
	"context"
	"encoding/json"

	"github.com/viant/datly-studio/sdk"
	catalog "github.com/viant/mcp/client/catalog/index"
)

type Query struct {
	Search           string
	Table            *sdk.DatabaseTable
	ColumnTerms      []string
	DescriptionTerms []string
	Cursor           string
	Limit            int
}
type Match struct {
	Table       sdk.DatabaseTable `json:"table"`
	Column      string            `json:"column,omitempty"`
	Description string            `json:"description,omitempty"`
}
type Index struct{ catalog *catalog.Catalog[Match] }

// New copies compact metadata once. No business rows or detailed schema objects
// are retained. A table entry has no column name, so column filters exclude it.
func New(snapshot string, tables []sdk.TableDetail) *Index {
	entries := make([]catalog.CatalogItem[Match], 0, len(tables))
	for _, table := range tables {
		name := tableIdentity(table.Table)
		compact := table.Table
		compact.Comment = snippet(compact.Comment)
		entries = append(entries, catalog.CatalogItem[Match]{Namespace: name, Description: table.Table.Comment, URI: name, Value: Match{Table: compact, Description: snippet(table.Table.Comment)}})
		for _, column := range table.Columns {
			entries = append(entries, catalog.CatalogItem[Match]{Name: column.Name, Namespace: name, Description: column.Description, URI: name + "#" + column.Name, Value: Match{Table: compact, Column: column.Name, Description: snippet(column.Description)}})
		}
	}
	return &Index{catalog: catalog.NewCatalog(snapshot, entries)}
}
func (i *Index) Search(ctx context.Context, query Query) (*catalog.CatalogPage[Match], error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 7
	}
	if limit > 25 {
		limit = 25
	}
	namespace := ""
	if query.Table != nil {
		namespace = tableIdentity(*query.Table)
	}
	return i.catalog.Page(ctx, catalog.CatalogQuery{Search: query.Search, Namespace: namespace, NameTerms: query.ColumnTerms, DescriptionTerms: query.DescriptionTerms, Cursor: query.Cursor, Limit: limit})
}
func tableIdentity(table sdk.DatabaseTable) string {
	b, _ := json.Marshal([]string{table.Catalog, table.Schema, table.Name})
	return string(b)
}
func snippet(value string) string {
	r := []rune(value)
	if len(r) > 240 {
		return string(r[:237]) + "..."
	}
	return value
}
