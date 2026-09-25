package host

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	_ "modernc.org/sqlite"
)

func TestActiveConnectorStorePreservesFieldsAndFiltering(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for i := 0; i < 120; i++ {
		name := fmt.Sprintf("active-%03d", i)
		_, err = db.ExecContext(ctx, `INSERT INTO connectors(name,driver,dsn_template,secret_ref,owner_id,status,options_json,etag,created_at,updated_at)
			VALUES(?,?,?,?,?,'active',?,1,?,?)`, name, "sqlite", "file:"+name, "secret:"+name, "owner", `{"cache":"shared"}`, now, now)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, row := range []struct {
		name, status string
		deleted      any
	}{
		{"draft", "draft", nil}, {"disabled", "disabled", nil}, {"deleted", "active", now},
	} {
		_, err = db.ExecContext(ctx, `INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at,deleted_at)
			VALUES(?,'sqlite','owner',?,1,?,?,?)`, row.name, row.status, now, now, row.deleted)
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = db.ExecContext(ctx, `INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at)
		VALUES('null-fields','sqlite','owner','active',1,?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	store := &activeConnectorStore{db: db}
	defer store.Close(ctx)
	got, err := store.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 121 {
		t.Fatalf("active connector count=%d, want 121", len(got))
	}
	for _, name := range []string{"draft", "disabled", "deleted"} {
		if _, ok := got[name]; ok {
			t.Errorf("filtered connector %q returned", name)
		}
	}
	item := got["active-119"]
	if item.name != "active-119" || item.driver != "sqlite" || item.dsn != "file:active-119" ||
		item.secretRef != "secret:active-119" || string(item.options) != `{"cache":"shared"}` {
		t.Fatalf("connector fields were not preserved: %+v", item)
	}
	item = got["null-fields"]
	if item.dsn != "" || item.secretRef != "" || len(item.options) != 0 {
		t.Fatalf("nullable connector fields changed: %+v", item)
	}
}

func TestActiveConnectorStoreReusesRuntimeButReadsFreshRows(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	store := &activeConnectorStore{db: db}
	defer store.Close(ctx)
	first, err := store.Read(ctx)
	if err != nil || len(first) != 0 || store.runtime == nil {
		t.Fatalf("initial read=%v runtime=%v err=%v", first, store.runtime, err)
	}
	initialRuntime := store.runtime
	now := time.Now().UTC()
	_, err = db.ExecContext(ctx, `INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at)
		VALUES('new-source','sqlite','owner','active',1,?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Read(ctx)
	if err != nil || len(second) != 1 || second["new-source"].name != "new-source" || store.runtime != initialRuntime {
		t.Fatalf("fresh read=%v same runtime=%v err=%v", second, store.runtime == initialRuntime, err)
	}
	if err := store.Close(ctx); err != nil || store.runtime != nil {
		t.Fatalf("close runtime=%v err=%v", store.runtime, err)
	}
}
