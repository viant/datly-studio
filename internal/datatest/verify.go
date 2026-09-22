package datatest

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"
)

// AssertRows compares ordered query results with a data-driven expectation.
// Scalar values are canonicalized so driver int64/[]byte representations do
// not leak into component tests.
func AssertRows(t testing.TB, ctx context.Context, db *sql.DB, query string, args []any, expected ...Row) {
	t.Helper()
	actual, err := QueryRows(ctx, db, query, args...)
	if err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	if !reflect.DeepEqual(canonicalRows(actual), canonicalRows(expected)) {
		t.Fatalf("query %q rows = %#v, want %#v", query, actual, expected)
	}
}

// AssertRowsJSON compares query results with an ordered JSON array of objects.
func AssertRowsJSON(t testing.TB, ctx context.Context, db *sql.DB, query string, args []any, expected []byte) {
	t.Helper()
	rows, err := RowsFromJSON(expected)
	if err != nil {
		t.Fatal(err)
	}
	AssertRows(t, ctx, db, query, args, rows...)
}

// QueryRows returns ordered rows keyed by the database result labels.
func QueryRows(ctx context.Context, db *sql.DB, query string, args ...any) ([]Row, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]Row, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		targets := make([]any, len(columns))
		for i := range values {
			targets[i] = &values[i]
		}
		if err = rows.Scan(targets...); err != nil {
			return nil, err
		}
		row := Row{}
		for i, column := range columns {
			row[column] = values[i]
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func canonicalRows(rows []Row) []Row {
	result := make([]Row, len(rows))
	for i, row := range rows {
		result[i] = Row{}
		for key, value := range row {
			result[i][key] = canonicalValue(value)
		}
	}
	return result
}

func canonicalValue(value any) any {
	switch actual := value.(type) {
	case nil:
		return nil
	case []byte:
		return string(actual)
	case time.Time:
		return actual.UTC().Format(time.RFC3339Nano)
	default:
		return fmt.Sprint(actual)
	}
}
