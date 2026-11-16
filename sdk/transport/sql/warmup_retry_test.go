package sqltransport

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"log"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/viant/datly-studio/sdk"
)

// Inject driver failures around a real SQLite transaction, retaining the
// original driver identity so Datly owns dialect-specific classification.
type warmupFaultConnector struct {
	driver driver.Driver
	dsn    string
	fault  *warmupStorageFault
}

func (c *warmupFaultConnector) Driver() driver.Driver { return c.driver }
func (c *warmupFaultConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.driver.Open(c.dsn)
	if err != nil {
		return nil, err
	}
	return &warmupFaultConn{Conn: conn, fault: c.fault}, nil
}

type warmupStorageFault struct {
	attempts      atomic.Int32
	failures      int32
	code          int
	status        string
	unknownCommit bool
	afterRollback func()
}

type warmupCodedStorageError int

func (e warmupCodedStorageError) Error() string { return "injected storage failure" }
func (e warmupCodedStorageError) Code() int     { return int(e) }

type warmupFaultConn struct {
	driver.Conn
	fault *warmupStorageFault
	tx    *warmupFaultTx
}

func (c *warmupFaultConn) Prepare(query string) (driver.Stmt, error) {
	stmt, err := c.Conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return &warmupFaultStmt{Stmt: stmt, conn: c, query: strings.ToLower(query)}, nil
}
func (c *warmupFaultConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	tx, err := c.Conn.(driver.ConnBeginTx).BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	c.tx = &warmupFaultTx{Tx: tx, fault: c.fault}
	return c.tx, nil
}

type warmupFaultStmt struct {
	driver.Stmt
	conn  *warmupFaultConn
	query string
}

func (s *warmupFaultStmt) Exec(args []driver.Value) (driver.Result, error) {
	if strings.HasPrefix(s.query, "update ") && strings.Contains(s.query, "component_warmup_runs") {
		for _, arg := range args {
			status, ok := arg.(string)
			matches := status == "completed" || status == "partial" || status == "failed"
			if s.conn.fault.status != "" {
				matches = status == s.conn.fault.status
			}
			if ok && matches {
				s.conn.tx.terminal = true
				if s.conn.fault.attempts.Add(1) <= s.conn.fault.failures {
					return nil, warmupCodedStorageError(s.conn.fault.code)
				}
				break
			}
		}
	}
	return s.Stmt.Exec(args)
}

type warmupFaultTx struct {
	driver.Tx
	fault    *warmupStorageFault
	terminal bool
}

func (tx *warmupFaultTx) Commit() error {
	err := tx.Tx.Commit()
	if err == nil && tx.terminal && tx.fault.unknownCommit {
		return errors.New("injected unknown commit outcome")
	}
	return err
}
func (tx *warmupFaultTx) Rollback() error {
	err := tx.Tx.Rollback()
	if err == nil && tx.terminal && tx.fault.afterRollback != nil {
		tx.fault.afterRollback()
	}
	return err
}

type countedWarmup struct {
	calls  int
	status string
}

func (w *countedWarmup) Warmup(ctx context.Context, _ string, _ int) (*sdk.WarmupResult, error) {
	w.calls++
	return terminalDiagnosticWarmup{status: w.status}.Warmup(ctx, "", 0)
}

func TestWarmupTerminalNativeTransactionRetry(t *testing.T) {
	for _, tc := range []struct {
		name, status  string
		faultStatus   string
		failures      int32
		code          int
		unknownCommit bool
		concurrent    bool
		attempts      int32
		persisted     string
		logged        bool
	}{
		{name: "completed busy", status: "completed", failures: 1, code: 5, attempts: 2, persisted: "completed"},
		{name: "partial locked", status: "partial", failures: 1, code: 6, attempts: 2, persisted: "partial"},
		{name: "failed busy", status: "failed", failures: 1, code: 5, attempts: 2, persisted: "failed"},
		{name: "retry exhausted", status: "completed", failures: 2, code: 5, attempts: 2, persisted: "running", logged: true},
		{name: "permanent failure", status: "completed", failures: 1, code: 19, attempts: 1, persisted: "running", logged: true},
		{name: "unknown commit", status: "completed", unknownCommit: true, attempts: 1, persisted: "completed", logged: true},
		{name: "concurrent update", status: "completed", failures: 1, code: 5, concurrent: true, attempts: 1, persisted: "running", logged: true},
		{name: "running is not retried", status: "completed", faultStatus: "running", failures: 1, code: 5, attempts: 1, persisted: "accepted"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newImportFixtureDSN(t, "?_txlock=immediate&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
			f.transport.Now = nil
			v, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
			if err != nil {
				t.Fatal(err)
			}
			now := time.Now().UTC().Truncate(time.Microsecond)
			_, err = f.db.ExecContext(f.ctx, `INSERT INTO component_warmup_runs(run_id,report_id,version_no,source_revision,spec_hash,plan_key,active_key,status,requested_by,target_json,requested_at,updated_at,updated_by) VALUES('retry-run',?,?,1,'hash','plan','retry-key','accepted','owner','{}',?,?,'owner')`, f.report.ID, v.VersionNo, now, now)
			if err != nil {
				t.Fatal(err)
			}
			var seq int
			var name, path string
			if err := f.db.QueryRowContext(f.ctx, "PRAGMA database_list").Scan(&seq, &name, &path); err != nil {
				t.Fatal(err)
			}
			fault := &warmupStorageFault{failures: tc.failures, code: tc.code, status: tc.faultStatus, unknownCommit: tc.unknownCommit}
			if tc.concurrent {
				fault.afterRollback = func() {
					_, err := f.db.ExecContext(f.ctx, "UPDATE component_warmup_runs SET updated_at=? WHERE run_id='retry-run'", time.Now().UTC().Add(time.Second))
					if err != nil {
						t.Error(err)
					}
				}
			}
			db := sql.OpenDB(&warmupFaultConnector{driver: f.db.Driver(), dsn: "file:" + path + "?_txlock=immediate&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", fault: fault})
			t.Cleanup(func() { _ = db.Close() })
			f.transport.DB = db
			warmup := &countedWarmup{status: tc.status}
			f.transport.Warmup = warmup
			messages := make(terminalDiagnosticLog, 8)
			previous := log.Writer()
			log.SetOutput(messages)
			t.Cleanup(func() { log.SetOutput(previous) })
			f.transport.ExecuteAcceptedWarmupRun(f.ctx, "retry-run", f.report.ID, v.VersionNo, now)
			wantCalls := 1
			if tc.persisted == "accepted" {
				wantCalls = 0
			}
			if warmup.calls != wantCalls || fault.attempts.Load() != tc.attempts {
				t.Fatalf("business executions=%d write attempts=%d, want %d/%d", warmup.calls, fault.attempts.Load(), wantCalls, tc.attempts)
			}
			if logged := len(messages) != 0; logged != tc.logged {
				t.Fatalf("logged=%v want=%v", logged, tc.logged)
			}
			var status string
			var completed sql.NullTime
			var key sql.NullString
			if err := f.db.QueryRowContext(f.ctx, "SELECT status,completed_at,active_key FROM component_warmup_runs WHERE run_id='retry-run'").Scan(&status, &completed, &key); err != nil {
				t.Fatal(err)
			}
			terminal := status == "completed" || status == "partial" || status == "failed"
			if status != tc.persisted || completed.Valid != terminal || key.Valid == terminal {
				t.Fatalf("persisted status=%s completed=%v active=%v", status, completed.Valid, key.Valid)
			}
		})
	}
}
