package schema

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

func TestCanonicalAuditColumnInventory(t *testing.T) {
	t.Parallel()

	db := openSQLite(t, "canonical_audit_inventory")
	defer db.Close()
	if err := ApplySQLite(context.Background(), db, "studio"); err != nil {
		t.Fatal(err)
	}

	auditColumns := []string{"created_at", "created_by", "updated_at", "updated_by"}
	requiredAuditTables := map[string]struct{}{
		"resource_policy_heads":     {},
		"resource_policy_revisions": {},
		"resource_namespace_claims": {},
		"report_warmup_runs":        {},
	}

	// Temporary migration backlog, not permanent exemptions. Every table in
	// this list still needs explicit created/updated actor semantics; a newly
	// added table must already be audited or deliberately join the backlog.
	legacyBacklog := map[string]struct{}{
		"connectors": {}, "namespaces": {}, "authorization_predicates": {},
		"reports": {}, "report_versions": {}, "report_views": {},
		"report_fields": {}, "report_parameters": {}, "report_predicates": {},
		"report_cube_configs": {}, "report_mcp_exposures": {},
		"report_resource_files": {}, "report_resource_folders": {},
		"report_skill_roots": {}, "runtime_generations": {},
		"bff_sessions": {}, "report_publications": {},
		"report_publication_events": {}, "report_acl": {},
	}

	rows, err := db.QueryContext(context.Background(), `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	tables := make([]string, 0)
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}

	seenAuditTables := make(map[string]bool, len(requiredAuditTables))
	for _, table := range tables {
		columns, err := canonicalTableColumns(t, db, table)
		if err != nil {
			t.Fatal(err)
		}
		missing := make([]string, 0)
		for _, column := range auditColumns {
			if !columns[column] {
				missing = append(missing, column)
			}
		}

		if _, required := requiredAuditTables[table]; required {
			seenAuditTables[table] = true
			if len(missing) != 0 {
				t.Errorf("canonical table %s is missing required audit columns: %s", table, strings.Join(missing, ", "))
			}
			continue
		}
		if _, backlogged := legacyBacklog[table]; backlogged {
			if len(missing) == 0 {
				t.Errorf("canonical table %s is audited; remove it from the migration backlog", table)
			}
			continue
		}
		t.Errorf("canonical table %s is neither fully audited nor documented in the migration backlog (missing: %s)", table, strings.Join(missing, ", "))
	}
	for table := range requiredAuditTables {
		if !seenAuditTables[table] {
			t.Errorf("required canonical audit table %s was not found", table)
		}
	}
}

func canonicalTableColumns(t *testing.T, db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, table string) (map[string]bool, error) {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), "PRAGMA table_info("+table+")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return nil, err
		}
		columns[name] = true
	}
	return columns, rows.Err()
}
