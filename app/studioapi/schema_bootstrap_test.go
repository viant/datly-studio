package studioapi

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"
)

func TestHostSchemaVerifierNeverInitializesAndPreservesRejection(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err = db.Exec("PRAGMA query_only=ON"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	// The old initialization path attempts DDL and fails against a read-only DB.
	if err := ensureSchema(ctx, db); err == nil {
		t.Fatal("default initialization unexpectedly passed on empty read-only schema")
	}
	rejected := errors.New("existing schema is incompatible")
	calls := 0
	for _, want := range []error{nil, rejected} {
		err = ensureHostSchema(ctx, db, func(actual context.Context, same *sql.DB) error {
			calls++
			if actual != ctx || same != db {
				t.Fatal("verifier must receive original context/database")
			}
			var count int
			if e := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count); e != nil {
				t.Fatal(e)
			}
			if count != 0 {
				t.Fatalf("startup created %d tables", count)
			}
			return want
		})
		if !errors.Is(err, want) {
			t.Fatalf("verifier error=%v want=%v", err, want)
		}
	}
	if calls != 2 {
		t.Fatalf("verifier calls=%d", calls)
	}
	if _, err = db.Exec("PRAGMA query_only=OFF"); err != nil {
		t.Fatal(err)
	}
	if err = ensureHostSchema(ctx, db, nil); err != nil {
		t.Fatalf("legacy default initialization: %v", err)
	}
	var count int
	if err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='connectors'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("default schema connectors=%d error=%v", count, err)
	}
}
