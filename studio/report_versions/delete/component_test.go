package delete_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/viant/datly-studio/internal/componentremoval/removaltest"
)

func TestNativeVersionDeletion(t *testing.T) {
	ctx := context.Background()
	f := removaltest.New(ctx, t)
	f.Seed(ctx, "versions", 2)
	f.Exec(ctx, `INSERT INTO component_warmup_schedules(schedule_id,namespace_id,report_id,version_no,interval_seconds,first_run_at,next_due_at,enabled,revision,created_at,created_by,updated_at,updated_by) VALUES('delete-owned-schedule',?,'versions',2,3600,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,1,1,CURRENT_TIMESTAMP,'alice',CURRENT_TIMESTAMP,'alice')`, f.Namespace)
	t.Run("stale revision", func(t *testing.T) {
		f.T = t
		f.HTTP(ctx, "versions.delete", "alice", f.Namespace, removaltest.Version("versions", 2, 6), 409)
		f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id='versions'", 2)
		f.Count(ctx, "SELECT current_draft_version FROM components WHERE id='versions'", 2)
	})
	t.Run("namespace isolation", func(t *testing.T) {
		f.T = t
		f.HTTP(ctx, "versions.delete", "alice", f.OtherNamespace, removaltest.Version("versions", 2, 7), 403)
		f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id='versions'", 2)
	})
	t.Run("current draft and sibling", func(t *testing.T) {
		f.T = t
		result := f.HTTP(ctx, "versions.delete", "alice", f.Namespace, removaltest.Version("versions", 2, 7), 200)
		f.Result(result, "versions", 2, 1)
		f.Count(ctx, "SELECT COUNT(*) FROM component_warmup_schedules WHERE report_id='versions'", 0)
		f.Count(ctx, "SELECT COUNT(*) FROM components WHERE id='versions' AND current_draft_version IS NULL AND last_version_no=2 AND etag=2", 1)
		f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id='versions' AND version_no=1 AND source_revision=7", 1)
		f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id='versions' AND version_no=2", 0)
		f.Count(ctx, "SELECT COUNT(*) FROM component_deletions WHERE component_id='versions'", 0)
	})
	t.Run("last version over MCP preserves empty component and allocation", func(t *testing.T) {
		f.T = t
		f.Result(f.MCP(ctx, "versions.delete", removaltest.Version("versions", 1, 7)), "versions", 1, 1)
		f.Count(ctx, "SELECT COUNT(*) FROM components WHERE id='versions' AND current_draft_version IS NULL AND last_version_no=2 AND etag=3", 1)
		f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id='versions'", 0)
		raw := f.HTTP(ctx, "versions.load_dql", "alice", f.Namespace, map[string]any{"reportId": "versions", "input": map[string]any{"dql": "SELECT 3 AS id"}}, 200)
		// Allocation must consume 3 even when no older versions remain.
		f.Count(ctx, "SELECT COUNT(*) FROM component_versions WHERE report_id='versions' AND version_no=3", 1)
		var imported struct {
			Version struct {
				VersionNo int `json:"versionNo"`
			} `json:"version"`
		}
		if err := json.Unmarshal(raw, &imported); err != nil || imported.Version.VersionNo != 3 {
			t.Fatalf("native import allocation: %s err=%v", raw, err)
		}
	})
}
