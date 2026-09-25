package schema

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

//go:embed schema.ddl sqlite/*.sql
var sqliteFiles embed.FS

var (
	mysqlUniqueKey         = regexp.MustCompile(`(?i)\bUNIQUE\s+KEY\s+[A-Za-z_][A-Za-z0-9_]*\s*\(`)
	mysqlTableTail         = regexp.MustCompile(`(?i)\)\s+ENGINE\s*=\s*InnoDB\s+DEFAULT\s+CHARSET\s*=\s*utf8mb4\s*;`)
	mysqlDateTimePrecision = regexp.MustCompile(`(?i)\bDATETIME\s*\(\s*\d+\s*\)`)
	sqliteTableName        = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+([A-Za-z_][A-Za-z0-9_]*)`)
)

const CanonicalVersion = 12

// ApplySQLite applies one embedded SQLite schema or fixture script.
func ApplySQLite(ctx context.Context, db *sql.DB, name string) error {
	if db == nil {
		return fmt.Errorf("SQLite database is required")
	}
	path := "sqlite/" + name + ".sql"
	if name == "studio" {
		path = "schema.ddl"
	}
	payload, err := sqliteFiles.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read SQLite script %q: %w", name, err)
	}
	if name == "studio" {
		payload = sqliteStudioDDL(payload)
	}
	if _, err = db.ExecContext(ctx, string(payload)); err != nil {
		return fmt.Errorf("apply SQLite script %q: %w", name, err)
	}
	return nil
}

// EnsureVersionTable creates the migration bookkeeping table. It is metadata
// about the canonical snapshot, not a second copy of the domain schema.
func EnsureVersionTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`)
	return err
}

func SQLiteVersion(ctx context.Context, db *sql.DB) (int, error) {
	if err := EnsureVersionTable(ctx, db); err != nil {
		return 0, err
	}
	var version int
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&version); err != nil {
		return 0, err
	}
	return version, nil
}

func SetSQLiteVersion(ctx context.Context, db *sql.DB, version int) error {
	if err := EnsureVersionTable(ctx, db); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM schema_version`); err != nil {
		return err
	}
	if version == 0 {
		return nil
	}
	_, err := db.ExecContext(ctx, `INSERT INTO schema_version(version) VALUES (?)`, version)
	return err
}

// AddSQLiteColumnFromCanonical adds one column using its definition from the
// authoritative schema.ddl. Upgrade code names only the table and column; it
// does not duplicate domain DDL.
func AddSQLiteColumnFromCanonical(ctx context.Context, db *sql.DB, table, column string) error {
	if !validSchemaIdentifier(table) || !validSchemaIdentifier(column) {
		return fmt.Errorf("invalid canonical column target %s.%s", table, column)
	}
	payload, err := sqliteFiles.ReadFile("schema.ddl")
	if err != nil {
		return err
	}
	definition, err := canonicalColumnDefinition(string(payload), table, column)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, "ALTER TABLE "+table+" ADD COLUMN "+definition)
	return err
}

// CreateSQLiteTableFromCanonical installs one table and its immediately
// following indexes from schema.ddl. Upgrade code names the table only so the
// canonical snapshot remains the sole owner of domain DDL.
func CreateSQLiteTableFromCanonical(ctx context.Context, db *sql.DB, table string) error {
	if !validSchemaIdentifier(table) {
		return fmt.Errorf("invalid canonical table target %s", table)
	}
	payload, err := sqliteFiles.ReadFile("schema.ddl")
	if err != nil {
		return err
	}
	source := string(sqliteStudioDDL(payload))
	lower := strings.ToLower(source)
	start := strings.Index(lower, "create table "+strings.ToLower(table)+" (")
	if start < 0 {
		return fmt.Errorf("canonical table %s was not found", table)
	}
	next := strings.Index(lower[start+1:], "create table ")
	end := len(source)
	if next >= 0 {
		end = start + 1 + next
	}
	if _, err = db.ExecContext(ctx, source[start:end]); err != nil {
		return fmt.Errorf("create canonical table %s: %w", table, err)
	}
	return nil
}

func canonicalColumnDefinition(ddl, table, column string) (string, error) {
	start := strings.Index(strings.ToLower(ddl), "create table "+strings.ToLower(table)+" (")
	if start < 0 {
		return "", fmt.Errorf("canonical table %s was not found", table)
	}
	body := ddl[start:]
	end := strings.Index(body, ") ENGINE")
	if end < 0 {
		return "", fmt.Errorf("canonical table %s has no end", table)
	}
	for _, line := range strings.Split(body[:end], "\n") {
		trimmed := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), ","))
		fields := strings.Fields(trimmed)
		if len(fields) > 1 && strings.EqualFold(fields[0], column) {
			return trimmed, nil
		}
	}
	return "", fmt.Errorf("canonical column %s.%s was not found", table, column)
}

func validSchemaIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for index, character := range value {
		if character == '_' || character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || index > 0 && character >= '0' && character <= '9' {
			continue
		}
		return false
	}
	return true
}

// DropSQLite removes every table declared by schema.ddl. The table list is
// derived from that file so the canonical schema remains the only domain DDL.
func DropSQLite(ctx context.Context, db *sql.DB) error {
	payload, err := sqliteFiles.ReadFile("schema.ddl")
	if err != nil {
		return err
	}
	matches := sqliteTableName.FindAllSubmatch(payload, -1)
	tables := make([]string, 0, len(matches))
	for _, match := range matches {
		tables = append(tables, string(match[1]))
	}
	sort.Slice(tables, func(i, j int) bool { return tables[i] > tables[j] })
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	defer func() { _, _ = db.ExecContext(ctx, `PRAGMA foreign_keys = ON`) }()
	for _, table := range tables {
		if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS `+table); err != nil {
			return fmt.Errorf("drop %s: %w", table, err)
		}
	}
	return SetSQLiteVersion(ctx, db, 0)
}

// sqliteStudioDDL derives SQLite test DDL from the authoritative MySQL schema.
// SQLite accepts the declared MySQL storage types; named UNIQUE KEY syntax,
// DATETIME precision (needed for go-sqlite3 time scanning), and InnoDB table
// options require normalization.
func sqliteStudioDDL(source []byte) []byte {
	result := mysqlDateTimePrecision.ReplaceAll(source, []byte("DATETIME"))
	result = mysqlUniqueKey.ReplaceAll(result, []byte("UNIQUE ("))
	return mysqlTableTail.ReplaceAll(result, []byte(");"))
}
