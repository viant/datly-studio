package sdk

import (
	"context"
	"encoding/json"
	"time"
)

const (
	OperationConnectorSchemas = "connectors.schemas"
	OperationConnectorTables  = "connectors.tables"
	OperationConnectorTable   = "connectors.table"
	OperationConnectorTestSQL = "connectors.test_sql"
)

type SchemaCatalogInput struct {
	Query string `json:"query,omitempty"`
}
type TableCatalogInput struct {
	Schema string `json:"schema,omitempty"`
	Query  string `json:"query,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}
type TableDetailInput struct {
	Schema string `json:"schema,omitempty"`
	Table  string `json:"table"`
}
type SQLTestInput struct {
	SQL   string `json:"sql"`
	Limit int    `json:"limit,omitempty"`
}
type SQLTestResult struct {
	Data     json.RawMessage `json:"data,omitempty"`
	Duration time.Duration   `json:"duration"`
}

type DatabaseSchema struct {
	Catalog string `json:"catalog,omitempty"`
	Name    string `json:"name"`
}
type DatabaseTable struct {
	Catalog string `json:"catalog,omitempty"`
	Schema  string `json:"schema,omitempty"`
	Name    string `json:"name"`
	Type    string `json:"type,omitempty"`
	Comment string `json:"comment,omitempty"`
}
type DatabaseColumn struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	Nullable        bool    `json:"nullable"`
	PrimaryKey      bool    `json:"primaryKey"`
	Unique          bool    `json:"unique"`
	AutoIncrement   bool    `json:"autoIncrement"`
	Default         *string `json:"default,omitempty"`
	ReferenceSchema string  `json:"referenceSchema,omitempty"`
	ReferenceTable  string  `json:"referenceTable,omitempty"`
	ReferenceColumn string  `json:"referenceColumn,omitempty"`
}
type SchemaCatalog struct {
	Items []DatabaseSchema `json:"items"`
}
type TableCatalog struct {
	Items  []DatabaseTable `json:"items"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}
type TableDetail struct {
	Table   DatabaseTable    `json:"table"`
	Columns []DatabaseColumn `json:"columns"`
}

type CatalogExplorer interface {
	Schemas(context.Context, *Connector, SchemaCatalogInput) (*SchemaCatalog, error)
	Tables(context.Context, *Connector, TableCatalogInput) (*TableCatalog, error)
	Table(context.Context, *Connector, TableDetailInput) (*TableDetail, error)
}

// ConnectorSQLTester compiles SQL as a transient Datly reader. Implementations
// must not execute browser-authored SQL through a parallel raw-row path.
type ConnectorSQLTester interface {
	TestSQL(context.Context, *Connector, SQLTestInput) (*SQLTestResult, error)
}

type CatalogService interface {
	Schemas(context.Context, string, SchemaCatalogInput) (*SchemaCatalog, error)
	Tables(context.Context, string, TableCatalogInput) (*TableCatalog, error)
	Table(context.Context, string, TableDetailInput) (*TableDetail, error)
	TestSQL(context.Context, string, SQLTestInput) (*SQLTestResult, error)
}

type catalogClient struct{ transport Transport }

func (c catalogClient) Schemas(ctx context.Context, name string, input SchemaCatalogInput) (*SchemaCatalog, error) {
	return invoke[SchemaCatalog](ctx, c.transport, OperationConnectorSchemas, struct {
		Name  string             `json:"name"`
		Input SchemaCatalogInput `json:"input"`
	}{name, input})
}
func (c catalogClient) Tables(ctx context.Context, name string, input TableCatalogInput) (*TableCatalog, error) {
	return invoke[TableCatalog](ctx, c.transport, OperationConnectorTables, struct {
		Name  string            `json:"name"`
		Input TableCatalogInput `json:"input"`
	}{name, input})
}
func (c catalogClient) Table(ctx context.Context, name string, input TableDetailInput) (*TableDetail, error) {
	return invoke[TableDetail](ctx, c.transport, OperationConnectorTable, struct {
		Name  string           `json:"name"`
		Input TableDetailInput `json:"input"`
	}{name, input})
}
func (c catalogClient) TestSQL(ctx context.Context, name string, input SQLTestInput) (*SQLTestResult, error) {
	return invoke[SQLTestResult](ctx, c.transport, OperationConnectorTestSQL, struct {
		Name  string       `json:"name"`
		Input SQLTestInput `json:"input"`
	}{name, input})
}
