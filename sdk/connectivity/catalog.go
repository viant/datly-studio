package connectivity

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/viant/datly-studio/internal/connectorinit"
	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/sqlx/io/config"
	"github.com/viant/sqlx/metadata"
	"github.com/viant/sqlx/metadata/info"
	"github.com/viant/sqlx/metadata/sink"
	"github.com/viant/sqlx/option"
)

type SQLCatalog struct{ Secrets connectorsecret.Resolver }

func (c SQLCatalog) Schemas(ctx context.Context, connector *sdk.Connector, input sdk.SchemaCatalogInput) (*sdk.SchemaCatalog, error) {
	db, err := openCatalogDB(ctx, connector, c.Secrets)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	var schemas []sink.Schema
	err = metadata.New().Info(ctx, db, info.KindSchemas, &schemas)
	if err != nil {
		session, sessionErr := config.Session(ctx, db)
		if sessionErr != nil {
			return nil, err
		}
		schemas = []sink.Schema{{Catalog: session.Catalog, Name: session.Schema}}
	}
	query := strings.ToLower(strings.TrimSpace(input.Query))
	result := &sdk.SchemaCatalog{}
	for _, item := range schemas {
		if query == "" || strings.Contains(strings.ToLower(item.Name), query) {
			result.Items = append(result.Items, sdk.DatabaseSchema{Catalog: item.Catalog, Name: item.Name})
		}
	}
	sort.Slice(result.Items, func(i, j int) bool { return result.Items[i].Name < result.Items[j].Name })
	return result, nil
}

func (c SQLCatalog) Tables(ctx context.Context, connector *sdk.Connector, input sdk.TableCatalogInput) (*sdk.TableCatalog, error) {
	db, err := openCatalogDB(ctx, connector, c.Secrets)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	session, err := config.Session(ctx, db)
	if err != nil {
		return nil, err
	}
	schema := input.Schema
	if schema == "" {
		schema = session.Schema
	}
	var tables []sink.Table
	if err = metadata.New().Info(ctx, db, info.KindTables, &tables, option.NewArgs(session.Catalog, schema)); err != nil {
		return nil, err
	}
	query := strings.ToLower(strings.TrimSpace(input.Query))
	var items []sdk.DatabaseTable
	for _, item := range tables {
		if query != "" && !strings.Contains(strings.ToLower(item.Name), query) {
			continue
		}
		comment := ""
		if item.Comment != nil {
			comment = *item.Comment
		}
		kind := "TABLE"
		if item.Type != nil && *item.Type != "" {
			kind = *item.Type
		}
		items = append(items, sdk.DatabaseTable{Catalog: item.Catalog, Schema: item.Schema, Name: item.Name, Type: kind, Comment: comment})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	limit := input.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	if offset > len(items) {
		offset = len(items)
	}
	return &sdk.TableCatalog{Items: items[offset:end], Limit: limit, Offset: offset}, nil
}

func (c SQLCatalog) Table(ctx context.Context, connector *sdk.Connector, input sdk.TableDetailInput) (*sdk.TableDetail, error) {
	if strings.TrimSpace(input.Table) == "" {
		return nil, fmt.Errorf("table is required")
	}
	db, err := openCatalogDB(ctx, connector, c.Secrets)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	session, err := config.Session(ctx, db)
	if err != nil {
		return nil, err
	}
	schema := input.Schema
	if schema == "" {
		schema = session.Schema
	}
	columns, err := config.Columns(ctx, session, db, input.Table)
	if err != nil {
		return nil, err
	}
	var primary, foreign []sink.Key
	_ = metadata.New().Info(ctx, db, info.KindPrimaryKeys, &primary, option.NewArgs(session.Catalog, schema, input.Table))
	_ = metadata.New().Info(ctx, db, info.KindForeignKeys, &foreign, option.NewArgs(session.Catalog, schema, input.Table))
	pk := map[string]bool{}
	for _, key := range primary {
		pk[strings.ToLower(key.Column)] = true
	}
	fk := map[string]sink.Key{}
	for _, key := range foreign {
		fk[strings.ToLower(key.Column)] = key
	}
	result := &sdk.TableDetail{Table: sdk.DatabaseTable{Catalog: session.Catalog, Schema: schema, Name: input.Table, Type: "TABLE"}}
	for _, column := range columns {
		reference := fk[strings.ToLower(column.Name)]
		result.Columns = append(result.Columns, sdk.DatabaseColumn{Name: column.Name, Type: column.Type, Nullable: column.IsNullable(), PrimaryKey: pk[strings.ToLower(column.Name)] || strings.EqualFold(column.Key, "PRI"), Unique: column.IsUnique(), AutoIncrement: column.Autoincrement(), Default: column.Default, ReferenceSchema: reference.ReferenceSchema, ReferenceTable: reference.ReferenceTable, ReferenceColumn: reference.ReferenceColumn})
	}
	sort.Slice(result.Columns, func(i, j int) bool { return result.Columns[i].Name < result.Columns[j].Name })
	return result, nil
}

func openCatalogDB(ctx context.Context, connector *sdk.Connector, resolver connectorsecret.Resolver) (*sql.DB, error) {
	if connector == nil {
		return nil, fmt.Errorf("connector is required")
	}
	driver := resolveDriver(strings.ToLower(strings.TrimSpace(connector.Driver)))
	dsn, err := connectorsecret.Resolve(ctx, resolver, connector.DSNTemplate, connector.SecretRef)
	if err != nil {
		return nil, err
	}
	if dsn == "" {
		return nil, fmt.Errorf("connector %q has no DSN", connector.Name)
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	if err = connectorinit.Configure(ctx, db, driver, connector.Options); err != nil {
		db.Close()
		return nil, err
	}
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
