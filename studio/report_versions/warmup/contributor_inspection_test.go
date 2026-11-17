package warmup_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/authoring/readerbuilder"
)

// Reuse the native server, owned datastore, credentials and transport from the
// warmup lifecycle fixture. Neither inspection projection executes warmup.
func testContributorInspection(ctx context.Context, t *testing.T, db *sql.DB, namespace, bearer, runner, httpAddress string, callRaw func(context.Context, string, string, map[string]any) (json.RawMessage, bool)) {
	t.Helper()
	for _, readOnly := range []bool{false, true} {
		variant := "editable"
		if readOnly {
			variant = "readonly"
		}
		id := "contributors-" + variant
		scope := "example.com/fixture/contributors_" + variant
		source := fmt.Sprintf(`#package('%s')
#setting($_ = $connector('main'))
#setting($_ = $route('/records/%s','GET'))
#setting($_ = $cache_warmup_cases('recent','Period=PRIVATE_NATIVE_CASE_CANARY'))
#define($_ = $Period<string>(query/period).Optional())
#define($_ = $Records<[]*Record>(output/view))
SELECT records.*, child.*, type(records,'Record'), type(child,'Child'),
use_cache(records,'named-root-root-cache'),use_cache(child,'named-child-child-cache'),
cache_warmup(records,'ID','Limit=0'),
cache_warmup(child,'ID','MaxCases=4','Limit=0','CaseRefs=recent','Period=older'),
cache_warmup(child,'ID','MaxCases=4','Limit=0','Period=oldest'),
cache_warmup(child,'NAME','Name=by_name')
FROM (SELECT ID, NAME FROM RECORDS) records
LEFT JOIN (SELECT ID, NAME FROM RECORDS) child ON child.ID=records.ID`, scope, id)
		if readOnly {
			source = strings.Replace(source, "cache_warmup(records,'ID','Limit=0')", "cache_warmup(records,'ID','Limit=0') AS warmup", 1)
		}
		native := readerbuilder.New(readerbuilder.Config{Name: "reader"}).Apply(ctx, readerbuilder.Request{DQL: source, Operation: readerbuilder.Operation{Type: readerbuilder.OperationInspect}})
		if len(native.Diagnostics) != 0 || len(native.Structure.TargetedWarmups) != 3 {
			t.Fatalf("invalid contributor fixture: %+v", native.Diagnostics)
		}
		specJSON, err := json.Marshal(native.Structure.Component)
		if err != nil {
			t.Fatal(err)
		}
		now := "2026-11-17 00:00:00"
		if err = datatest.Hydrate(ctx, db,
			datatest.Table{Name: "components", Rows: []datatest.Row{{"id": id, "namespace_id": namespace, "namespace": "alpha", "slug": id, "title": id, "owner_id": "alice", "status": "draft", "default_connector_name": "main", "component_scope": scope, "component_name": "reader", "created_at": now, "updated_at": now}}},
			datatest.Table{Name: "component_versions", Rows: []datatest.Row{{"namespace_id": namespace, "report_id": id, "version_no": 2, "state": "draft", "authoring_mode": "dql", "authored_dql": source, "generated_dql": source, "component_spec_json": string(specJSON), "type_manifest_json": "{}", "spec_format_version": "1", "spec_hash": id, "compile_status": "valid", "compile_diagnostics_json": "[{\"message\":\"PRIVATE_NATIVE_DIAGNOSTIC_CANARY\"}]", "datly_version": "v1", "compiler_version": "v1", "source_revision": 7, "created_by": "alice", "created_at": now}}},
		); err != nil {
			t.Fatal(err)
		}
		if _, err = db.ExecContext(ctx, "INSERT INTO component_acl(namespace_id,report_id,subject_type,subject_id,can_view,can_run,can_publish,can_use_dql) VALUES(?,?,'user','contributor-runner',1,1,1,0)", namespace, id); err != nil {
			t.Fatal(err)
		}
		for _, protocol := range []string{"http", "mcp"} {
			for _, allowed := range []bool{true, false} {
				t.Run(fmt.Sprintf("%s/%s/source=%v", variant, protocol, allowed), func(t *testing.T) {
					token := bearer
					if !allowed {
						token = runner
					}
					arguments := map[string]any{"namespaceId": namespace, "reportId": id, "versionNo": 2, "discoverColumns": false, "includePreviewInput": false, "includeEffectiveWarmup": false}
					var raw json.RawMessage
					if protocol == "mcp" {
						var ok bool
						raw, ok = callRaw(ctx, "studio.sdk.versions.inspect", token, arguments)
						if !ok {
							t.Fatalf("native MCP inspection failed: %s", raw)
						}
					} else {
						body, _ := json.Marshal(arguments)
						req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+httpAddress+"/v1/studio/sdk/versions.inspect", bytes.NewReader(body))
						if err != nil {
							t.Fatal(err)
						}
						req.Header.Set("Authorization", token)
						req.Header.Set("Content-Type", "application/json")
						req.Header.Set("X-Studio-Namespace", namespace)
						response, err := http.DefaultClient.Do(req)
						if err != nil {
							t.Fatal(err)
						}
						raw, err = io.ReadAll(response.Body)
						response.Body.Close()
						if err != nil || response.StatusCode != http.StatusOK {
							t.Fatalf("native HTTP inspection %d: %s err=%v", response.StatusCode, raw, err)
						}
					}
					var got sdk.ReaderInspection
					if err := json.Unmarshal(raw, &got); err != nil {
						t.Fatalf("inspection JSON: %v: %s", err, raw)
					}
					if got.Version == nil || got.Version.ReportID != id || got.Version.VersionNo != 2 || got.Version.SourceRevision != 7 || got.Capabilities.CanUseDQL != allowed {
						t.Fatalf("wrong snapshot or permission: %s", raw)
					}
					if allowed {
						var structure struct {
							TargetedWarmups []readerbuilder.TargetedWarmupPlan `json:"targetedWarmups"`
						}
						if err := json.Unmarshal(got.Structure, &structure); err != nil {
							t.Fatal(err)
						}
						if !reflect.DeepEqual(structure.TargetedWarmups, native.Structure.TargetedWarmups) {
							t.Fatalf("native metadata was changed: %s", got.Structure)
						}
						plans := structure.TargetedWarmups
						if plans[0].Editable == readOnly || !plans[1].Editable || !plans[2].Editable || plans[1].Settings.MaxCases == nil || *plans[1].Settings.MaxCases != 4 || len(plans[1].Contributors) != 2 || len(plans[1].Settings.Cases) != 2 || !reflect.DeepEqual(plans[1].Settings.CaseRefs, []string{"recent"}) || plans[0].Settings.MaxCases != nil || plans[0].Settings.Limit == nil || *plans[0].Settings.Limit != 0 || plans[2].Settings.Limit != nil || plans[2].Settings.MaxCases != nil {
							t.Fatalf("incomplete authored metadata: %+v", plans)
						}
						if readOnly {
							var wire struct {
								TargetedWarmups []map[string]json.RawMessage `json:"targetedWarmups"`
							}
							if err := json.Unmarshal(got.Structure, &wire); err != nil {
								t.Fatal(err)
							}
							if plans[0].Diagnostic == "" || string(wire.TargetedWarmups[0]["editable"]) != "false" || string(wire.TargetedWarmups[0]["contributors"]) != "[]" {
								t.Fatalf("read-only wire shape changed: %s", got.Structure)
							}
						}
					} else {
						if got.DQL != "" || got.Version.AuthoredDQL != "" || got.Version.GeneratedDQL != "" || len(got.Version.ComponentSpec) != 0 || len(got.Version.CompileDiagnostics) != 0 {
							t.Fatalf("source-bearing envelope fields leaked: %s", raw)
						}
						for _, value := range []string{`"targetedWarmups"`, `"contributors"`, `"inlineWarmups"`, `"sourceSpan"`, "PRIVATE_NATIVE_CASE_CANARY", "PRIVATE_NATIVE_DIAGNOSTIC_CANARY"} {
							if strings.Contains(string(raw), value) {
								t.Fatalf("source metadata leaked: %s", raw)
							}
						}
						if readOnly && strings.Contains(string(raw), native.Structure.TargetedWarmups[0].Diagnostic) {
							t.Fatalf("source-derived diagnostic leaked: %s", raw)
						}
					}
				})
			}
		}
		var stored string
		var revision int64
		if err = db.QueryRowContext(ctx, "SELECT authored_dql,source_revision FROM component_versions WHERE report_id=? AND version_no=2", id).Scan(&stored, &revision); err != nil || stored != source || revision != 7 {
			t.Fatalf("inspection changed source: %v", err)
		}
	}
}
