package connectivity

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly-studio/sdk"
	_ "github.com/viant/sqlx/metadata/product/sqlite"
	_ "modernc.org/sqlite"
)

func TestSQLCatalogDiscoversSQLiteTablesColumnsAndKeys(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "catalog.db")
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`PRAGMA foreign_keys=ON; CREATE TABLE vendors(id INTEGER PRIMARY KEY,name TEXT NOT NULL); CREATE TABLE products(id INTEGER PRIMARY KEY,vendor_id INTEGER NOT NULL,name TEXT,FOREIGN KEY(vendor_id) REFERENCES vendors(id));`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	connector := &sdk.Connector{Name: "catalog", Driver: "sqlite", DSNTemplate: "${Database}", SecretRef: "secret://catalog", Status: "active"}
	catalog := SQLCatalog{Secrets: connectorsecret.ResolverFunc(func(_ context.Context, template, reference string) (string, error) {
		if template != "${Database}" || reference != "secret://catalog" {
			t.Fatalf("template=%q reference=%q", template, reference)
		}
		return dsn, nil
	})}
	tables, err := catalog.Tables(context.Background(), connector, sdk.TableCatalogInput{Query: "prod"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tables.Items) != 1 || tables.Items[0].Name != "products" {
		t.Fatalf("tables=%+v", tables.Items)
	}
	detail, err := catalog.Table(context.Background(), connector, sdk.TableDetailInput{Table: "products"})
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Columns) != 3 {
		t.Fatalf("columns=%+v", detail.Columns)
	}
	byName := map[string]sdk.DatabaseColumn{}
	for _, column := range detail.Columns {
		byName[column.Name] = column
	}
	if !byName["id"].PrimaryKey || byName["vendor_id"].ReferenceTable != "vendors" || byName["vendor_id"].ReferenceColumn != "id" {
		t.Fatalf("columns=%+v", detail.Columns)
	}
}
