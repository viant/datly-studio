package connectivity

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
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

func TestDiscoveryScopeSelectsAttachedSchemaForTablesAndColumns(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	mainDSN := "file:" + filepath.Join(root, "main.db")
	auxDSN := "file:" + filepath.Join(root, "aux.db")
	for _, fixture := range []struct{ dsn, ddl string }{{mainDSN, `CREATE TABLE same_name(main_only TEXT);`}, {auxDSN, `CREATE TABLE same_name(aux_only INTEGER); CREATE TABLE aux_table(id INTEGER);`}} {
		db, err := sql.Open("sqlite", fixture.dsn)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(fixture.ddl); err != nil {
			t.Fatal(err)
		}
		db.Close()
	}
	options, _ := json.Marshal(map[string]any{"sqliteAttachments": map[string]string{"aux": auxDSN}})
	connector := &sdk.Connector{Name: "source", Driver: "sqlite", DSNTemplate: mainDSN, Options: options}
	catalog := SQLCatalog{}
	discovery, err := catalog.Discover(ctx, connector, sdk.DatabaseDiscoveryInput{})
	if err != nil || discovery.Product == "" || discovery.Version == "" {
		t.Fatalf("discovery=%+v err=%v", discovery, err)
	}
	detail, err := catalog.Table(ctx, connector, sdk.TableDetailInput{Schema: "aux", Table: "same_name"})
	if err != nil || len(detail.Columns) != 1 || detail.Columns[0].Name != "aux_only" {
		t.Fatalf("selected schema columns=%+v err=%v", detail, err)
	}
	tables, err := catalog.Tables(ctx, connector, sdk.TableCatalogInput{Schema: "aux"})
	if err != nil || len(tables.Items) != 2 {
		t.Fatalf("selected schema tables=%+v err=%v", tables, err)
	}
	if tables.Items[0].Schema != "aux" {
		t.Fatalf("lost selected schema: %+v", tables.Items)
	}
}

func TestBigQueryDiscoveryOverridesScopeWithoutChangingCredentials(t *testing.T) {
	original := "bigquery://original-project/us/original_dataset?credURL=file%3A%2F%2F%2Fprivate.enc&quotaProject=billing-project"
	got, err := discoveryDSN("bigquery", original, discoveryScope{Catalog: "other-project", Schema: "analytics"})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Host != "other-project" || parsed.Path != "/us/analytics" || parsed.Query().Get("credURL") != "file:///private.enc" || parsed.Query().Get("quotaProject") != "billing-project" {
		t.Fatal("scope override lost connection settings")
	}
	for _, name := range []string{"project` UNION SELECT", "project.schema", "schema;DROP TABLE x"} {
		if _, err := resolveDiscoveryScope(&sdk.Connector{}, name, ""); err == nil {
			t.Errorf("unsafe catalog accepted: %q", name)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (SQLCatalog{}).Discover(ctx, &sdk.Connector{Driver: "sqlite", DSNTemplate: ":memory:"}, sdk.DatabaseDiscoveryInput{}); err == nil {
		t.Fatal("cancelled discovery succeeded")
	}
}
