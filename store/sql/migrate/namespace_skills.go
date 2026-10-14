package migrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/viant/datly-studio/schema"
)

// Namespace skills have independent ownership and immutable content revisions.
// Existing component resource rows retain their original keys and ownership.
func migrateNamespaceSkills(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, table := range []string{"namespace_skills", "namespace_skill_revisions"} {
		exists, err := sqliteTableExists(ctx, tx, table)
		if err != nil {
			return err
		}
		if !exists {
			if err := schema.CreateSQLiteTableFromCanonical(ctx, tx, table); err != nil {
				return fmt.Errorf("create namespace skill storage: %w", err)
			}
		}
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM schema_version"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO schema_version(version) VALUES (?)", schema.CanonicalVersion); err != nil {
		return err
	}
	return tx.Commit()
}
