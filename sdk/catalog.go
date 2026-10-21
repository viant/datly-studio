package sdk

import (
	"context"
	"encoding/json"
	"time"
)

const (
	OperationConnectorDiscover = "connectors.discover"
	OperationConnectorSchemas  = "connectors.schemas"
	OperationConnectorTables   = "connectors.tables"
	OperationConnectorTable    = "connectors.table"
	OperationConnectorTestSQL  = "connectors.test_sql"
)

type SchemaCatalogInput struct {
	Catalog string `json:"catalog,omitempty"`
	Query   string `json:"query,omitempty"`
}
type TableCatalogInput struct {
	Catalog string `json:"catalog,omitempty"`
	Schema  string `json:"schema,omitempty"`
	Query   string `json:"query,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
}
type TableDetailInput struct {
	Catalog string `json:"catalog,omitempty"`
	Schema  string `json:"schema,omitempty"`
	Table   string `json:"table"`
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
	Discover(context.Context, string, DatabaseDiscoveryInput) (*DatabaseDiscovery, error)
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

// DatabaseDiscoveryInput selects metadata scope without changing the connector.
type DatabaseDiscoveryInput struct {
	Catalog string `json:"catalog,omitempty"`
	Schema  string `json:"schema,omitempty"`
}
type DatabaseCatalog struct {
	Name string `json:"name"`
}
type DatabaseDiscovery struct {
	Driver   string            `json:"driver"`
	Product  string            `json:"product"`
	Version  string            `json:"version,omitempty"`
	Catalog  string            `json:"catalog,omitempty"`
	Schema   string            `json:"schema,omitempty"`
	Catalogs []DatabaseCatalog `json:"catalogs"`
	Schemas  []DatabaseSchema  `json:"schemas"`
	Warnings []string          `json:"warnings,omitempty"`
}
type DatabaseDiscoverer interface {
	Discover(context.Context, *Connector, DatabaseDiscoveryInput) (*DatabaseDiscovery, error)
}

func (c catalogClient) Discover(ctx context.Context, name string, input DatabaseDiscoveryInput) (*DatabaseDiscovery, error) {
	return invoke[DatabaseDiscovery](ctx, c.transport, OperationConnectorDiscover, struct {
		Name  string                 `json:"name"`
		Input DatabaseDiscoveryInput `json:"input"`
	}{name, input})
}
