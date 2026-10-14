package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"github.com/viant/datly-studio/sdk/executionguard"
	_ "modernc.org/sqlite"
	"path/filepath"
	"testing"
)

func TestAuthoringTransactionCommitKeepsServerGuardAndOwner(t *testing.T) {
	for _, deny := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "revoke-before-commit"}[deny], func(t *testing.T) {
			db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "owned.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			ctx := context.Background()
			for _, q := range []string{"CREATE TABLE records(id INTEGER PRIMARY KEY)", "CREATE TABLE audit(id INTEGER)", "CREATE TRIGGER inserted AFTER INSERT ON records BEGIN INSERT INTO audit VALUES(NEW.id);END"} {
				if _, err = db.ExecContext(ctx, q); err != nil {
					t.Fatal(err)
				}
			}
			revoked := false
			denied := errors.New("exact original app candidate revoked")
			ctx = executionguard.With(ctx, func(context.Context) error {
				if revoked {
					return denied
				}
				return nil
			})
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			if _, err = tx.ExecContext(ctx, "INSERT INTO records VALUES(1)"); err != nil {
				t.Fatal(err)
			}
			revoked = deny
			err = commitWithServerGuard(ctx, tx)
			if deny {
				if !errors.Is(err, denied) {
					t.Fatalf("guard lost: %v", err)
				}
				var pending int
				if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit").Scan(&pending); err != nil || pending != 1 {
					t.Fatalf("borrowed unit completed autonomously: pending=%d err=%v", pending, err)
				}
				if err = tx.Rollback(); err != nil {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			want := 1
			if deny {
				want = 0
			}
			for _, table := range []string{"records", "audit"} {
				var n int
				if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&n); err != nil || n != want {
					t.Fatalf("%s committed=%d want=%d err=%v", table, n, want, err)
				}
			}
		})
	}
}
