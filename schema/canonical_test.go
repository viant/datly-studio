package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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
		"connectors":                         {"name", "dsn_template", "last_test_status"},
		"namespaces":                         {"owner_id", "name", "title", "status", "etag"},
		"authorization_predicates":           {"name", "package_path", "type_name", "owner_id", "status", "etag"},
		"components":                         {"id", "namespace", "default_connector_name", "component_scope", "component_name", "current_draft_version"},
		"component_versions":                 {"report_id", "authoring_mode", "authored_sql", "authored_dql", "generated_dql", "spec_hash"},
		"component_views":                    {"report_id", "view_id", "source_kind"},
		"component_fields":                   {"view_id", "field_name", "go_type"},
		"component_parameters":               {"parameter_id", "name", "activation_json"},
		"component_predicates":               {"parameter_id", "predicate_index", "apply_when_absent"},
		"component_cube_configs":             {"cube_enabled", "compose_enabled"},
		"component_mcp_exposures":            {"route_id", "route_path", "kind"},
		"component_resource_files":           {"resource_path", "content_sha256"},
		"component_resource_folders":         {"folder_id", "root_path", "uri_prefix"},
		"resource_namespace_claims":          {"namespace", "report_id", "created_at", "created_by", "updated_at", "updated_by"},
		"resource_policy_namespace_bindings": {"tenant_id", "resource_kind", "resource_id", "resource_version", "policy_revision", "namespace_id", "created_at", "created_by", "updated_at", "updated_by"},
		"component_skill_roots":              {"skill_id", "folder_id", "skill_root"},
		"runtime_generations":                {"generation_no", "build_manifest_json", "source_revision"},
		"bff_sessions":                       {"session_id_hash", "subject_id", "payload_ciphertext", "expires_at_unix"},
		"component_publications":             {"active_version_no", "desired_version_no", "desired_generation", "publication_status"},
		"component_publication_events":       {"event_id", "report_id", "owner_id", "operation", "version_no", "generation_no", "status", "requested_by", "failure_message", "occurred_at"},
		"component_acl":                      {"subject_type", "subject_id", "can_use_dql", "etag"},
	}
	wantPrimaryKeys := map[string][]string{
		"resource_policy_namespace_bindings": {"tenant_id", "resource_kind", "resource_id", "resource_version", "policy_revision"},
	}
	for table, columns := range want {
		actual := map[string]bool{}
		primaryKey := map[int]string{}
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
			if pk > 0 {
				primaryKey[pk] = name
			}
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
		for _, column := range columns {
			if !actual[column] {
				t.Errorf("canonical table %s is missing column %s", table, column)
			}
		}
		if expected, ok := wantPrimaryKeys[table]; ok {
			got := make([]string, len(primaryKey))
			for index := 1; index <= len(primaryKey); index++ {
				got[index-1] = primaryKey[index]
			}
			if strings.Join(got, ",") != strings.Join(expected, ",") {
				t.Errorf("canonical table %s primary key=%v want=%v", table, got, expected)
			}
		}
	}

	for table, removedColumns := range map[string][]string{
		"components":         {"mode", "connector_name", "mcp_enabled", "mcp_tool_name", "mcp_description"},
		"component_versions": {"input_mode", "original_sql", "core_dql", "effective_dql", "generated_sql", "column_overrides_json", "permissions_json", "cube_json"},
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
