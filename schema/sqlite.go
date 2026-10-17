package schema

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	policySchema "github.com/viant/authz/component/schema"
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
	mysqlColumnCharset     = regexp.MustCompile(`(?i)\s+CHARACTER\s+SET\s+(?:ascii|utf8mb4)\s+COLLATE\s+(?:ascii_bin|utf8mb4_bin)`)
	sqliteTableName        = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([A-Za-z_][A-Za-z0-9_]*)`)
)

// CanonicalVersion identifies the current schema layout, not an upgrade sequence.
const CanonicalVersion = 23

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
		payload, err = composeSQLiteStudioDDL(payload)
		if err != nil {
			return err
		}
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

// SQLiteTableStatements returns canonical DDL for transactional table rebuilds.
func SQLiteTableStatements(table string) (string, error) {
	if !validSchemaIdentifier(table) {
		return "", fmt.Errorf("invalid canonical table target %s", table)
	}
	if sharedPolicyTable(table) {
		payload, err := policySchema.PolicyDDL("sqlite")
		if err != nil {
			return "", err
		}
		return tableStatement(payload, table)
	}
	payload, err := sqliteFiles.ReadFile("schema.ddl")
	if err != nil {
		return "", err
	}
	source := string(sqliteStudioDDL(payload))
	return tableStatement(source, table)
}

func tableStatement(source, table string) (string, error) {
	lower := strings.ToLower(source)
	start := -1
	for _, match := range sqliteTableName.FindAllStringSubmatchIndex(source, -1) {
		name := source[match[2]:match[3]]
		if strings.EqualFold(name, table) {
			start = match[0]
			break
		}
	}
	if start < 0 {
		return "", fmt.Errorf("canonical table %s was not found", table)
	}
	next := strings.Index(lower[start+1:], "create table ")
	end := len(source)
	if next >= 0 {
		end = start + 1 + next
	}
	return source[start:end], nil
}

func sharedPolicyTable(table string) bool {
	return table == "resource_policies" || table == "resource_policy_revisions"
}

func composeSQLiteStudioDDL(source []byte) ([]byte, error) {
	result := sqliteStudioDDL(source)
	policyDDL, err := policySchema.PolicyDDL("sqlite")
	if err != nil {
		return nil, err
	}
	result = append(result, []byte("\n\n"+policyDDL+"\n")...)
	return result, nil
}

// MySQLDDL composes Studio's own canonical tables with the shared Authz
// component tables. The returned script is for explicit tooling such as the
// local Endly bootstrap; no duplicate policy DDL is stored in schema.ddl.
func MySQLDDL() (string, error) {
	studioDDL, err := sqliteFiles.ReadFile("schema.ddl")
	if err != nil {
		return "", err
	}
	policyDDL, err := policySchema.PolicyDDL("mysql")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(studioDDL)) + "\n\n" + policyDDL + "\n", nil
}

// MySQLTableStatement returns one Studio-owned table definition from the
// canonical snapshot. Shared policy tables must use the Authz component's
// schema API instead.
func MySQLTableStatement(table string) (string, error) {
	if !validSchemaIdentifier(table) || sharedPolicyTable(table) {
		return "", fmt.Errorf("invalid Studio-owned MySQL table %s", table)
	}
	payload, err := sqliteFiles.ReadFile("schema.ddl")
	if err != nil {
		return "", err
	}
	return tableStatement(string(payload), table)
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
	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS sqlx_sequence_reservations`); err != nil {
		return fmt.Errorf("drop SQLX sequence ledger: %w", err)
	}
	_, err = db.ExecContext(ctx, `DROP TABLE IF EXISTS schema_version`)
	return err
}

// sqliteStudioDDL derives SQLite test DDL from the authoritative MySQL schema.
// SQLite accepts the declared MySQL storage types; named UNIQUE KEY syntax,
// DATETIME precision (needed for go-sqlite3 time scanning), and InnoDB table
// options require normalization.
func sqliteStudioDDL(source []byte) []byte {
	result := mysqlDateTimePrecision.ReplaceAll(source, []byte("DATETIME"))
	result = mysqlColumnCharset.ReplaceAll(result, nil)
	result = mysqlUniqueKey.ReplaceAll(result, []byte("UNIQUE ("))
	return mysqlTableTail.ReplaceAll(result, []byte(");"))
}
