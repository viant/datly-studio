package datatest

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/viant/datly-studio/schema"
)

// Row is one data-driven fixture or expected query row.
type Row map[string]any

// Table is an ordered fixture table. Dataset order is preserved for FK-safe hydration.
type Table struct {
	Name string `json:"name"`
	Rows []Row  `json:"rows"`
}

// OpenSQLite opens an isolated in-memory database and applies the requested
// canonical schema scripts.
func OpenSQLite(t testing.TB, name string, scripts ...string) *sql.DB {
	t.Helper()
	dsnName := regexp.MustCompile(`[^a-zA-Z0-9_]+`).ReplaceAllString(t.Name()+"_"+name, "_")
	db, err := sql.Open("sqlite3", "file:"+dsnName+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open SQLite %s: %v", name, err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err = db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable SQLite foreign keys: %v", err)
	}
	ctx := context.Background()
	for _, script := range scripts {
		if err = schema.ApplySQLite(ctx, db, script); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

// Hydrate inserts a deterministic dataset in one transaction. Column ordering
// is derived from sorted fixture keys, so map iteration cannot change SQL.
func Hydrate(ctx context.Context, db *sql.DB, dataset ...Table) error {
	if db == nil {
		return fmt.Errorf("hydrate database is required")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, table := range dataset {
		if _, err = validateFixtureTable(ctx, tx, table); err != nil {
			return err
		}
		name, err := quoteIdentifier(table.Name)
		if err != nil {
			return err
		}
		for rowIndex, row := range table.Rows {
			columns := make([]string, 0, len(row))
			for column := range row {
				columns = append(columns, column)
			}
			sort.Strings(columns)
			if len(columns) == 0 {
				return fmt.Errorf("hydrate %s row %d has no columns", table.Name, rowIndex)
			}
			quoted := make([]string, len(columns))
			values := make([]any, len(columns))
			for i, column := range columns {
				quoted[i], err = quoteIdentifier(column)
				if err != nil {
					return err
				}
				values[i], err = normalizeFixtureValue(row[column])
				if err != nil {
					return fmt.Errorf("hydrate %s row %d column %s: %w", table.Name, rowIndex, column, err)
				}
			}
			statement := "INSERT INTO " + name + " (" + strings.Join(quoted, ",") + ") VALUES (" + strings.TrimSuffix(strings.Repeat("?,", len(columns)), ",") + ")"
			if _, err = tx.ExecContext(ctx, statement, values...); err != nil {
				return fmt.Errorf("hydrate %s row %d: %w", table.Name, rowIndex, err)
			}
		}
	}
	return tx.Commit()
}

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func quoteIdentifier(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !identifierPattern.MatchString(value) {
		return "", fmt.Errorf("invalid fixture identifier %q", value)
	}
	return `"` + value + `"`, nil
}
