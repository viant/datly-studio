package connectivity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/viant/datly/constant"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/viant/datly-studio/internal/connectorinit"
	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly-studio/internal/providererror"
	"github.com/viant/datly-studio/sdk"
	sqlxio "github.com/viant/sqlx/io"
	"github.com/viant/sqlx/io/config"
	"github.com/viant/sqlx/metadata"
	"github.com/viant/sqlx/metadata/info"
	bqmeta "github.com/viant/sqlx/metadata/product/bigquery"
	"github.com/viant/sqlx/metadata/sink"
	"github.com/viant/sqlx/option"
)

type SQLCatalog struct {
	Secrets connectorsecret.Resolver
	Const   *constant.Values
}

func (c SQLCatalog) Schemas(ctx context.Context, connector *sdk.Connector, input sdk.SchemaCatalogInput) (*sdk.SchemaCatalog, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	scope, err := resolveDiscoveryScope(connector, input.Catalog, "")
	if err != nil {
		return nil, err
	}
	db, err := openDiscoveryDB(ctx, connector, c.Secrets, c.Const, scope)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	session, err := config.Session(ctx, db)
	if err != nil {
		return nil, err
	}
	scope.apply(session)
	var schemas []sink.Schema
	err = metadata.New().Info(ctx, db, info.KindSchemas, &schemas, option.NewArgs(session.Catalog))
	if err != nil {
		if scope.Catalog != "" {
			return nil, err
		}
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
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	limit := input.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if connector == nil {
		return nil, fmt.Errorf("connector is required")
	}
	// The metadata provider owns cursor paging. Do not emulate offsets by
	// fetching and discarding potentially unbounded pages.
	paged := resolveDriver(strings.ToLower(strings.TrimSpace(connector.Driver))) == "bigquery"
	if paged && input.Offset != 0 {
		return nil, fmt.Errorf("table metadata uses pageToken; omit offset and continue with nextPageToken")
	}
	if !paged && input.PageToken != "" {
		return nil, fmt.Errorf("this connector does not support table metadata pageToken; use offset")
	}
	scope, err := resolveDiscoveryScope(connector, input.Catalog, input.Schema)
	if err != nil {
		return nil, err
	}
	if paged && (scope.Catalog == "" || scope.Schema == "") {
		return nil, fmt.Errorf("table listing requires explicit catalog and schema in the request or connector discovery settings")
	}
	db, err := openDiscoveryDB(ctx, connector, c.Secrets, c.Const, scope)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	session := &sink.Session{}
	if !paged {
		session, err = config.Session(ctx, db)
		if err != nil {
			return nil, err
		}
	}
	scope.apply(session)
	schema := session.Schema
	var tables []sink.Table
	options := []option.Option{option.NewArgs(session.Catalog, schema)}
	page := &option.MetadataPage{Limit: int64(limit), Token: input.PageToken}
	if paged {
		options = append(options, page, bqmeta.BigQuery())
	}
	if err = metadata.New().Info(ctx, db, info.KindTables, &tables, options...); err != nil {
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
		if item.Catalog == "" {
			item.Catalog = session.Catalog
		}
		if item.Schema == "" {
			item.Schema = schema
		}
		items = append(items, sdk.DatabaseTable{Catalog: item.Catalog, Schema: item.Schema, Name: item.Name, Type: kind, Comment: comment})
	}
	if paged {
		if !page.Handled {
			return nil, fmt.Errorf("connector metadata provider did not handle pagination")
		}
		return &sdk.TableCatalog{Items: items, Limit: limit, NextPageToken: page.NextToken}, nil
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
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
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if strings.TrimSpace(input.Table) == "" {
		return nil, fmt.Errorf("table is required")
	}
	scope, err := resolveDiscoveryScope(connector, input.Catalog, input.Schema)
	if err != nil {
		return nil, err
	}
	db, err := openDiscoveryDB(ctx, connector, c.Secrets, c.Const, scope)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	session := &sink.Session{}
	apiMetadata := resolveDriver(strings.ToLower(strings.TrimSpace(connector.Driver))) == "bigquery" && scope.Catalog != "" && scope.Schema != ""
	var metadataOptions []option.Option
	if apiMetadata {
		metadataOptions = append(metadataOptions, bqmeta.BigQuery())
	} else {
		session, err = config.Session(ctx, db)
		if err != nil {
			return nil, err
		}
	}
	scope.apply(session)
	schema := session.Schema
	columns, err := config.Columns(ctx, session, db, input.Table, metadataOptions...)
	if err == nil && len(columns) == 0 {
		err = fmt.Errorf("column metadata returned no columns; verify the exact table and schema or metadata access")
	}
	metadataSource := "catalog"
	var diagnostics []sdk.Diagnostic
	if err != nil {
		primary := err
		columns, err = probeTableColumns(ctx, db, session, input.Table, primary)
		if err != nil {
			return nil, err
		}
		metadataSource = "zero_row_probe"
		diagnostics = []sdk.Diagnostic{{Severity: "warning", Stage: "catalog.columns", Code: "column_metadata_fallback", Message: providererror.AppendCatalogStatus("Primary column metadata discovery failed; a zero-row probe supplied the column metadata.", primary), Hint: "Verify the exact source catalog and schema and the configured connector metadata access. Constraint information may be unknown; inspect constraintsKnown and nullableKnown before using it."}}
	}
	var primary, foreign []sink.Key
	constraintsKnown := false
	if !apiMetadata {
		primaryErr := metadata.New().Info(ctx, db, info.KindPrimaryKeys, &primary, option.NewArgs(session.Catalog, schema, input.Table))
		foreignErr := metadata.New().Info(ctx, db, info.KindForeignKeys, &foreign, option.NewArgs(session.Catalog, schema, input.Table))
		constraintsKnown = primaryErr == nil && foreignErr == nil
	}
	// The single-table schema adapter does not expose key constraints. Do not
	// issue synthetic empty SQL queries or report unknown constraints as known.
	pk := map[string]bool{}
	for _, key := range primary {
		pk[strings.ToLower(key.Column)] = true
	}
	fk := map[string]sink.Key{}
	for _, key := range foreign {
		fk[strings.ToLower(key.Column)] = key
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result := &sdk.TableDetail{Diagnostics: diagnostics, MetadataSource: metadataSource, ConstraintsKnown: metadataSource == "catalog" && constraintsKnown, Table: sdk.DatabaseTable{Catalog: session.Catalog, Schema: schema, Name: input.Table, Type: "TABLE"}}
	for _, column := range columns {
		reference := fk[strings.ToLower(column.Name)]
		nullableKnown := column.IsNullable() || column.IsNotNull()
		result.Columns = append(result.Columns, sdk.DatabaseColumn{Description: column.Comments, NullableKnown: &nullableKnown, Name: column.Name, Type: column.Type, Nullable: column.IsNullable(), PrimaryKey: pk[strings.ToLower(column.Name)] || strings.EqualFold(column.Key, "PRI"), Unique: column.IsUnique(), AutoIncrement: column.Autoincrement(), Default: column.Default, ReferenceSchema: reference.ReferenceSchema, ReferenceTable: reference.ReferenceTable, ReferenceColumn: reference.ReferenceColumn})
	}
	sort.Slice(result.Columns, func(i, j int) bool { return result.Columns[i].Name < result.Columns[j].Name })
	return result, nil
}

func openCatalogDB(ctx context.Context, connector *sdk.Connector, resolver connectorsecret.Resolver, values ...*constant.Values) (*sql.DB, error) {
	return openScopedCatalogDB(ctx, connector, resolver, discoveryScope{}, values...)
}

func openDiscoveryDB(ctx context.Context, connector *sdk.Connector, resolver connectorsecret.Resolver, constants *constant.Values, scope discoveryScope) (*sql.DB, error) {
	return openScopedCatalogDB(ctx, connector, resolver, scope, constants)
}

func openScopedCatalogDB(ctx context.Context, connector *sdk.Connector, resolver connectorsecret.Resolver, scope discoveryScope, values ...*constant.Values) (*sql.DB, error) {
	if connector == nil {
		return nil, fmt.Errorf("connector is required")
	}
	driver := resolveDriver(strings.ToLower(strings.TrimSpace(connector.Driver)))
	dsn, err := connectorsecret.Resolve(ctx, resolver, connector.DSNTemplate, connector.SecretRef, values...)
	if err != nil {
		return nil, err
	}
	if dsn == "" {
		return nil, fmt.Errorf("connector %q has no DSN", connector.Name)
	}
	// SQLX's SQLite metadata queries target main. Open the configured attachment
	// as the discovery connection rather than silently describing main's tables.
	if strings.Contains(driver, "sqlite") && scope.Schema != "" && scope.Schema != "main" {
		var configured struct {
			Attachments map[string]string `json:"sqliteAttachments"`
		}
		if json.Unmarshal(connector.Options, &configured) != nil || configured.Attachments[scope.Schema] == "" {
			return nil, fmt.Errorf("selected SQLite schema is not a configured attachment")
		}
		dsn = configured.Attachments[scope.Schema]
	}
	dsn, err = discoveryDSN(driver, dsn, scope)
	if err != nil {
		return nil, err
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

// Discovery defaults are ordinary connector options, not schema or driver locks.
// Per-request values override them without modifying stored connection material.
type discoveryScope struct {
	Catalog string `json:"catalog"`
	Schema  string `json:"schema"`
}

func resolveDiscoveryScope(connector *sdk.Connector, catalog, schema string) (discoveryScope, error) {
	var options struct {
		Discovery discoveryScope `json:"discovery"`
	}
	if connector == nil {
		return discoveryScope{}, fmt.Errorf("connector is required")
	}
	if len(connector.Options) > 0 && json.Unmarshal(connector.Options, &options) != nil {
		return discoveryScope{}, fmt.Errorf("invalid connector discovery options")
	}
	scope := options.Discovery
	if strings.TrimSpace(catalog) != "" {
		scope.Catalog = catalog
	}
	if strings.TrimSpace(schema) != "" {
		scope.Schema = schema
	}
	scope.Catalog = strings.TrimSpace(scope.Catalog)
	scope.Schema = strings.TrimSpace(scope.Schema)
	// SQLX metadata templates interpolate these identifiers for some products.
	for _, name := range []string{scope.Catalog, scope.Schema} {
		for _, char := range name {
			if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '_' && char != '-' && char != '$' {
				return discoveryScope{}, fmt.Errorf("project/catalog and schema names may contain letters, numbers, underscores, hyphens and dollar signs")
			}
		}
	}
	return scope, nil
}
func (s discoveryScope) apply(session *sink.Session) {
	if s.Catalog != "" {
		session.Catalog = s.Catalog
	}
	if s.Schema != "" {
		session.Schema = s.Schema
	}
}
func discoveryDSN(driver, dsn string, scope discoveryScope) (string, error) {
	if driver != "bigquery" || scope.Catalog == "" && scope.Schema == "" {
		return dsn, nil
	}
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Scheme != "bigquery" {
		return "", fmt.Errorf("invalid BigQuery discovery connection")
	}
	// Catalog scopes the metadata query through scope.apply(session). The DSN
	// host remains the configured job project, which may differ from the source
	// catalog and have its own jobs.create permission and billing boundary.
	if scope.Schema != "" {
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		if len(parts) > 1 {
			parsed.Path = "/" + parts[0] + "/" + scope.Schema
		} else {
			parsed.Path = "/" + scope.Schema
		}
		parsed.RawPath = ""
	}
	return parsed.String(), nil
}

// probeTableColumns reuses SQLX's metadata-only detector on the already opened
// authorized connection. No rows are consumed and no connection scope changes.
func probeTableColumns(ctx context.Context, db *sql.DB, session *sink.Session, table string, primary error) ([]sink.Column, error) {
	if err := ctx.Err(); err != nil {
		return nil, &ColumnDiscoveryError{Primary: primary, Probe: err, Phase: "cancellation"}
	}
	if errors.Is(primary, context.Canceled) || errors.Is(primary, context.DeadlineExceeded) {
		return nil, primary
	}
	dialect, err := config.Dialect(ctx, db)
	if err != nil {
		return nil, &ColumnDiscoveryError{Primary: primary, Probe: err, Phase: "dialect"}
	}
	name, err := dialect.QuoteIdentifier(table)
	if err == nil {
		schema := session.Schema
		// Attachment discovery opens the attachment itself as main.
		if strings.EqualFold(dialect.Name, "sqlite") {
			schema = "main"
		}
		if strings.EqualFold(dialect.Name, "bigquery") {
			if session.Catalog == "" || schema == "" {
				err = fmt.Errorf("exact catalog and schema are required for BigQuery column discovery")
			} else {
				name, err = dialect.QuoteIdentifier(session.Catalog + "." + schema + "." + table)
			}
		} else if schema != "" {
			var quoted string
			quoted, err = dialect.QuoteIdentifier(schema)
			name = quoted + "." + name
		}
	}
	if err != nil {
		return nil, &ColumnDiscoveryError{Primary: primary, Probe: err, Phase: "identifier"}
	}
	nullableKnown := map[string]bool{}
	detector := sqlxio.ColumnDetector{ResolveTypes: func(columns []sqlxio.Column) map[int]reflect.Type {
		for _, column := range columns {
			_, known := column.Nullable()
			nullableKnown[column.Name()] = known
		}
		return nil
	}}
	detected, err := detector.Detect(ctx, db, "SELECT * FROM "+name+" WHERE 1=0")
	if err != nil {
		return nil, &ColumnDiscoveryError{Primary: primary, Probe: err, Phase: "zero-row probe"}
	}
	result := make([]sink.Column, 0, len(detected))
	for _, column := range detected {
		if !nullableKnown[column.Name] {
			column.Nullable = ""
		} else if column.Nullable == "" {
			column.Nullable = "0"
		}
		result = append(result, *column)
	}
	return result, nil
}

// ColumnDiscoveryError preserves the primary metadata failure separately from
// recovery failure. Consumers classify Primary; Unwrap retains both identities.
type ColumnDiscoveryError struct {
	Primary error  `json:"-"`
	Probe   error  `json:"-"`
	Phase   string `json:"phase"`
}

func (e *ColumnDiscoveryError) Error() string {
	return fmt.Sprintf("table column discovery: %v; %s failed: %v", e.Primary, e.Phase, e.Probe)
}
func (e *ColumnDiscoveryError) Unwrap() []error { return []error{e.Primary, e.Probe} }
