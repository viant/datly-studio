package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/viant/datly-studio/schema"
)

func migrateNamespaceClaimKeys(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = rebuildNamespaceClaimKeys(ctx, tx); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE schema_version SET version=?", schema.CanonicalVersion); err != nil {
		return err
	}
	return tx.Commit()
}
func rebuildNamespaceClaimKeys(ctx context.Context, tx *sql.Tx) error {
	exists, err := sqliteTableExists(ctx, tx, "resource_namespace_claims")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	var invalid int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM resource_namespace_claims claim LEFT JOIN components c ON c.id=claim.report_id
 WHERE c.namespace_id IS NULL OR c.namespace_id='' OR (claim.namespace_id<>'' AND claim.namespace_id<>c.namespace_id)`).Scan(&invalid); err != nil {
		return err
	}
	if invalid > 0 {
		return fmt.Errorf("namespace claims contain %d missing or inconsistent component owners", invalid)
	}
	if _, err = tx.ExecContext(ctx, "ALTER TABLE resource_namespace_claims RENAME TO resource_namespace_claims_previous"); err != nil {
		return err
	}
	if err = schema.CreateSQLiteTableFromCanonical(ctx, tx, "resource_namespace_claims"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO resource_namespace_claims(namespace_id,namespace,report_id,created_at,created_by,updated_at,updated_by)
 SELECT c.namespace_id,claim.namespace,claim.report_id,claim.created_at,claim.created_by,claim.updated_at,claim.updated_by
 FROM resource_namespace_claims_previous claim JOIN components c ON c.id=claim.report_id`); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "DROP TABLE resource_namespace_claims_previous")
	return err
}
