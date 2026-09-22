package schema

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCanonicalStudioSchemaInventory(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}
	if err := ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{
		"connectors":                {"name", "dsn_template", "last_test_status"},
		"namespaces":                {"owner_id", "name", "title", "status", "etag"},
		"authorization_predicates":  {"name", "package_path", "type_name", "owner_id", "status", "etag"},
		"reports":                   {"id", "namespace", "default_connector_name", "component_scope", "component_name", "current_draft_version"},
		"report_versions":           {"report_id", "authoring_mode", "authored_sql", "authored_dql", "generated_dql", "spec_hash"},
		"report_views":              {"report_id", "view_id", "source_kind"},
		"report_fields":             {"view_id", "field_name", "go_type"},
		"report_parameters":         {"parameter_id", "name", "activation_json"},
		"report_predicates":         {"parameter_id", "predicate_index", "apply_when_absent"},
		"report_cube_configs":       {"cube_enabled", "compose_enabled"},
		"report_mcp_exposures":      {"route_id", "route_path", "kind"},
		"report_resource_files":     {"resource_path", "content_sha256"},
		"report_resource_folders":   {"folder_id", "root_path", "uri_prefix"},
		"report_skill_roots":        {"skill_id", "folder_id", "skill_root"},
		"runtime_generations":       {"generation_no", "build_manifest_json", "source_revision"},
		"bff_sessions":              {"session_id_hash", "subject_id", "payload_ciphertext", "expires_at_unix"},
		"report_publications":       {"active_version_no", "desired_version_no", "desired_generation", "publication_status"},
		"report_publication_events": {"event_id", "report_id", "owner_id", "operation", "version_no", "generation_no", "status", "requested_by", "failure_message", "occurred_at"},
		"report_acl":                {"subject_type", "subject_id", "can_use_dql", "etag"},
	}
	for table, columns := range want {
		actual := map[string]bool{}
		rows, err := db.QueryContext(ctx, "PRAGMA table_info("+table+")")
		if err != nil {
			t.Fatalf("PRAGMA table_info(%s): %v", table, err)
		}
		for rows.Next() {
			var cid int
			var name, typ string
			var notNull, pk int
			var defaultValue any
			if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			actual[name] = true
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
		for _, column := range columns {
			if !actual[column] {
				t.Errorf("canonical table %s is missing column %s", table, column)
			}
		}
	}

	for table, removedColumns := range map[string][]string{
		"reports":         {"mode", "connector_name", "mcp_enabled", "mcp_tool_name", "mcp_description"},
		"report_versions": {"input_mode", "original_sql", "core_dql", "effective_dql", "generated_sql", "column_overrides_json", "permissions_json", "cube_json"},
	} {
		for _, removed := range removedColumns {
			var count int
			if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM pragma_table_info(?) WHERE name = ?`, table, removed).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Errorf("removed catalog column %q is present on %s", removed, table)
			}
		}
	}
}
