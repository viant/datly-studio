package warmup_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/sdk"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/datly-studio/studio/host"
	"github.com/viant/datly/authoring/readerbuilder"
	"github.com/viant/datly/spec"
)

// The shared native fixture owns the listeners, host snapshot and datastore.
func testHostCacheSelection(ctx context.Context, t *testing.T, db *sql.DB, namespace, bearer, runner, httpAddress string, authoring *host.Authoring, callRaw func(context.Context, string, string, map[string]any) (json.RawMessage, bool)) {
	t.Helper()
	call := func(ctx context.Context, protocol, operation, token string, args map[string]any) (json.RawMessage, bool) {
		t.Helper()
		if protocol == "mcp" {
			return callRaw(ctx, "studio.sdk."+operation, token, args)
		}
		body, err := json.Marshal(args)
		if err != nil {
			t.Fatal(err)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+httpAddress+"/v1/studio/sdk/"+operation, bytes.NewReader(body))
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
		raw, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		return raw, response.StatusCode == http.StatusOK
	}
	types, err := authoring.RuntimeTypes()
	if err != nil {
		t.Fatal(err)
	}
	access, err := authorization.NewSDKAuthorizer(db)
	if err != nil {
		t.Fatal(err)
	}
	defer access.Close(context.WithoutCancel(ctx))
	transport := &sqltransport.Transport{DB: db, ReaderCaches: authoring.ReaderCaches(), RuntimeTypes: types, Predicates: authoring.PredicateCatalog(), Authorizer: access}
	defer transport.Close(context.WithoutCancel(ctx))
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	// The SDK's namespace-role verifier needs the same signed credential used
	// by HTTP/MCP, rather than an unverified subject-only test context.
	owner := oauth.WithBearer(sdk.WithNamespaceSelection(sdk.WithPrincipal(ctx, sdk.Principal{Subject: "alice"}), namespace), strings.TrimPrefix(bearer, "Bearer "))
	type structure struct {
		AvailableCaches []string                           `json:"availableCaches"`
		Component       *spec.Component                    `json:"component"`
		Functions       []readerbuilder.FunctionOccurrence `json:"functions"`
	}
	decode := func(raw json.RawMessage) structure {
		t.Helper()
		var value structure
		if err := json.Unmarshal(raw, &value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	for _, protocol := range []string{"http", "mcp"} {
		t.Run(protocol, func(t *testing.T) {
			id := "storage-selection-" + protocol
			scope := "example.com/fixture/storage_" + protocol
			source := fmt.Sprintf(`#package('%s')
#setting($_ = $connector('main'))
#setting($_ = $route('/records/%s','GET'))
#setting($_ = $cache('component','1m').WithLocation('source-cache'))
#setting($_ = $cache_warmup_cases('recent','Period=recent'))
#setting($_ = $cache_warmup('ID','Limit=0'))
#define($_ = $Period<string>(query/period).Optional())
SELECT r.*, c.*, s.*,use_cache(r,'old-root'),use_cache(c,'old-child'),use_cache(s,'unchanged'),
use_connector(c,'main'),cache_warmup(c,'ID','CaseRefs=recent','Limit=0'),cache_warmup(s,'host-policy')
FROM records r LEFT JOIN children c ON c.parent_id=r.ID LEFT JOIN summary s ON s.ID=r.ID`, scope, id)
			now := "2026-11-17 00:00:00"
			if err := datatest.Hydrate(ctx, db,
				datatest.Table{Name: "components", Rows: []datatest.Row{{"id": id, "namespace_id": namespace, "namespace": "alpha", "slug": id, "title": id, "owner_id": "alice", "status": "draft", "default_connector_name": "main", "component_scope": scope, "component_name": "reader", "created_at": now, "updated_at": now}}},
				datatest.Table{Name: "component_versions", Rows: []datatest.Row{{"namespace_id": namespace, "report_id": id, "version_no": 2, "state": "draft", "authoring_mode": "dql", "authored_dql": source, "generated_dql": source, "component_spec_json": "{}", "type_manifest_json": "{}", "spec_format_version": "1", "spec_hash": id, "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 7, "created_by": "alice", "created_at": now}}},
			); err != nil {
				t.Fatal(err)
			}
			if _, err := db.ExecContext(ctx, "INSERT INTO component_acl(namespace_id,report_id,subject_type,subject_id,can_view,can_run,can_publish,can_use_dql) VALUES(?,?,'user','cache-runner',1,1,1,0)", namespace, id); err != nil {
				t.Fatal(err)
			}
			args := map[string]any{"namespaceId": namespace, "reportId": id, "versionNo": 2, "discoverColumns": false, "includePreviewInput": false, "includeEffectiveWarmup": false}
			raw, ok := call(ctx, protocol, "versions.inspect", bearer, args)
			var inspected sdk.ReaderInspection
			if !ok || json.Unmarshal(raw, &inspected) != nil {
				t.Fatalf("native inspection failed: %s", raw)
			}
			before := decode(inspected.Structure)
			sdkInspection, err := client.Versions().Inspect(owner, id, 2)
			if err != nil {
				var detail *sdk.Error
				if errors.As(err, &detail) {
					t.Fatalf("SDK inspection failed: %v cause=%v", err, detail.Cause)
				}
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before.AvailableCaches, decode(sdkInspection.Structure).AvailableCaches) {
				t.Fatal("native and SDK storage catalogs differ")
			}
			names := map[string]bool{}
			for _, name := range before.AvailableCaches {
				names[name] = true
			}
			if !names["shared-storage"] || !names["combined-storage"] || names["policy-only"] || names["disabled-storage"] || names["nil-storage"] || names["not-the-key"] {
				t.Fatalf("incorrect trusted catalog: %v", before.AvailableCaches)
			}
			for _, private := range []string{"PRIVATE_HOST_", "unreachable.invalid", "not-the-key"} {
				if strings.Contains(string(raw), private) {
					t.Fatalf("host configuration leaked: %s", raw)
				}
			}
			hidden, ok := call(ctx, protocol, "versions.inspect", runner, args)
			if !ok || strings.Contains(string(hidden), `"availableCaches"`) || strings.Contains(string(hidden), "PRIVATE_HOST_") {
				t.Fatalf("catalog permission filtering failed: %s", hidden)
			}
			var calls []readerbuilder.FunctionOccurrence
			for _, f := range before.Functions {
				if f.Name == "use_cache" && f.Occurrence < 2 {
					calls = append(calls, f)
				}
			}
			if len(calls) != 2 {
				t.Fatal("fixture bindings missing")
			}
			update := func(f readerbuilder.FunctionOccurrence, name string) readerbuilder.Operation {
				return readerbuilder.Operation{Type: readerbuilder.OperationUpdateFunction, Function: &readerbuilder.FunctionMutation{Name: f.Name, Occurrence: f.Occurrence, ExpectedArgs: f.Args, Args: []string{f.Args[0], "'" + name + "'"}}}
			}
			mutate := func(revision int64, token string, ops ...readerbuilder.Operation) (sdk.ReaderBuilderResult, bool) {
				t.Helper()
				operation, _ := json.Marshal(readerbuilder.Operation{Type: readerbuilder.OperationBatch, Operations: ops})
				raw, ok := call(ctx, protocol, "versions.builder", token, map[string]any{"namespaceId": namespace, "reportId": id, "versionNo": 2, "command": sdk.ReaderBuilderCommand{ExpectedSourceRevision: revision, Operation: operation}})
				if !ok {
					return sdk.ReaderBuilderResult{}, false
				}
				var result sdk.ReaderBuilderResult
				if err := json.Unmarshal(raw, &result); err != nil {
					t.Fatalf("builder response: %v: %s", err, raw)
				}
				return result, true
			}
			unchanged := func() {
				t.Helper()
				var authored, generated string
				var revision int64
				if err := db.QueryRowContext(ctx, "SELECT authored_dql,generated_dql,source_revision FROM component_versions WHERE report_id=? AND version_no=2", id).Scan(&authored, &generated, &revision); err != nil || authored != source || generated != source || revision != 7 {
					t.Fatalf("rejected mutation changed persisted source: %v", err)
				}
			}
			root, child := update(calls[0], "shared-storage"), update(calls[1], "shared-storage")
			if _, ok := mutate(8, bearer, root, child); ok {
				t.Fatal("stale revision accepted")
			}
			unchanged()
			bad := update(calls[1], "shared-storage")
			bad.Function.ExpectedArgs = []string{"c", "'stale'"}
			rejected, ok := mutate(7, bearer, root, bad)
			if !ok || rejected.Applied {
				t.Fatalf("late argument mismatch accepted: %+v", rejected)
			}
			unchanged()
			rejected, ok = mutate(7, bearer, root, update(calls[1], "policy-only"))
			if !ok || rejected.Applied {
				t.Fatal("policy-only entry accepted as storage")
			}
			found := false
			for _, d := range rejected.Inspection.Diagnostics {
				found = found || strings.Contains(d.Message, "no enabled configuration available to the reader builder service")
			}
			if !found {
				t.Fatalf("native validation missing: %+v", rejected)
			}
			unchanged()
			if _, ok := mutate(7, runner, root, child); ok {
				t.Fatal("source-denied user mutated bindings")
			}
			unchanged()
			saved, ok := mutate(7, bearer, root, child)
			if !ok || !saved.Applied || saved.Inspection.Version.SourceRevision != 8 {
				t.Fatalf("guarded storage save failed: %+v", saved)
			}
			want := before.Component.Clone()
			want.RootView.Source.Bindings.CacheName = "shared-storage"
			want.RootView.Relations[0].View.Source.Bindings.CacheName = "shared-storage"
			after := decode(saved.Inspection.Structure)
			wantJSON, _ := json.Marshal(want)
			afterJSON, _ := json.Marshal(after.Component)
			if !bytes.Equal(wantJSON, afterJSON) || !reflect.DeepEqual(before.AvailableCaches, after.AvailableCaches) {
				t.Fatal("binding save changed unrelated definitions or catalog")
			}
			native := readerbuilder.New(readerbuilder.Config{Name: before.Component.Name, Scope: before.Component.Key.Scope, AvailableCaches: before.AvailableCaches, AvailableConnectors: []string{"main"}}).Apply(ctx, readerbuilder.Request{DQL: source, Operation: readerbuilder.Operation{Type: readerbuilder.OperationBatch, Operations: []readerbuilder.Operation{root, child}}})
			if !native.Applied || saved.Inspection.DQL != native.DQL {
				t.Fatal("Studio rewrote source beyond the native guarded binding edits")
			}
			raw, ok = call(ctx, protocol, "versions.inspect", bearer, args)
			if !ok || json.Unmarshal(raw, &inspected) != nil {
				t.Fatalf("native readback failed: %s", raw)
			}
			if inspected.Version.SourceRevision != 8 || !reflect.DeepEqual(after, decode(inspected.Structure)) {
				t.Fatal("native binding metadata did not round-trip")
			}
		})
	}
}
