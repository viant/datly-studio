package preview

import (
	"context"
	"database/sql"
	"encoding/json"
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
	if _, err := facts.Exec(`CREATE TABLE fact(event_date TEXT,advertiser_id INTEGER,publisher_id INTEGER,units INTEGER,amount REAL); INSERT INTO fact VALUES('2026-10-01',10,1,2,20),('2026-10-01',10,1,3,30),('2026-10-01',10,2,7,70)`); err != nil {
		t.Fatal(err)
	}
	if _, err := labels.Exec(`CREATE TABLE publisher(ID INTEGER PRIMARY KEY,NAME TEXT); INSERT INTO publisher VALUES(1,'Publisher One'); CREATE TABLE advertiser(ID INTEGER PRIMARY KEY,NAME TEXT); INSERT INTO advertiser VALUES(10,'Advertiser Ten')`); err != nil {
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
#define($_ = $AdvertiserIds<[]int>(query/advertiserIds).Required().WithPredicate(0,'in','f','advertiser_id'))
#define($_ = $Rows<[]*FactRow>(output/view))
SELECT fact.*,type(fact,'FactRow'),CAST(fact.units AS int),CAST(fact.amount AS float64),tag(fact.units,'groupable:"false"'),tag(fact.amount,'groupable:"false"'),
 publisher.*,type(publisher,'Publisher'),advertiser.*,type(advertiser,'Advertiser'),
 use_connector(fact,'facts'),use_connector(publisher,'labels'),use_connector(advertiser,'labels'),
 cardinality(publisher,'One'),cardinality(advertiser,'One'),allow_nulls(publisher),allow_nulls(advertiser),
 grouping_enabled(fact),set_limit(fact,50),selector_fields(fact,true),selector_limit(fact,true)
FROM (SELECT f.event_date,f.advertiser_id,f.publisher_id,SUM(f.units) AS units,SUM(f.amount) AS amount FROM fact f
${predicate.Builder().CombineOr($predicate.FilterGroup(0,"AND")).Build("WHERE")}
GROUP BY f.event_date,f.advertiser_id,f.publisher_id) fact
LEFT JOIN (SELECT ID AS publisher_id,NAME AS publisher_name FROM publisher) publisher ON publisher.publisher_id=fact.publisher_id
LEFT JOIN (SELECT ID AS advertiser_id,NAME AS advertiser_name FROM advertiser) advertiser ON advertiser.advertiser_id=fact.advertiser_id`
	if _, err := studio.Exec(`INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES('dictionary-cube',1,'draft','dql',?,?,'{}','studio.v1','hash','{}','pending','v1','studio.v1',1,'owner',?)`, dql, dql, now); err != nil {
		t.Fatal(err)
	}
	execute := func(dimensions map[string]bool) []map[string]any {
		input, _ := json.Marshal(map[string]any{"dimensions": dimensions, "measures": map[string]bool{"units": true, "amount": true}, "filters": map[string]any{"dates": []string{"2026-10-01"}, "advertiserIds": []int{10}}, "limit": 10})
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
	rows := execute(map[string]bool{"eventDate": true, "advertiserId": true, "publisherId": true})
	if len(rows) != 2 {
		t.Fatalf("facts were lost or expanded: %+v", rows)
	}
	for _, row := range rows {
		advertiser, ok := row["advertiser"].(map[string]any)
		if !ok || advertiser["advertiserName"] != "Advertiser Ten" {
			t.Fatalf("advertiser lookup failed: %+v", row)
		}
		if row["publisherId"] == float64(1) {
			publisher, ok := row["publisher"].(map[string]any)
			if !ok || publisher["publisherName"] != "Publisher One" || row["units"] != float64(5) || row["amount"] != float64(50) {
				t.Fatalf("matched publisher changed facts: %+v", row)
			}
		} else if row["publisherId"] != float64(2) || row["publisher"] != nil || row["units"] != float64(7) {
			t.Fatalf("missing publisher changed facts: %+v", row)
		}
	}
	reduced := execute(map[string]bool{"eventDate": true})
	if len(reduced) != 1 || reduced[0]["units"] != float64(12) || reduced[0]["amount"] != float64(120) || reduced[0]["publisher"] != nil || reduced[0]["advertiser"] != nil {
		t.Fatalf("reduced grain acquired keys/labels or wrong totals: %+v", reduced)
	}
	for _, filters := range []map[string]any{{}, {"dates": []string{"2026-10-01"}}} {
		input, _ := json.Marshal(map[string]any{"dimensions": map[string]bool{"eventDate": true}, "measures": map[string]bool{"units": true}, "filters": filters, "limit": 10})
		if _, err := (Dynamic{StudioDB: studio, ModulePath: "example.com/app"}).Execute(ctx, "dictionary-cube", 1, sdk.PreviewInput{Cube: true, Input: input, Limit: 10}); err == nil {
			t.Fatalf("cube executed without required source scope: %+v", filters)
		}
	}

	for _, filters := range []map[string]any{{"dates": []string{}, "advertiserIds": []int{10}}, {"dates": []string{"2026-10-01"}, "advertiserIds": []int{}}} {
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

}
