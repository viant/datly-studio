package schema

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// verifyRemovalConstraints checks the constraints which distinguish retained
// history from the old cascading layout. It never repairs an existing store.
func verifyRemovalConstraints(ctx context.Context, db *sql.DB, driver string) error {
	query := `SELECT name, type, "notnull", dflt_value, pk FROM pragma_table_info('components') WHERE name='last_version_no'`
	if driver == "mysql" {
		query = `SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE='NO', COLUMN_DEFAULT, COLUMN_KEY='PRI' FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='components' AND COLUMN_NAME='last_version_no'`
	}
	var name, typ string
	var notNull, primary int
	var defaultValue sql.NullString
	if err := db.QueryRowContext(ctx, query).Scan(&name, &typ, &notNull, &defaultValue, &primary); err != nil || !strings.EqualFold(typ, "INT") || notNull != 1 || !defaultValue.Valid || defaultValue.String != "0" {
		return fmt.Errorf("canonical components.last_version_no INT NOT NULL DEFAULT 0 is required")
	}
	query = `SELECT name, type, "notnull", pk FROM pragma_table_info('component_deletions')`
	if driver == "mysql" {
		query = `SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE='NO', COLUMN_KEY='PRI' FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='component_deletions'`
	}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	expected := map[string]string{"component_id": "varchar(64)", "namespace_id": "varchar(64)", "owner_id": "varchar(128)", "deleted_at": "datetime", "deleted_by": "varchar(128)"}
	for rows.Next() {
		if err = rows.Scan(&name, &typ, &notNull, &primary); err != nil {
			rows.Close()
			return err
		}
		want, ok := expected[name]
		actual := strings.ToLower(typ)
		if driver == "mysql" && name == "deleted_at" {
			want = "datetime(6)"
		}
		wantPrimary := 0
		if name == "component_id" {
			wantPrimary = 1
		}
		if !ok || actual != want || notNull != 1 || primary != wantPrimary {
			rows.Close()
			return fmt.Errorf("canonical component_deletions receipt constraint is required for %s", name)
		}
		delete(expected, name)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if len(expected) != 0 {
		return fmt.Errorf("canonical component_deletions receipt is required")
	}
	for _, table := range []string{"component_deletions", "component_publication_events"} {
		query = `SELECT "table", "from", "to", on_delete FROM pragma_foreign_key_list('` + table + `')`
		if driver == "mysql" {
			query = `SELECT k.REFERENCED_TABLE_NAME,k.COLUMN_NAME,k.REFERENCED_COLUMN_NAME,r.DELETE_RULE FROM information_schema.KEY_COLUMN_USAGE k JOIN information_schema.REFERENTIAL_CONSTRAINTS r ON r.CONSTRAINT_SCHEMA=k.CONSTRAINT_SCHEMA AND r.TABLE_NAME=k.TABLE_NAME AND r.CONSTRAINT_NAME=k.CONSTRAINT_NAME WHERE k.TABLE_SCHEMA=DATABASE() AND k.TABLE_NAME='` + table + `' AND k.REFERENCED_TABLE_NAME IS NOT NULL`
		}
		rows, err = db.QueryContext(ctx, query)
		if err != nil {
			return err
		}
		count := 0
		for rows.Next() {
			var target, from, to, rule string
			if err = rows.Scan(&target, &from, &to, &rule); err != nil {
				rows.Close()
				return err
			}
			count++
			if table == "component_deletions" || target != "runtime_generations" || from != "generation_no" || to != "generation_no" || (rule != "NO ACTION" && rule != "RESTRICT") {
				rows.Close()
				return fmt.Errorf("canonical %s foreign keys are required", table)
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if table == "component_publication_events" && count != 1 {
			return fmt.Errorf("canonical publication event generation foreign key is required")
		}
	}
	return nil
}
