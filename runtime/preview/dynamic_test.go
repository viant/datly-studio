package preview

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
)

func TestDynamicExecutesVersionedReaderWithRuntimeContracts(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app\n\ngo 1.25.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sourceDSN := "file:" + filepath.Join(root, "vendor.db") + "?cache=shared"
	source, err := sql.Open("sqlite", sourceDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if _, err = source.Exec(`CREATE TABLE VENDOR (ID INTEGER PRIMARY KEY, NAME TEXT NOT NULL); INSERT INTO VENDOR(ID, NAME) VALUES (1, 'Northwind'),(2,'Contoso'),(3,'Fabrikam'); CREATE TABLE PRODUCT (ID INTEGER PRIMARY KEY, VENDOR_ID INTEGER NOT NULL, STATUS INTEGER NOT NULL); INSERT INTO PRODUCT(ID, VENDOR_ID, STATUS) VALUES (1,1,1),(2,1,1),(3,2,2);`); err != nil {
		t.Fatal(err)
	}
	studio, err := sql.Open("sqlite", "file:preview-studio?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer studio.Close()
	if err = schema.ApplySQLite(ctx, studio, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err = studio.ExecContext(ctx, `INSERT INTO connectors(name,driver,dsn_template,owner_id,status,options_json,etag,created_at,updated_at) VALUES ('vendor','sqlite',?,'owner','active','{}',1,?,?)`, sourceDSN, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err = studio.ExecContext(ctx, `INSERT INTO reports(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES ('vendors','vendors','Vendors','owner','draft','vendor','example.com/app/dynamic/vendors','reader',1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	dql := fmt.Sprintf(`#package('example.com/app/dynamic/vendors')
#setting($_ = $connector('vendor'))
#setting($_ = $route('/vendors','GET'))
#setting($_ = $mcp('owner.vendors.read', 'Read vendors', 'docs/vendors.md'))
#define($_ = $Tenant<int>(query/tenant).Optional())
#define($_ = $Vendors<[]*Vendor>(output/view))
#setting($_ = $cache('vendors', '1m').WithLocation('%s'))
#setting($_ = $cache_warmup('ID', 'Tenant=1,2'))
SELECT vendors.*, products.* EXCEPT VENDOR_ID, type(vendors, 'Vendor'), type(products, 'Product')
FROM (SELECT ID, NAME FROM VENDOR) vendors
JOIN (SELECT ID, VENDOR_ID, STATUS FROM PRODUCT) products ON products.VENDOR_ID=vendors.ID`, filepath.Join(root, "cache"))
	if _, err = studio.ExecContext(ctx, `INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES ('vendors',1,'draft','dql',?,?, '{}','studio.v1','hash','{}','pending','v1','studio.v1',1,'owner',?)`, dql, dql, now); err != nil {
		t.Fatal(err)
	}
	if _, err = studio.ExecContext(ctx, `INSERT INTO report_resource_files(report_id,version_no,resource_id,namespace,resource_path,content,content_size,content_sha256,is_binary,created_at) VALUES ('vendors',1,'vendors-doc','docs','docs/vendors.md',?,?, 'hash',FALSE,?)`, "Vendor API documentation", len("Vendor API documentation"), now); err != nil {
		t.Fatal(err)
	}
	result, err := (Dynamic{StudioDB: studio, RootDir: root}).Execute(ctx, "vendors", 1, sdk.PreviewInput{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result.Data), "Northwind") || strings.Contains(string(result.Data), "Contoso") || result.Duration <= 0 || result.Evidence.ReportID != "vendors" || result.Evidence.VersionNo != 1 || result.Evidence.SourceRevision != 1 || result.Evidence.Limit != 1 || result.Evidence.ReturnedRows != 1 || !result.Evidence.Truncated || result.Evidence.EncodedBytes != len(result.Data) {
		t.Fatalf("preview=%+v", result)
	}
	if err = (Dynamic{StudioDB: studio, RootDir: root}).Validate(ctx, "vendors", 1); err != nil {
		t.Fatalf("runtime validation: %v", err)
	}
	warmup, err := (Dynamic{StudioDB: studio, RootDir: root}).Warmup(ctx, "vendors", 1)
	if err != nil {
		t.Fatal(err)
	}
	if warmup.Entries <= 0 || warmup.Duration <= 0 || warmup.PlannedCases != 3 || warmup.CompletedCases != 3 {
		t.Fatalf("warmup=%+v", warmup)
	}
	view, err := (Dynamic{StudioDB: studio, RootDir: root}).TestView(ctx, "vendors", 1, "vendors", sdk.ViewTestInput{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if view.View != "vendors" || !strings.Contains(string(view.Data), "Northwind") || view.Duration <= 0 || view.Evidence.ReturnedRows != 1 || !view.Evidence.Truncated {
		t.Fatalf("view test=%+v", view)
	}
	relation, err := (Dynamic{StudioDB: studio, RootDir: root}).TestRelation(ctx, "vendors", 1, "products", sdk.ViewTestInput{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if relation.Relation != "products" || relation.ParentView != "vendors" || relation.ChildView != "products" || relation.ParentRows != 3 || relation.MatchedParents != 2 || relation.UnmatchedParents != 1 || relation.AttachedChildren != 3 || len(relation.Keys) != 1 || relation.Keys[0].ParentColumn != "ID" || relation.Keys[0].ChildColumn != "VENDOR_ID" || relation.Evidence.VersionNo != 1 {
		t.Fatalf("relation test=%+v", relation)
	}
	sqlTest, err := (Dynamic{StudioDB: studio, RootDir: root}).TestSQL(ctx, &sdk.Connector{Name: "vendor", Driver: "sqlite", DSNTemplate: sourceDSN}, sdk.SQLTestInput{SQL: "SELECT ID, NAME FROM VENDOR", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sqlTest.Data), "Northwind") || sqlTest.Duration <= 0 {
		t.Fatalf("SQL test=%+v", sqlTest)
	}
	if _, err = studio.ExecContext(ctx, `INSERT INTO reports(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES ('summary','summary','Summary','owner','draft','vendor','example.com/app/dynamic/summary','reader',1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	composeDQL := `#package('example.com/app/dynamic/summary/reader')
#setting($_ = $connector('vendor'))
#setting($_ = $route('/summary','GET'))
#setting($_ = $cube())
#setting($_ = $cubeCompose(true, false, 4, 20, 5000))
#define($_ = $Rows<[]*Summary>(output/view))
SELECT summary.*, groupable(summary), tag(summary.status, 'groupable:"true"'), CAST(summary.product_count AS float64), type(summary, 'Summary')
FROM (SELECT STATUS AS status, COUNT(*) AS product_count FROM PRODUCT GROUP BY STATUS) summary`
	if _, err = studio.ExecContext(ctx, `INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES ('summary',1,'draft','dql',?,?, '{}','studio.v1','hash','{}','pending','v1','studio.v1',1,'owner',?)`, composeDQL, composeDQL, now); err != nil {
		t.Fatal(err)
	}
	compose, err := (Dynamic{StudioDB: studio, RootDir: root}).TestCompose(ctx, "summary", 1, sdk.CubeComposeTestInput{
		Cubes: []json.RawMessage{json.RawMessage(`{"dimensions":{"status":true},"measures":{"productCount":true},"filters":{}}`)},
		SQL:   "SELECT t1.status, t1.product_count FROM $CubeSQL1 AS t1 ORDER BY t1.status",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(compose.Data), `"product_count":2`) || compose.Duration <= 0 {
		t.Fatalf("compose=%+v", compose)
	}
}

func TestOpenSourcesRegistersExactNamedStudioConnectors(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	defaultDSN := "file:" + filepath.Join(root, "default.db")
	lookupDSN := "file:" + filepath.Join(root, "lookup.db")
	definition := &definition{
		Scope: "example.com/app/dynamic/catalog", Name: "reader",
		Connector: "primary", Driver: "sqlite", DSN: "${Primary}", SecretRef: "secret://primary",
		Connectors: []connectorDefinition{
			{Name: "primary", Driver: "sqlite", DSN: "${Primary}", SecretRef: "secret://primary"},
			{Name: "lookup", Driver: "sqlite", DSN: "${Lookup}", SecretRef: "secret://lookup"},
		},
	}
	definition.Secrets = connectorsecret.ResolverFunc(func(_ context.Context, template, reference string) (string, error) {
		switch reference {
		case "secret://primary":
			return defaultDSN, nil
		case "secret://lookup":
			return lookupDSN, nil
		default:
			return "", fmt.Errorf("unexpected secret reference %s for %s", reference, template)
		}
	})
	dql := `#package('example.com/app/dynamic/catalog')
#setting($_ = $connector('primary'))
#setting($_ = $route('/catalog','GET'))
#define($_ = $Rows<[]*Row>(output/view))
SELECT rows.*, use_connector(rows, 'lookup'), type(rows, 'Row') FROM (SELECT 1 AS id) rows`
	sources, err := openSources(ctx, definition, dql)
	if err != nil {
		t.Fatal(err)
	}
	defer sources.Close()
	if len(sources.Connections) != 2 {
		t.Fatalf("connections=%v", sources.Connections)
	}
	resolved, err := sources.SQL.Resolve(ctx, "lookup")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.DB != sources.Connections["lookup"] {
		t.Fatal("lookup connector did not resolve to its named Studio database")
	}

	_, err = openSources(ctx, definition, strings.ReplaceAll(dql, "'lookup'", "'missing'"))
	if err == nil || !strings.Contains(err.Error(), `requires active Studio connector "missing"`) {
		t.Fatalf("missing connector error=%v", err)
	}
}
