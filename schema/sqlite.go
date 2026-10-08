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

const (
	PrePolicyNamespaceVersion = 20
	PolicyNamespaceVersion    = 21
	CanonicalVersion          = 22
)

// EnsureSQLiteSequenceLedger installs SQLX's write-intent table before a
// publication transaction begins. Creating it inside a deferred transaction
// can fail after earlier catalog reads have established a SQLite snapshot.
func EnsureSQLiteSequenceLedger(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS sqlx_sequence_reservations (
		table_name TEXT PRIMARY KEY,
		value INTEGER NOT NULL CHECK(value >= 0)
	)`)
	return err
}

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
	if name == "studio" {
		if err := EnsureSQLiteSequenceLedger(ctx, db); err != nil {
			return fmt.Errorf("prepare SQLite sequence ledger: %w", err)
		}
		if err := policySchema.CreatePolicies(ctx, db, "sqlite"); err != nil {
			return fmt.Errorf("initialize shared policy schema: %w", err)
		}
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
func AddSQLiteColumnFromCanonical(ctx context.Context, executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, table, column string) error {
	if !validSchemaIdentifier(table) || !validSchemaIdentifier(column) {
		return fmt.Errorf("invalid canonical column target %s.%s", table, column)
	}
	if sharedPolicyTable(table) {
		definition, err := policySchema.PolicyColumnDefinition("sqlite", table, column)
		if err != nil {
			return err
		}
		_, err = executor.ExecContext(ctx, "ALTER TABLE "+table+" ADD COLUMN "+definition)
		return err
	}
	payload, err := sqliteFiles.ReadFile("schema.ddl")
	if err != nil {
		return err
	}
	definition, err := canonicalColumnDefinition(string(payload), table, column)
	if err != nil {
		return err
	}
	_, err = executor.ExecContext(ctx, "ALTER TABLE "+table+" ADD COLUMN "+definition)
	return err
}

// CreateSQLiteTableFromCanonical installs one table and its immediately
// following indexes from schema.ddl. Upgrade code names the table only so the
// canonical snapshot remains the sole owner of domain DDL.
func CreateSQLiteTableFromCanonical(ctx context.Context, executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, table string) error {
	if !validSchemaIdentifier(table) {
		return fmt.Errorf("invalid canonical table target %s", table)
	}
	statement, err := SQLiteTableStatements(table)
	if err != nil {
		return err
	}
	if _, err = executor.ExecContext(ctx, statement); err != nil {
		return fmt.Errorf("create canonical table %s: %w", table, err)
	}
	return nil
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

func canonicalColumnDefinition(ddl, table, column string) (string, error) {
	if sharedPolicyTable(table) {
		return policySchema.PolicyColumnDefinition("sqlite", table, column)
	}
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
