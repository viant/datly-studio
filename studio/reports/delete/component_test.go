package delete_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/viant/datly-studio/internal/componentremoval/removaltest"
	"github.com/viant/datly-studio/internal/datatest"
)

func TestNativeComponentDeletion(t *testing.T) {
	ctx := context.Background()
	f := removaltest.New(ctx, t)
	f.Seed(ctx, "remove", 2)
	f.Seed(ctx, "keep", 1)
	f.Exec(ctx, `CREATE TABLE application_rows (id INTEGER PRIMARY KEY, value TEXT NOT NULL)`)
	f.Exec(ctx, `INSERT INTO application_rows VALUES (1,'preserved')`)
	f.Hydrate(ctx, datatest.Table{Name: "component_publication_events", Rows: []datatest.Row{{"namespace_id": f.Namespace, "event_id": "historical-event", "report_id": "remove", "owner_id": "alice", "operation": "unpublish", "version_no": 2, "status": "succeeded", "requested_by": "alice", "occurred_at": removaltest.Now}}},
		datatest.Table{Name: "tool_call_audit", Rows: []datatest.Row{{"namespace_id": f.Namespace, "component_id": "remove", "tool_name": "fixture.read", "started_at": removaltest.Now, "status": "succeeded", "jwt_sub": "alice", "request_body": []byte("history")}}})
	t.Run("delegated permissions do not confer ownership", func(t *testing.T) {
		f.T = t
		for _, subject := range []string{"editor", "publisher", "admin"} {
			f.Hydrate(ctx, datatest.Table{Name: "component_acl", Rows: []datatest.Row{{"namespace_id": f.Namespace, "report_id": "remove", "subject_type": "user", "subject_id": subject, "can_view": true, "can_edit": subject == "editor", "can_publish": subject != "editor"}}})
			f.HTTP(ctx, "components.delete", subject, f.Namespace, removaltest.Component("remove", 1), 403, subject)
			f.HTTP(ctx, "versions.delete", subject, f.Namespace, removaltest.Version("remove", 2, 7), 403, subject)
		}
		f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id='remove'", 2)
		f.Count(ctx, "SELECT COUNT(*) FROM components WHERE id='remove' AND current_draft_version=2 AND etag=1", 1)
	})
	t.Run("stale etag", func(t *testing.T) {
		f.T = t
		f.HTTP(ctx, "components.delete", "alice", f.Namespace, removaltest.Component("remove", 2), 409)
		f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id='remove'", 2)
		f.Count(ctx, "SELECT COUNT(*) FROM component_deletions WHERE component_id='remove'", 0)
	})
	t.Run("namespace isolation", func(t *testing.T) {
		f.T = t
		f.HTTP(ctx, "components.delete", "alice", f.OtherNamespace, removaltest.Component("remove", 1), 403)
		f.Count(ctx, "SELECT COUNT(*) FROM components WHERE id='remove'", 1)
	})
	t.Run("receipt and retained history", func(t *testing.T) {
		f.T = t
		f.Result(f.HTTP(ctx, "components.delete", "alice", f.Namespace, removaltest.Component("remove", 1), 200), "remove", 0, 2)
		f.Count(ctx, "SELECT COUNT(*) FROM components WHERE id='remove'", 0)
		f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id='remove'", 0)
		f.Count(ctx, "SELECT COUNT(*) FROM component_deletions WHERE component_id='remove' AND namespace_id=? AND owner_id='alice' AND deleted_by='alice' AND deleted_at IS NOT NULL", 1, f.Namespace)
		f.Count(ctx, "SELECT COUNT(*) FROM component_publication_events WHERE event_id='historical-event' AND owner_id='alice' AND report_id='remove' AND version_no=2", 1)
		f.Count(ctx, "SELECT COUNT(*) FROM tool_call_audit WHERE component_id='remove' AND request_body=?", 1, []byte("history"))
		f.Count(ctx, "SELECT COUNT(*) FROM application_rows WHERE id=1 AND value='preserved'", 1)
		f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id='keep'", 1)
		history := f.HTTP(ctx, "publications.events.list", "alice", f.Namespace, map[string]any{"reportId": "remove", "input": map[string]any{}}, 200)
		var page struct {
			Items []struct {
				EventID        string `json:"eventId"`
				VersionDeleted bool   `json:"versionDeleted"`
			} `json:"items"`
		}
		if err := json.Unmarshal(history, &page); err != nil || len(page.Items) != 1 || page.Items[0].EventID != "historical-event" || !page.Items[0].VersionDeleted {
			t.Fatalf("owner retained history: %s err=%v", history, err)
		}
		// The native history reader enforces ownership by filtering its result.
		deniedHistory := f.HTTP(ctx, "publications.events.list", "admin", f.Namespace, map[string]any{"reportId": "remove", "input": map[string]any{}}, 200, "admin")
		if err := json.Unmarshal(deniedHistory, &page); err != nil || len(page.Items) != 0 {
			t.Fatalf("nonowner read retained history: %s err=%v", deniedHistory, err)
		}
		f.HTTP(ctx, "components.create", "alice", f.Namespace, map[string]any{"id": "remove", "slug": "replacement", "title": "Replacement", "defaultConnectorName": "main"}, 409)
		f.HTTP(ctx, "components.delete", "alice", f.Namespace, removaltest.Component("remove", 1), 404)
		f.Count(ctx, "SELECT COUNT(*) FROM component_deletions WHERE component_id='remove'", 1)
	})
	t.Run("empty component over MCP", func(t *testing.T) {
		f.T = t
		f.Seed(ctx, "empty", 0)
		f.Result(f.MCP(ctx, "components.delete", removaltest.Component("empty", 1)), "empty", 0, 0)
		f.Count(ctx, "SELECT COUNT(*) FROM components WHERE id='empty'", 0)
		f.Count(ctx, "SELECT COUNT(*) FROM component_deletions WHERE component_id='empty'", 1)
	})
}

func TestNativeDeletionBlockers(t *testing.T) {
	ctx := context.Background()
	f := removaltest.New(ctx, t)
	sequence := 0
	for _, operation := range []string{"components.delete", "versions.delete"} {
		for _, blocker := range []string{"active publication", "desired pending", "desired failed", "accepted warmup", "running warmup", "retained JSON release", "retained numeric version spelling"} {
			t.Run(operation+"/"+blocker, func(t *testing.T) {
				f.T = t
				id := fmt.Sprintf("blocked-%d", sequence)
				sequence++
				f.Seed(ctx, id, 2)
				switch blocker {
				case "active publication", "desired pending", "desired failed":
					active, status := 1, "active"
					var desired any
					if blocker != "active publication" {
						active, desired, status = 2, 1, "pending"
					}
					if blocker == "desired failed" {
						status = "failed"
					}
					f.Hydrate(ctx, datatest.Table{Name: "component_publications", Rows: []datatest.Row{{"namespace_id": f.Namespace, "report_id": id, "active_version_no": active, "desired_version_no": desired, "desired_generation": 1, "publication_status": status, "spec_hash": "fixture", "published_by": "alice", "published_at": removaltest.Now}}})
				case "accepted warmup":
					f.Warmup(ctx, id, 1, "accepted")
				case "running warmup":
					f.Warmup(ctx, id, 1, "running")
				case "retained JSON release":
					f.RetainedBinding(ctx, id, "1")
				case "retained numeric version spelling":
					f.RetainedBinding(ctx, id, "01")
				}
				body := removaltest.Component(id, 1)
				if operation == "versions.delete" {
					body = removaltest.Version(id, 1, 7)
				}
				raw := f.HTTP(ctx, operation, "alice", f.Namespace, body, 409)
				if !bytes.Contains(bytes.ToLower(raw), []byte("publication")) && !bytes.Contains(bytes.ToLower(raw), []byte("warmup")) && !bytes.Contains(bytes.ToLower(raw), []byte("release")) {
					t.Fatalf("conflict did not identify dependency: %s", raw)
				}
				f.Count(ctx, "SELECT COUNT(*) FROM components WHERE id=? AND current_draft_version=2 AND etag=1 AND last_version_no=2", 1, id)
				f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id=?", 2, id)
				f.Count(ctx, "SELECT COUNT(*) FROM component_deletions WHERE component_id=?", 0, id)
			})
		}
	}
	t.Run("references to sibling version do not block exact target", func(t *testing.T) {
		f.T = t
		f.Seed(ctx, "sibling-reference", 2)
		f.RetainedBinding(ctx, "sibling-reference", "2")
		f.Warmup(ctx, "sibling-reference", 2, "running")
		f.Hydrate(ctx, datatest.Table{Name: "component_publications", Rows: []datatest.Row{{"namespace_id": f.Namespace, "report_id": "sibling-reference", "active_version_no": 2, "desired_version_no": 2, "desired_generation": 1, "publication_status": "failed", "spec_hash": "fixture", "published_by": "alice", "published_at": removaltest.Now}}})
		f.Result(f.HTTP(ctx, "versions.delete", "alice", f.Namespace, removaltest.Version("sibling-reference", 1, 7), 200), "sibling-reference", 1, 1)
		f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id='sibling-reference' AND version_no=2", 1)
		f.Count(ctx, "SELECT COUNT(*) FROM components WHERE id='sibling-reference' AND current_draft_version=2 AND last_version_no=2", 1)
	})
}

func TestNativeDeletionTerminalWarmupsAndRollback(t *testing.T) {
	ctx := context.Background()
	f := removaltest.New(ctx, t)
	for _, operation := range []string{"components.delete", "versions.delete"} {
		t.Run(operation, func(t *testing.T) {
			f.T = t
			id := "atomic-component"
			if operation == "versions.delete" {
				id = "atomic-version"
			}
			f.Seed(ctx, id, 2)
			for v := 1; v <= 2; v++ {
				f.Graph(ctx, id, v)
				for _, status := range []string{"completed", "partial", "failed", "canceled"} {
					f.Warmup(ctx, id, v, status)
				}
			}
			f.Hydrate(ctx, datatest.Table{Name: "component_acl", Rows: []datatest.Row{{"namespace_id": f.Namespace, "report_id": id, "subject_type": "user", "subject_id": "editor", "can_view": true, "can_edit": true}}},
				datatest.Table{Name: "resource_namespace_claims", Rows: []datatest.Row{{"namespace_id": f.Namespace, "namespace": id + ".bundle", "report_id": id, "created_at": removaltest.Now, "created_by": "alice", "updated_at": removaltest.Now, "updated_by": "alice"}}})
			// The late root/version delete fails only after owned child deletions have
			// been queued. All changes, including receipt creation, must roll back.
			trigger := `CREATE TRIGGER reject_removal BEFORE DELETE ON components WHEN OLD.id='atomic-component' BEGIN SELECT RAISE(ABORT,'injected late removal failure'); END`
			if operation == "versions.delete" {
				trigger = `CREATE TRIGGER reject_removal BEFORE DELETE ON component_versions WHEN OLD.report_id='atomic-version' AND OLD.version_no=2 BEGIN SELECT RAISE(ABORT,'injected late removal failure'); END`
			}
			f.Exec(ctx, trigger)
			body := removaltest.Component(id, 1)
			if operation == "versions.delete" {
				body = removaltest.Version(id, 2, 7)
			}
			// Native storage faults are internal errors; an exact status makes an
			// earlier validation conflict unable to masquerade as the injected fault.
			f.HTTP(ctx, operation, "alice", f.Namespace, body, 500)
			for _, table := range []string{"component_versions", "component_views", "component_fields", "component_parameters", "component_predicates", "component_resource_files", "component_resource_folders", "component_skill_roots"} {
				f.Count(ctx, "SELECT COUNT(*) FROM "+table+" WHERE report_id=?", 2, id)
			}
			f.Count(ctx, "SELECT COUNT(*) FROM component_warmup_runs WHERE report_id=?", 8, id)
			f.Count(ctx, "SELECT COUNT(*) FROM components WHERE id=? AND etag=1 AND last_version_no=2 AND current_draft_version=2", 1, id)
			f.Count(ctx, "SELECT COUNT(*) FROM component_deletions WHERE component_id=?", 0, id)
			f.Count(ctx, "SELECT COUNT(*) FROM component_acl WHERE report_id=?", 1, id)
			f.Count(ctx, "SELECT COUNT(*) FROM resource_namespace_claims WHERE report_id=?", 1, id)
			f.Exec(ctx, "DROP TRIGGER reject_removal")
			deleted, remaining := 2, 0
			if operation == "versions.delete" {
				deleted, remaining = 1, 1
			}
			result := f.HTTP(ctx, operation, "alice", f.Namespace, body, 200)
			version := 0
			if operation == "versions.delete" {
				version = 2
			}
			f.Result(result, id, version, deleted)
			for _, table := range []string{"component_versions", "component_views", "component_fields", "component_parameters", "component_predicates", "component_resource_files", "component_resource_folders", "component_skill_roots"} {
				f.Count(ctx, "SELECT COUNT(*) FROM "+table+" WHERE report_id=?", remaining, id)
			}
			f.Count(ctx, "SELECT COUNT(*) FROM component_warmup_runs WHERE report_id=?", 4*remaining, id)
			f.Count(ctx, "SELECT COUNT(*) FROM component_acl WHERE report_id=?", remaining, id)
			f.Count(ctx, "SELECT COUNT(*) FROM resource_namespace_claims WHERE report_id=?", remaining, id)
		})
	}
}

func TestNativeDeletionUnboundedGraph(t *testing.T) {
	ctx := context.Background()
	f := removaltest.New(ctx, t)
	const size = 105
	f.Seed(ctx, "large-graph", size)
	rows := make([]datatest.Row, 0, size)
	for i := 0; i < size; i++ {
		rows = append(rows, datatest.Row{"namespace_id": f.Namespace, "run_id": fmt.Sprintf("terminal-%d", i), "report_id": "large-graph", "version_no": size, "source_revision": 7, "spec_hash": fmt.Sprintf("%064d", size), "plan_key": fmt.Sprintf("%064d", i), "status": "completed", "requested_by": "alice", "target_json": "{}", "requested_at": removaltest.Now})
	}
	f.Hydrate(ctx, datatest.Table{Name: "component_warmup_runs", Rows: rows})
	f.Result(f.HTTP(ctx, "components.delete", "alice", f.Namespace, removaltest.Component("large-graph", 1), 200), "large-graph", 0, size)
	f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id='large-graph'", 0)
	f.Count(ctx, "SELECT COUNT(*) FROM component_warmup_runs WHERE report_id='large-graph'", 0)
	f.Count(ctx, "SELECT COUNT(*) FROM component_deletions WHERE component_id='large-graph'", 1)
}

// The shipped host must not expose the generic writer's physical delete marker.
// Its source package remains available for internal/library reuse.
func TestRawComponentWriterIsWithdrawn(t *testing.T) {
	ctx := context.Background()
	f := removaltest.New(ctx, t)
	f.Seed(ctx, "withdrawn-writer", 1)
	f.RetainedBinding(ctx, "withdrawn-writer", "1")
	f.HTTP(ctx, "components.delete", "alice", f.Namespace, removaltest.Component("withdrawn-writer", 1), http.StatusConflict)
	request, err := http.NewRequestWithContext(ctx, http.MethodPatch, "http://"+f.Addresses[0]+"/v1/studio/reports", bytes.NewBufferString(`{"data":[{"id":"withdrawn-writer","etag":1,"shouldDelete":true}]}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", f.Token("alice"))
	request.Header.Set("X-Studio-Namespace", f.Namespace)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusNotFound && response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("withdrawn raw writer status=%d body=%s", response.StatusCode, body)
	}
	f.Count(ctx, "SELECT COUNT(*) FROM components WHERE id='withdrawn-writer'", 1)
	f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id='withdrawn-writer'", 1)
	f.Count(ctx, "SELECT COUNT(*) FROM artifact_release_bundle_bindings", 1)
	f.Count(ctx, "SELECT COUNT(*) FROM component_deletions WHERE component_id='withdrawn-writer'", 0)
}
