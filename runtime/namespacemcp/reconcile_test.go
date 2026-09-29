package namespacemcp

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/schema"
	_ "modernc.org/sqlite"
)

func TestPersistedNamespaceSettingsDriveIndependentListeners(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	aID, bID := namespaceaccess.ID("owner", "alpha"), namespaceaccess.ID("owner", "beta")
	for name, id := range map[string]string{"alpha": aID, "beta": bID} {
		if _, err := db.Exec(`INSERT INTO namespaces(namespace_id,owner_id,name,title,status,mcp_enabled,etag,created_at,updated_at) VALUES(?,'owner',?,?,'active',TRUE,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, id, name, name); err != nil {
			t.Fatal(err)
		}
	}
	manager, _ := New(factory)
	defer manager.Close(ctx)
	reconciler := &Reconciler{Manager: manager, Definitions: SQLDefinitions{DB: db}}
	changes, err := reconciler.Apply(ctx)
	if err != nil || len(changes) != 2 {
		t.Fatalf("initial reconciliation=%+v err=%v", changes, err)
	}
	a, _ := manager.Get(aID)
	b, _ := manager.Get(bID)
	if a.Port == b.Port {
		t.Fatal("persisted namespaces shared a port")
	}
	if !bytes.Contains(call(t, a, "tools/list", nil), []byte("read_"+aID[:8])) {
		t.Fatal("persisted namespace catalog missing")
	}
	if _, err := db.Exec(`UPDATE namespaces SET mcp_enabled=FALSE WHERE namespace_id=?`, aID); err != nil {
		t.Fatal(err)
	}
	if _, err := reconciler.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := manager.Get(aID); ok {
		t.Fatal("disabled namespace listener stayed ready")
	}
	currentB, _ := manager.Get(bID)
	if currentB != b {
		t.Fatal("disabling A changed B")
	}
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE namespaces SET mcp_port=? WHERE namespace_id=?`, occupied.Addr().(*net.TCPAddr).Port, bID); err != nil {
		t.Fatal(err)
	}
	changes, err = reconciler.Apply(ctx)
	if err == nil || len(changes) != 1 || changes[0].Status != "failed" || changes[0].Endpoint == nil || *changes[0].Endpoint != b {
		t.Fatalf("failed persisted rebind=%+v err=%v", changes, err)
	}
	occupied.Close()
	if _, err := db.Exec(`UPDATE namespaces SET mcp_port=NULL,status='archived' WHERE namespace_id=?`, bID); err != nil {
		t.Fatal(err)
	}
	if _, err := reconciler.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := manager.Get(bID); ok {
		t.Fatal("archived namespace listener stayed ready")
	}
}

type failedDefinitions struct{}

func (failedDefinitions) Load(context.Context) ([]Definition, error) {
	return nil, errors.New("snapshot unavailable")
}
func TestFailedSnapshotDoesNotStopExistingListener(t *testing.T) {
	id := namespaceaccess.ID("owner", "existing")
	manager, _ := New(factory)
	defer manager.Close(context.Background())
	endpoint, err := manager.Start(context.Background(), id, 0)
	if err != nil {
		t.Fatal(err)
	}
	reconciler := &Reconciler{Manager: manager, Definitions: failedDefinitions{}}
	if _, err := reconciler.Apply(context.Background()); err == nil {
		t.Fatal("failed snapshot accepted")
	}
	current, ok := manager.Get(id)
	if !ok || current != endpoint {
		t.Fatal("failed snapshot removed active namespace")
	}
}
