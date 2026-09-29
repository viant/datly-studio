package host

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/datly-studio/runtime/namespacemcp"
	"github.com/viant/datly-studio/schema"
	_ "modernc.org/sqlite"
)

func TestNamespaceGroupPollingUsesPersistedSnapshotAndStops(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "studio.db")
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	config := Config{HTTP: Listener{Address: "127.0.0.1:0"}, MCP: Listener{Address: "127.0.0.1:0"}, Authentication: Authentication{DefaultMode: "public"}, Studio: Studio{Driver: "sqlite", DSN: dsn}, Admin: Admin{Token: "fixture-admin"}, RootDir: "."}
	group, err := NewNamespaceGroup(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	runCtx, cancel := context.WithCancel(ctx)
	calls := 0
	if err := group.Run(runCtx, time.Millisecond, func(changes []namespacemcp.Change, err error) {
		calls++
		if err != nil || len(changes) != 0 {
			t.Errorf("empty persisted configuration changes=%+v err=%v", changes, err)
		}
		cancel()
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("unexpected polling count: %d", calls)
	}
	if err := group.Run(ctx, 0, nil); err == nil {
		t.Fatal("zero polling interval accepted")
	}
	closeCtx, closeCancel := context.WithTimeout(ctx, time.Second)
	defer closeCancel()
	if err := group.Close(closeCtx); err != nil {
		t.Fatal(err)
	}
}
