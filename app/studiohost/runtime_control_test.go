package studiohost

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	runtimehost "github.com/viant/datly-studio/runtime/host"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
)

func TestReloadStartsNewlyEnabledNamespaceBeforePolling(t *testing.T) {
	for _, tc := range []struct{ name, address, publicOrigin string }{
		{"loopback", "127.0.0.1:0", ""},
		{"production proxy", "0.0.0.0:0", "https://mcp.example"},
		{"wildcard without connection export", "0.0.0.0:0", ""},
	} {
		t.Run(tc.name, func(t *testing.T) { testReloadEnabledNamespace(t, tc.address, tc.publicOrigin) })
	}
}

func testReloadEnabledNamespace(t *testing.T, address, publicOrigin string) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "studio.db")
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	config := runtimehost.Config{InProcess: true, RootDir: ".", NamespaceMCPMode: "shared", HTTP: runtimehost.Listener{Address: "127.0.0.1:0"}, MCP: runtimehost.Listener{Address: address}, Authentication: runtimehost.Authentication{DefaultMode: "public", PublicMCPURL: publicOrigin}, Studio: runtimehost.Studio{Driver: "sqlite", DSN: dsn}}
	group, err := runtimehost.NewNamespaceGroup(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer group.Close(ctx)
	id := strings.Repeat("a", 64)
	if _, err = db.Exec(`INSERT INTO namespaces(namespace_id,name,title,owner_id,status,mcp_enabled,etag,created_at,updated_at) VALUES(?,'test','Test','owner','active',1,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, id); err != nil {
		t.Fatal(err)
	}
	var active atomic.Pointer[runtimehost.NamespaceGroup]
	active.Store(group)
	if err = inProcessControl(&active).Reload(sdk.WithNamespaceSelection(ctx, id), 0); err != nil {
		t.Fatal(err)
	}
	if _, ok := group.Listeners.Get(id); !ok {
		t.Fatal("reload did not establish the persisted namespace")
	}
}
