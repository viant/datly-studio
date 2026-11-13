package connectivity

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/sqlx/io/config"
	"github.com/viant/sqlx/metadata"
	"github.com/viant/sqlx/metadata/info"
	bqmeta "github.com/viant/sqlx/metadata/product/bigquery"
	"github.com/viant/sqlx/metadata/sink"
	"github.com/viant/sqlx/option"
	"golang.org/x/sync/errgroup"
)

func bigQueryMetadataPage(ctx context.Context, db *sql.DB, scope discoveryScope, input SchemaMetadataInput, limit int) (*SchemaMetadataPage, error) {
	if scope.Catalog == "" {
		return nil, fmt.Errorf("metadata search requires an explicit project/catalog")
	}
	if limit > 25 {
		limit = 25
	}
	page := &option.MetadataPage{Limit: int64(limit), Token: input.PageToken}
	var tables []sink.Table
	if input.Table != "" {
		if input.PageToken != "" {
			return nil, fmt.Errorf("single-table metadata does not accept a page token")
		}
		tables = []sink.Table{{Catalog: scope.Catalog, Schema: scope.Schema, Name: input.Table}}
	} else {
		if err := metadata.New().Info(ctx, db, info.KindTables, &tables, bqmeta.BigQuery(), option.NewArgs(scope.Catalog, scope.Schema), page); err != nil {
			return nil, err
		}
		if !page.Handled {
			return nil, fmt.Errorf("metadata provider did not handle table pagination")
		}
	}
	result := &SchemaMetadataPage{Tables: make([]sdk.TableDetail, len(tables)), NextPageToken: page.NextToken, Complete: page.NextToken == ""}
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(4)
	for i, table := range tables {
		i, table := i, table
		group.Go(func() error {
			columns, err := config.Columns(groupCtx, &sink.Session{Catalog: scope.Catalog, Schema: scope.Schema}, db, table.Name, bqmeta.BigQuery())
			if err != nil {
				return fmt.Errorf("discover table columns %s: %w", table.Name, err)
			}
			if len(columns) == 0 {
				return fmt.Errorf("table column metadata returned no columns for %s", table.Name)
			}
			shaped := columnMetadataPage(columns, "")
			if len(shaped.Tables) != 1 {
				return fmt.Errorf("table metadata returned inconsistent identities for %s", table.Name)
			}
			result.Tables[i] = shaped.Tables[0]
			result.Tables[i].Table = sdk.DatabaseTable{Catalog: scope.Catalog, Schema: scope.Schema, Name: table.Name}
			if table.Type != nil {
				result.Tables[i].Table.Type = *table.Type
			}
			if table.Comment != nil {
				result.Tables[i].Table.Comment = *table.Comment
			}
			result.Tables[i].MetadataSource = "table_api"
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return result, nil
}
