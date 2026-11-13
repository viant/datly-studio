package connectivity

import (
	"context"
	"fmt"
	"strings"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/sqlx/metadata"
	"github.com/viant/sqlx/metadata/info"
	mysqlmeta "github.com/viant/sqlx/metadata/product/mysql"
	"github.com/viant/sqlx/metadata/sink"
	"github.com/viant/sqlx/option"
)

// SchemaMetadataInput is a scoped provider-metadata page, not a business query.
// PageToken is independent of the search index's result cursor. Limit bounds
// columns on MySQL and tables on BigQuery (capped at 25 tables).
type SchemaMetadataInput struct {
	Catalog, Schema, Table, PageToken string
	Limit                             int
}
type SchemaMetadataPage struct {
	Tables        []sdk.TableDetail
	NextPageToken string
	Complete      bool
}

// ColumnMetadataPage batches a database's column metadata through SQLX. It does
// not fetch relations or execute data queries for each table. Other providers
// require their native metadata adapter; unsupported paging never falls back to
// an unbounded scan.
func (c SQLCatalog) ColumnMetadataPage(ctx context.Context, connector *sdk.Connector, input SchemaMetadataInput) (*SchemaMetadataPage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	scope, err := resolveDiscoveryScope(connector, input.Catalog, input.Schema)
	if err != nil {
		return nil, err
	}
	if scope.Schema == "" {
		return nil, fmt.Errorf("metadata search requires an explicit database/dataset")
	}
	driver := resolveDriver(strings.ToLower(strings.TrimSpace(connector.Driver)))
	if driver != "mysql" && driver != "bigquery" {
		return nil, fmt.Errorf("batched column metadata is not available for driver %s", driver)
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 1000
		if driver == "bigquery" {
			limit = 25
		}
	}
	if limit > 1000 {
		return nil, fmt.Errorf("metadata page limit exceeds 1000")
	}
	db, err := openDiscoveryDB(ctx, connector, c.Secrets, c.Const, scope)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if driver == "bigquery" {
		return bigQueryMetadataPage(ctx, db, scope, input, limit)
	}
	page := &option.MetadataPage{Limit: int64(limit), Token: input.PageToken}
	var columns []sink.Column
	if err = metadata.New().Info(ctx, db, info.KindTable, &columns, mysqlmeta.MySQL5(), option.NewArgs(scope.Catalog, scope.Schema, input.Table), page); err != nil {
		return nil, err
	}
	if !page.Handled {
		return nil, fmt.Errorf("metadata provider does not support bounded column pages")
	}
	return columnMetadataPage(columns, page.NextToken), nil
}
func columnMetadataPage(columns []sink.Column, next string) *SchemaMetadataPage {
	result := &SchemaMetadataPage{NextPageToken: next, Complete: next == ""}
	byTable := map[[3]string]int{}
	for _, column := range columns {
		key := [3]string{column.Catalog, column.Schema, column.Table}
		index, ok := byTable[key]
		if !ok {
			index = len(result.Tables)
			byTable[key] = index
			result.Tables = append(result.Tables, sdk.TableDetail{Table: sdk.DatabaseTable{Catalog: column.Catalog, Schema: column.Schema, Name: column.Table}, MetadataSource: "catalog_page", ConstraintsKnown: false})
		}
		known := column.IsNullable() || column.IsNotNull()
		result.Tables[index].Columns = append(result.Tables[index].Columns, sdk.DatabaseColumn{Name: column.Name, Description: column.Comments, Type: column.Type, Nullable: column.IsNullable(), NullableKnown: &known})
	}
	return result
}
