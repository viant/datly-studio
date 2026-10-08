package migrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/viant/datly-studio/schema"
)

// migratePolicyNamespaceBindings keeps Studio namespace ownership beside the
// shared authorization policy schema. The legacy namespace_id columns are
// preserved; no shared policy table is rebuilt or rewritten here.
func migratePolicyNamespaceBindings(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	exists, err := sqliteTableExists(ctx, tx, "resource_policy_namespace_bindings")
	if err != nil {
		return err
	}
	if !exists {
		if err := schema.CreateSQLiteTableFromCanonical(ctx, tx, "resource_policy_namespace_bindings"); err != nil {
			return fmt.Errorf("create policy namespace bindings: %w", err)
		}
	}
	for _, source := range []struct {
		table          string
		policyRevision string
	}{{table: "resource_policies", policyRevision: "0"}, {table: "resource_policy_revisions", policyRevision: "src.revision"}} {
		tableExists, err := sqliteTableExists(ctx, tx, source.table)
		if err != nil {
			return err
		}
		if !tableExists {
			return fmt.Errorf("shared policy table %s is missing", source.table)
		}
		hasNamespace, err := sqliteColumnExists(ctx, tx, source.table, "namespace_id")
		if err != nil {
			return err
		}
		if !hasNamespace {
			continue
		}
		for _, column := range []string{"created_at", "created_by", "updated_at", "updated_by"} {
			hasColumn, err := sqliteColumnExists(ctx, tx, source.table, column)
			if err != nil {
				return err
			}
			if !hasColumn {
				return fmt.Errorf("shared policy table %s lacks canonical audit column %s", source.table, column)
			}
		}
		insert := `INSERT OR IGNORE INTO resource_policy_namespace_bindings
			(tenant_id,resource_kind,resource_id,resource_version,policy_revision,namespace_id,created_at,created_by,updated_at,updated_by)
			SELECT tenant_id,resource_kind,resource_id,resource_version,` + source.policyRevision + `,namespace_id,created_at,created_by,updated_at,updated_by
			FROM ` + source.table + ` AS src WHERE COALESCE(namespace_id,'')<>''`
		if _, err := tx.ExecContext(ctx, insert); err != nil {
			return fmt.Errorf("copy namespace bindings from %s: %w", source.table, err)
		}
		verify := `SELECT COUNT(*) FROM ` + source.table + ` src WHERE COALESCE(src.namespace_id,'')<>'' AND NOT EXISTS (
			SELECT 1 FROM resource_policy_namespace_bindings b
			WHERE b.tenant_id=src.tenant_id AND b.resource_kind=src.resource_kind AND b.resource_id=src.resource_id
			AND b.resource_version=src.resource_version AND b.policy_revision=` + source.policyRevision + ` AND b.namespace_id=src.namespace_id)`
		var missing int
		if err := tx.QueryRowContext(ctx, verify).Scan(&missing); err != nil {
			return err
		}
		if missing != 0 {
			return fmt.Errorf("namespace binding copy from %s left %d records unmapped", source.table, missing)
		}
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM schema_version"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO schema_version(version) VALUES (?)", schema.PolicyNamespaceVersion); err != nil {
		return err
	}
	return tx.Commit()
}
