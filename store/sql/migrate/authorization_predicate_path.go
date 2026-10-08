package migrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/viant/datly-studio/schema"
)

// migrateAuthorizationPredicatePackagePath advances the SQLite schema after
// the MySQL package/type key receives exact binary collations. SQLite's
// canonical unique key already uses its default BINARY collation, so this
// migration validates legacy data and advances the version without rewriting
// rows or changing the key.
func migrateAuthorizationPredicatePackagePath(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	exists, err := sqliteTableExists(ctx, tx, "authorization_predicates")
	if err != nil {
		return err
	}
	if !exists {
		if err := schema.CreateSQLiteTableFromCanonical(ctx, tx, "authorization_predicates"); err != nil {
			return fmt.Errorf("create canonical authorization_predicates for schema version 22: %w", err)
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT package_path FROM authorization_predicates`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			rows.Close()
			return err
		}
		if len(path) > 1000 {
			rows.Close()
			return fmt.Errorf("authorization predicate package_path exceeds the canonical length")
		}
		for _, character := range path {
			if character > 127 {
				rows.Close()
				return fmt.Errorf("authorization predicate package_path contains non-ASCII data")
			}
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM schema_version"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO schema_version(version) VALUES (?)", schema.CanonicalVersion); err != nil {
		return err
	}
	return tx.Commit()
}
