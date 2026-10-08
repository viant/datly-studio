package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/viant/datly-studio/schema"
)

// MigrateAuthorizationPredicateKeyMySQL applies the exact full-length MySQL
// key contract for existing authorization predicate catalogs. Every value is
// preflighted before ALTER TABLE so non-ASCII module paths or overlength names
// fail without changing the table.
func MigrateAuthorizationPredicateKeyMySQL(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("MySQL database is required")
	}
	exists, err := mysqlTableExists(ctx, db, "authorization_predicates")
	if err != nil {
		return err
	}
	if !exists {
		ddl, err := schema.MySQLTableStatement("authorization_predicates")
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("create canonical authorization_predicates: %w", err)
		}
	}
	for _, column := range []string{"package_path", "type_name"} {
		exists, err := mysqlColumnExists(ctx, db, "authorization_predicates", column)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("authorization_predicates lacks key column %s", column)
		}
	}
	rows, err := db.QueryContext(ctx, `SELECT package_path,type_name FROM authorization_predicates`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var packagePath, typeName string
		if err := rows.Scan(&packagePath, &typeName); err != nil {
			rows.Close()
			return err
		}
		if len(packagePath) > 1000 || !isASCII(packagePath) {
			rows.Close()
			return fmt.Errorf("authorization predicate package_path must fit 1000 ASCII bytes")
		}
		if utf8.RuneCountInString(typeName) > 300 {
			rows.Close()
			return fmt.Errorf("authorization predicate type_name exceeds 300 characters")
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	fullIndex, indexExists, err := mysqlPredicateKeyIndex(ctx, db)
	if err != nil {
		return err
	}
	if !fullIndex {
		var duplicates int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM (
			SELECT BINARY package_path,BINARY type_name FROM authorization_predicates
			GROUP BY BINARY package_path,BINARY type_name HAVING COUNT(*)>1) duplicate_keys`).Scan(&duplicates); err != nil {
			return err
		}
		if duplicates != 0 {
			return fmt.Errorf("authorization predicate exact key contains duplicate rows")
		}
	}
	packagePath, err := mysqlColumnCollation(ctx, db, "package_path")
	if err != nil {
		return err
	}
	typeName, err := mysqlColumnCollation(ctx, db, "type_name")
	if err != nil {
		return err
	}
	var clauses []string
	if packagePath.characterSet != "ascii" || packagePath.collation != "ascii_bin" || packagePath.length != 1000 {
		clauses = append(clauses, "MODIFY COLUMN package_path VARCHAR(1000) CHARACTER SET ascii COLLATE ascii_bin NOT NULL")
	}
	if typeName.characterSet != "utf8mb4" || typeName.collation != "utf8mb4_bin" || typeName.length != 300 {
		clauses = append(clauses, "MODIFY COLUMN type_name VARCHAR(300) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL")
	}
	if indexExists && !fullIndex {
		clauses = append(clauses, "DROP INDEX uq_authorization_predicate_type")
	}
	if !fullIndex {
		clauses = append(clauses, "ADD UNIQUE KEY uq_authorization_predicate_type (package_path,type_name)")
	}
	if len(clauses) > 0 {
		if _, err := db.ExecContext(ctx, "ALTER TABLE authorization_predicates "+strings.Join(clauses, ",")); err != nil {
			return fmt.Errorf("apply exact authorization predicate key: %w", err)
		}
	}
	fullIndex, _, err = mysqlPredicateKeyIndex(ctx, db)
	if err != nil {
		return err
	}
	packagePath, err = mysqlColumnCollation(ctx, db, "package_path")
	if err != nil {
		return err
	}
	typeName, err = mysqlColumnCollation(ctx, db, "type_name")
	if err != nil {
		return err
	}
	if !fullIndex || packagePath.characterSet != "ascii" || packagePath.collation != "ascii_bin" || packagePath.length != 1000 || typeName.characterSet != "utf8mb4" || typeName.collation != "utf8mb4_bin" || typeName.length != 300 {
		return fmt.Errorf("authorization predicate key is not full-length and binary")
	}
	return nil
}

type mysqlColumnKey struct {
	characterSet string
	collation    string
	length       int64
}

func mysqlColumnCollation(ctx context.Context, db *sql.DB, column string) (mysqlColumnKey, error) {
	var result mysqlColumnKey
	err := db.QueryRowContext(ctx, `SELECT character_set_name,collation_name,character_maximum_length
		FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='authorization_predicates' AND column_name=?`, column).
		Scan(&result.characterSet, &result.collation, &result.length)
	return result, err
}

func mysqlPredicateKeyIndex(ctx context.Context, db *sql.DB) (full bool, exists bool, resultErr error) {
	rows, err := db.QueryContext(ctx, `SELECT non_unique,seq_in_index,column_name,sub_part
		FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='authorization_predicates'
		AND index_name='uq_authorization_predicate_type' ORDER BY seq_in_index`)
	if err != nil {
		return false, false, err
	}
	type item struct {
		nonUnique int
		sequence  int
		column    string
		prefix    sql.NullInt64
	}
	var items []item
	for rows.Next() {
		var value item
		if err := rows.Scan(&value.nonUnique, &value.sequence, &value.column, &value.prefix); err != nil {
			rows.Close()
			return false, false, err
		}
		items = append(items, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return false, false, err
	}
	if err := rows.Close(); err != nil {
		return false, false, err
	}
	if len(items) == 0 {
		return false, false, nil
	}
	exists = true
	full = len(items) == 2 && items[0].nonUnique == 0 && items[1].nonUnique == 0 && items[0].sequence == 1 && items[1].sequence == 2 && items[0].column == "package_path" && items[1].column == "type_name" && !items[0].prefix.Valid && !items[1].prefix.Valid
	return full, exists, nil
}

func isASCII(value string) bool {
	for _, character := range value {
		if character > 127 {
			return false
		}
	}
	return true
}
