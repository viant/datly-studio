package preview

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
)

func TestViewDQLUsesVersionResourcesAndSelectedConnector(t *testing.T) {
	store, err := resource.New().WithDefault(fstest.MapFS{"sql/root.sql": {Data: []byte("SELECT ID FROM events")}, "sql/channel.sql": {Data: []byte("SELECT ID,NAME FROM channels")}})
	if err != nil {
		t.Fatal(err)
	}
	definition := &definition{Scope: "example.com/fixture", Name: "reader", Connector: "main", Connectors: []connectorDefinition{{Name: "main"}, {Name: "lookup"}}, Resources: store, DQL: `#package('example.com/fixture')
#setting($_ = $connector('main'))
#setting($_ = $route('/fixture','GET'))
#define($_ = $Rows<[]*Row>(output/view))
SELECT root.*,channel.*,type(root,'Row'),type(channel,'Channel'),use_connector(channel,'lookup')
FROM (${embed:sql/root.sql}) root
JOIN (${embed:sql/channel.sql}) channel ON root.ID=channel.ID AND 1=1`}
	source, err := testViewDQL(context.Background(), definition, "channel", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source, "$connector('lookup')") || !strings.Contains(source, "${embed:sql/channel.sql}") || strings.Contains(source, "JOIN") {
		t.Fatalf("incorrect view test source: %s", source)
	}
	connectors := requiredConnectors(context.Background(), definition, source, []string{"main", "lookup"})
	if len(connectors) != 2 || connectors[0] != "lookup" || connectors[1] != "main" {
		t.Fatalf("embedded view connectors=%v", connectors)
	}
}

func TestViewDQLReturnsErrorForMissingComponentAuthority(t *testing.T) {
	_, err := testViewDQL(context.Background(), &definition{Scope: "example.com/fixture", Name: "reader", Connector: "main", DQL: "SELECT * FROM (${embed:missing.sql}) root"}, "root", nil)
	if err == nil {
		t.Fatal("missing component authority was accepted")
	}
}

func TestDynamicTestViewPreservesSelectors(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	dsn := "file:" + filepath.Join(root, "source.db")
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err = db.ExecContext(ctx, `CREATE TABLE records(id INTEGER PRIMARY KEY,name TEXT);
INSERT INTO records VALUES(1,'first'),(2,'second'),(3,'third')`); err != nil {
		t.Fatal(err)
	}
	studio, err := sql.Open("sqlite", "file:"+filepath.Join(root, "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = studio.Close() })
	if err = schema.ApplySQLite(ctx, studio, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err = studio.ExecContext(ctx, `INSERT INTO connectors(name,driver,dsn_template,owner_id,status,options_json,etag,created_at,updated_at)
VALUES('main','sqlite',?,'owner','active','{}',1,?,?)`, dsn, now, now); err != nil {
		t.Fatal(err)
	}
	for _, alias := range []string{"entries", "MetricRows"} {
		t.Run(alias, func(t *testing.T) {
			scope := "example.com/fixture/" + strings.ToLower(alias)
			source := fmt.Sprintf(`#package('%s')
#setting($_ = $connector('main'))
#setting($_ = $route('/records','GET'))
#define($_ = $MinimumId<int>(query/minimumId).Required().WithPredicate(0,'greater_or_equal','r','id'))
#define($_ = $Fields<[]string>(query/_fields).Optional().QuerySelector('%[2]s'))
#define($_ = $OrderBy<[]string>(query/_orderBy).Optional().QuerySelector('%[2]s'))
#define($_ = $Limit<int>(query/_limit).Optional().QuerySelector('%[2]s'))
#define($_ = $Offset<int>(query/_offset).Optional().QuerySelector('%[2]s'))
#define($_ = $Rows<[]*Record>(output/view))
SELECT %[2]s.*,type(%[2]s,'Record')
FROM (SELECT r.id,r.name FROM records r
${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND")).Build("WHERE")}) %[2]s`, scope, alias)
			if _, err = studio.ExecContext(ctx, `INSERT INTO components(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at)
VALUES(?,?,?,'owner','draft','main',?,'reader',1,?,?)`, alias, alias, alias, scope, now, now); err != nil {
				t.Fatal(err)
			}
			if _, err = studio.ExecContext(ctx, `INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at)
VALUES(?,1,'draft','dql',?,?,'{}','studio.v1','hash','{}','pending','v1','studio.v1',1,'owner',?)`, alias, source, source, now); err != nil {
				t.Fatal(err)
			}
			engine := Dynamic{StudioDB: studio, ModulePath: "example.com/fixture"}
			input := json.RawMessage(`{"minimumId":2,"fields":["name"],"orderBy":["id DESC"],"limit":1,"offset":1}`)
			baseline, err := engine.Execute(ctx, alias, 1, sdk.PreviewInput{Input: input, Limit: 10})
			if err != nil {
				t.Fatal(err)
			}
			// Lookup is case insensitive; reconstruction must retain authored casing.
			result, err := engine.TestView(ctx, alias, 1, strings.ToLower(alias), sdk.ViewTestInput{Input: input, Limit: 10})
			if err != nil {
				t.Fatal(err)
			}
			var rows struct {
				Rows []struct {
					Name string
					ID   *int
				}
			}
			if err = json.Unmarshal(result.Data, &rows); err != nil {
				t.Fatal(err)
			}
			if len(rows.Rows) != 1 || rows.Rows[0].ID != nil || rows.Rows[0].Name != "second" {
				t.Fatalf("selection, sorting, paging or required filter changed: %s", result.Data)
			}
			if string(result.Data) != string(baseline.Data) {
				t.Fatalf("view=%s baseline=%s", result.Data, baseline.Data)
			}
			if result.Evidence.ReportID != alias || result.Evidence.VersionNo != 1 || result.Evidence.ReturnedRows != 1 {
				t.Fatalf("wrong evidence: %+v", result.Evidence)
			}
			if _, err = engine.TestView(ctx, alias, 1, alias, sdk.ViewTestInput{Input: json.RawMessage(`{"fields":["name"]}`)}); err == nil {
				t.Fatal("isolated view dropped required input validation")
			}
		})
	}
}
