package connectorinit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestPoolLimitQueuesUntilConnectionReleased(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = Configure(context.Background(), db, "example-driver", json.RawMessage(`{"pool":{"maxOpenConns":1,"maxIdleConns":1}}`)); err != nil {
		t.Fatal(err)
	}
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err = db.ExecContext(ctx, "SELECT 1"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second borrower did not wait: %v", err)
	}
	if err = conn.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(context.Background(), "SELECT 1"); err != nil {
		t.Fatal(err)
	}
	if db.Stats().WaitCount != 1 {
		t.Fatalf("unexpected wait count: %+v", db.Stats())
	}
}

func TestPoolOmissionAndInvalidOptionsPreserveExistingLimits(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(7)
	if err = Configure(context.Background(), db, "example-driver", json.RawMessage(`{"discovery":{}}`)); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`{"pool":{"maxOpenConns":-1}}`, `{"pool":{"maxIdleConns":-1}}`, `{"pool":{"maxOpenConns":1,"maxIdleConns":2}}`} {
		if err = Configure(context.Background(), db, "example-driver", json.RawMessage(raw)); err == nil {
			t.Fatalf("invalid pool accepted: %s", raw)
		}
		if db.Stats().MaxOpenConnections != 7 {
			t.Fatal("invalid pool changed the existing limit")
		}
	}
	if err = Configure(context.Background(), db, "example-driver", json.RawMessage(`{"pool":{"maxOpenConns":0}}`)); err != nil {
		t.Fatal(err)
	}
	if db.Stats().MaxOpenConnections != 0 {
		t.Fatal("explicit unlimited pool was lost")
	}
}
