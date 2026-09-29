package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/viant/datly-studio/schema"
	"regexp"
	"strings"
)

// Rebuild without renaming the old parent: SQLite must retain every child's
// foreign-key target and all component IDs. Foreign-key state is restored on
// the same pinned connection after the transaction completes.
func migrateComponentSlugKeys(ctx context.Context, db *sql.DB) (result error) {
	connection, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer connection.Close()
	var foreignKeys int
	if err = connection.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		return err
	}
	if _, err = connection.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); err != nil {
		return err
	}
	defer func() {
		_, err := connection.ExecContext(context.WithoutCancel(ctx), fmt.Sprintf("PRAGMA foreign_keys=%d", foreignKeys))
		result = errors.Join(result, err)
	}()
	tx, err := connection.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	exists, err := sqliteTableExists(ctx, tx, "components")
	if err != nil {
		return err
	}
	if !exists {
		if err = schema.CreateSQLiteTableFromCanonical(ctx, tx, "components"); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE schema_version SET version=?", schema.CanonicalVersion); err != nil {
			return err
		}
		return tx.Commit()
	}
	var invalid int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM components WHERE namespace_id=''").Scan(&invalid); err != nil {
		return err
	}
	if invalid > 0 {
		return fmt.Errorf("%d components have no namespace ownership", invalid)
	}
	var currentDDL string
	if err = tx.QueryRowContext(ctx, "SELECT sql FROM sqlite_master WHERE type='table' AND name='components'").Scan(&currentDDL); err != nil {
		return err
	}
	globalSlug := regexp.MustCompile(`(?i)UNIQUE\s*\(\s*slug\s*\)`)
	if !globalSlug.MatchString(currentDDL) {
		if _, err = tx.ExecContext(ctx, "UPDATE schema_version SET version=?", schema.CanonicalVersion); err != nil {
			return err
		}
		return tx.Commit()
	}
	indexes, err := schema.SQLiteTableStatements("components")
	if err != nil {
		return err
	}
	statements := strings.Split(indexes, ";")
	create := globalSlug.ReplaceAllString(currentDDL, "UNIQUE (namespace_id, slug)")
	create = regexp.MustCompile(`(?i)^CREATE TABLE\s+(?:"components"|components)\s*\(`).ReplaceAllString(create, "CREATE TABLE components_next (")
	if _, err = tx.ExecContext(ctx, create); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, "PRAGMA table_info('components')")
	if err != nil {
		return err
	}
	var columns []string
	for rows.Next() {
		var cid, notNull, pk int
		var name, kind string
		var defaultValue any
		if err = rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &pk); err != nil {
			rows.Close()
			return err
		}
		columns = append(columns, `"`+name+`"`)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	names := strings.Join(columns, ",")
	if _, err = tx.ExecContext(ctx, "INSERT INTO components_next ("+names+") SELECT "+names+" FROM components"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DROP TABLE components"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "ALTER TABLE components_next RENAME TO components"); err != nil {
		return err
	}
	for _, statement := range statements[1:] {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err = tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE schema_version SET version=?", schema.CanonicalVersion); err != nil {
		return err
	}
	return tx.Commit()
}
