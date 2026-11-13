package preview

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
)

// Driver-independent runtime proof for the same two-connector graph used by the
// live BigQuery/MySQL skill acceptance: matching/missing labels and regrouping.
func TestCubeOuterDictionariesPreserveFactsAndSelectedGrain(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	open := func(name string) *sql.DB {
		db, err := sql.Open("sqlite", "file:"+filepath.Join(root, name+".db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		return db
	}
	facts, labels, studio := open("facts"), open("labels"), open("studio")
	if _, err := facts.Exec(`CREATE TABLE fact(event_date TEXT,account_id INTEGER,product_id INTEGER,units INTEGER,amount REAL); INSERT INTO fact VALUES('2026-10-01',10,1,2,20),('2026-10-01',10,1,3,30),('2026-10-01',10,2,7,70)`); err != nil {
		t.Fatal(err)
	}
	if _, err := labels.Exec(`CREATE TABLE product(ID INTEGER PRIMARY KEY,NAME TEXT); INSERT INTO product VALUES(1,'Product One'); CREATE TABLE account(ID INTEGER PRIMARY KEY,NAME TEXT); INSERT INTO account VALUES(10,'Account Ten')`); err != nil {
		t.Fatal(err)
	}
	if err := schema.ApplySQLite(ctx, studio, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, name := range []string{"facts", "labels"} {
		if _, err := studio.Exec(`INSERT INTO connectors(name,driver,dsn_template,owner_id,status,etag,created_at,updated_at) VALUES(?,'sqlite',?,'owner','active',1,?,?)`, name, "file:"+filepath.Join(root, name+".db"), now, now); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := studio.Exec(`INSERT INTO components(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES('dictionary-cube','dictionary-cube','Dictionary cube','owner','draft','facts','example.com/app/dynamic/dictionarycube','reader',1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	dql := `#package('example.com/app/dynamic/dictionarycube')
#setting($_ = $connector('facts'))
#setting($_ = $route('/dictionary-cube','GET'))
#setting($_ = $case_format('lc'))
#setting($_ = $cube())
#define($_ = $Dates<[]string>(query/dates).Required().WithPredicate(0,'in','f','event_date'))
#define($_ = $AccountIds<[]int>(query/accountIds).Required().WithPredicate(0,'in','f','account_id'))
#define($_ = $Rows<[]*FactRow>(output/view))
SELECT fact.*,type(fact,'FactRow'),CAST(fact.units AS int),CAST(fact.amount AS float64),tag(fact.units,'groupable:"false"'),tag(fact.amount,'groupable:"false"'),
 product.*,type(product,'Product'),account.*,type(account,'Account'),
 use_connector(fact,'facts'),use_connector(product,'labels'),use_connector(account,'labels'),
 cardinality(product,'One'),cardinality(account,'One'),allow_nulls(product),allow_nulls(account),
 grouping_enabled(fact),set_limit(fact,50),selector_fields(fact,true),selector_limit(fact,true)
FROM (SELECT f.event_date,f.account_id,f.product_id,SUM(f.units) AS units,SUM(f.amount) AS amount FROM fact f
${predicate.Builder().CombineOr($predicate.FilterGroup(0,"AND")).Build("WHERE")}
GROUP BY f.event_date,f.account_id,f.product_id) fact
LEFT JOIN (SELECT ID AS product_id,NAME AS product_name FROM product) product ON product.product_id=fact.product_id
LEFT JOIN (SELECT ID AS account_id,NAME AS account_name FROM account) account ON account.account_id=fact.account_id`
	if _, err := studio.Exec(`INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES('dictionary-cube',1,'draft','dql',?,?,'{}','studio.v1','hash','{}','pending','v1','studio.v1',1,'owner',?)`, dql, dql, now); err != nil {
		t.Fatal(err)
	}
	execute := func(dimensions map[string]bool) []map[string]any {
		input, _ := json.Marshal(map[string]any{"dimensions": dimensions, "measures": map[string]bool{"units": true, "amount": true}, "filters": map[string]any{"dates": []string{"2026-10-01"}, "accountIds": []int{10}}, "limit": 10})
		result, err := (Dynamic{StudioDB: studio, ModulePath: "example.com/app"}).Execute(ctx, "dictionary-cube", 1, sdk.PreviewInput{Cube: true, Input: input, Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		var data struct {
			Rows []map[string]any `json:"rows"`
		}
		if err = json.Unmarshal(result.Data, &data); err != nil {
			t.Fatal(err)
		}
		return data.Rows
	}
	rows := execute(map[string]bool{"eventDate": true, "accountId": true, "productId": true})
	if len(rows) != 2 {
		t.Fatalf("facts were lost or expanded: %+v", rows)
	}
	for _, row := range rows {
		account, ok := row["account"].(map[string]any)
		if !ok || account["accountName"] != "Account Ten" {
			t.Fatalf("account lookup failed: %+v", row)
		}
		if row["productId"] == float64(1) {
			product, ok := row["product"].(map[string]any)
			if !ok || product["productName"] != "Product One" || row["units"] != float64(5) || row["amount"] != float64(50) {
				t.Fatalf("matched product changed facts: %+v", row)
			}
		} else if row["productId"] != float64(2) || row["product"] != nil || row["units"] != float64(7) {
			t.Fatalf("missing product changed facts: %+v", row)
		}
	}
	reduced := execute(map[string]bool{"eventDate": true})
	if len(reduced) != 1 || reduced[0]["units"] != float64(12) || reduced[0]["amount"] != float64(120) || reduced[0]["product"] != nil || reduced[0]["account"] != nil {
		t.Fatalf("reduced grain acquired keys/labels or wrong totals: %+v", reduced)
	}
	for _, filters := range []map[string]any{{}, {"dates": []string{"2026-10-01"}}} {
		input, _ := json.Marshal(map[string]any{"dimensions": map[string]bool{"eventDate": true}, "measures": map[string]bool{"units": true}, "filters": filters, "limit": 10})
		if _, err := (Dynamic{StudioDB: studio, ModulePath: "example.com/app"}).Execute(ctx, "dictionary-cube", 1, sdk.PreviewInput{Cube: true, Input: input, Limit: 10}); err == nil {
			t.Fatalf("cube executed without required source scope: %+v", filters)
		}
	}

	for _, filters := range []map[string]any{{"dates": []string{}, "accountIds": []int{10}}, {"dates": []string{"2026-10-01"}, "accountIds": []int{}}} {
		input, _ := json.Marshal(map[string]any{"dimensions": map[string]bool{"eventDate": true}, "measures": map[string]bool{"units": true}, "filters": filters, "limit": 10})
		result, err := (Dynamic{StudioDB: studio, ModulePath: "example.com/app"}).Execute(ctx, "dictionary-cube", 1, sdk.PreviewInput{Cube: true, Input: input, Limit: 10})
		if err == nil && result != nil {
			var empty struct {
				Rows []map[string]any `json:"rows"`
			}
			if json.Unmarshal(result.Data, &empty) != nil || len(empty.Rows) != 0 {
				t.Fatalf("empty required scope broadened the query: %s", result.Data)
			}
		}
	}

	_, compositionErr := (Dynamic{StudioDB: studio, ModulePath: "example.com/app"}).TestCompose(ctx, "dictionary-cube", 1, sdk.CubeComposeTestInput{})
	var compositionError *sdk.Error
	if !errors.As(compositionErr, &compositionError) || len(compositionError.Diagnostics) != 1 || compositionError.Diagnostics[0].Code != "cube_composition_not_enabled" {
		t.Fatalf("single cube/composition error is not actionable: %v", compositionErr)
	}
	ungrouped := `#package('example.com/app/dynamic/dictionarycube')
#setting($_ = $connector('facts'))
#setting($_ = $route('/dictionary-cube','POST'))
#setting($_ = $cube())
#define($_ = $Rows<[]*UngroupedRow>(output/view))
SELECT fact.*,type(fact,'UngroupedRow')
FROM (SELECT event_date,account_id,units,amount FROM fact) fact`
	if _, err := studio.Exec("UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id='dictionary-cube'", ungrouped, ungrouped); err != nil {
		t.Fatal(err)
	}
	_, err := (Dynamic{StudioDB: studio, ModulePath: "example.com/app"}).Execute(ctx, "dictionary-cube", 1, sdk.PreviewInput{Cube: true, Input: json.RawMessage(`{"filters":{"dates":["2026-10-01"],"accountIds":[10]}}`), Limit: 10})
	var contractError *sdk.Error
	if !errors.As(err, &contractError) || contractError.Code != sdk.ErrorInvalidArgument || len(contractError.Diagnostics) != 1 || contractError.Diagnostics[0].Code != "cube_source_not_eligible" {
		t.Fatalf("non-groupable cube source error is not actionable: %v", err)
	}

}
