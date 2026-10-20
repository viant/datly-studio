package schema

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Initialize creates the complete canonical schema in an empty store. Existing
// stores are verified without ALTER, backfill, repair, or version upgrades.
func Initialize(ctx context.Context, db *sql.DB, driver string) error {
	if ctx == nil || db == nil {
		return errors.New("schema initialization requires a context and database")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	ddl, err := canonicalDDL(driver)
	if err != nil {
		return err
	}
	var tables int
	query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name NOT IN ('resource_policies','resource_policy_revisions')"
	if driver == "mysql" {
		query = "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_type='BASE TABLE' AND table_name NOT IN ('resource_policies','resource_policy_revisions')"
	}
	if err := db.QueryRowContext(ctx, query).Scan(&tables); err != nil {
		return err
	}
	if tables == 0 {
		if driver == "sqlite" {
			transaction, err := db.BeginTx(ctx, nil)
			if err != nil {
				return err
			}
			defer transaction.Rollback()
			if _, err = transaction.ExecContext(ctx, ddl); err != nil {
				return fmt.Errorf("create canonical Studio schema: %w", err)
			}
			if err = transaction.Commit(); err != nil {
				return err
			}
		} else {
			// MySQL DDL commits implicitly; execute the canonical script statement by
			// statement so Store does not require the multiStatements DSN option.
			for _, statement := range mysqlStatements(ddl) {
				if strings.TrimSpace(statement) == "" {
					continue
				}
				if _, err := db.ExecContext(ctx, statement); err != nil {
					return fmt.Errorf("create canonical Studio schema: %w", err)
				}
			}
		}
	}
	return Verify(ctx, db, driver)
}

// Verify checks existing storage against the canonical table/column/index
// inventory. It performs reads only and never repairs storage. Historical
// schema_version rows do not determine whether the actual layout is usable.
func Verify(ctx context.Context, db *sql.DB, driver string) error {
	if driver == "sqlite" {
		var definition string
		if err := db.QueryRowContext(ctx, "SELECT sql FROM sqlite_master WHERE type='table' AND name='runtime_generations'").Scan(&definition); err != nil || !strings.Contains(strings.ToUpper(definition), "AUTOINCREMENT") {
			return errors.New("canonical runtime generation AUTOINCREMENT identity is required")
		}
	}
	if driver == "mysql" {
		var extra string
		if err := db.QueryRowContext(ctx, "SELECT EXTRA FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='runtime_generations' AND COLUMN_NAME='generation_no'").Scan(&extra); err != nil || !strings.Contains(strings.ToLower(extra), "auto_increment") {
			return errors.New("canonical runtime generation identity is required")
		}
	}
	ddl, err := canonicalDDL(driver)
	if err != nil {
		return err
	}
	for _, table := range sqliteTableName.FindAllStringSubmatch(ddl, -1) {
		name := table[1]
		body, err := tableStatement(ddl, name)
		if err != nil {
			return err
		}
		columns := []string{}
		for _, line := range strings.Split(body, "\n")[1:] {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, ")") {
				break
			}
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			column := strings.Trim(fields[0], "`\"")
			if column == strings.ToUpper(column) || !validSchemaIdentifier(column) {
				continue
			}
			columns = append(columns, "`"+column+"`")
		}
		if len(columns) == 0 {
			return fmt.Errorf("canonical table %s has no column inventory", name)
		}
		rows, err := db.QueryContext(ctx, "SELECT "+strings.Join(columns, ",")+" FROM `"+name+"` WHERE 1=0")
		if err != nil {
			return fmt.Errorf("Studio store does not match canonical table %s", name)
		}
		if err = rows.Close(); err != nil {
			return err
		}
	}
	for _, index := range canonicalIndexes.FindAllStringSubmatch(ddl, -1) {
		var count int
		query := "SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?"
		if driver == "mysql" {
			query = "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND index_name=?"
		}
		if err := db.QueryRowContext(ctx, query, index[1]).Scan(&count); err != nil || count < 1 {
			return fmt.Errorf("Studio store is missing canonical index %s", index[1])
		}
	}
	return nil
}

var canonicalIndexes = regexp.MustCompile(`(?i)CREATE\s+(?:UNIQUE\s+)?INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?([A-Za-z_][A-Za-z0-9_]*)`)

func canonicalDDL(driver string) (string, error) {
	if driver == "mysql" {
		return MySQLDDL()
	}
	if driver != "sqlite" {
		return "", errors.New("Studio schema supports sqlite or mysql")
	}
	source, err := sqliteFiles.ReadFile("schema.ddl")
	if err != nil {
		return "", err
	}
	ddl, err := composeSQLiteStudioDDL(source)
	return string(ddl), err
}

// MySQLStatements is the canonical creation script as individual executable
// statements. Full-line comments are removed before delimiter splitting so
// punctuation in prose cannot become a SQL command.
func MySQLStatements() ([]string, error) {
	ddl, err := canonicalDDL("mysql")
	if err != nil {
		return nil, err
	}
	return mysqlStatements(ddl), nil
}
func mysqlStatements(ddl string) []string {
	lines := strings.Split(ddl, "\n")
	kept := lines[:0]
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		kept = append(kept, line)
	}
	result := []string{}
	for _, statement := range strings.Split(strings.Join(kept, "\n"), ";") {
		if statement = strings.TrimSpace(statement); statement != "" {
			result = append(result, statement)
		}
	}
	return result
}
