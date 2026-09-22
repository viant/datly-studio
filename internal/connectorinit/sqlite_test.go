package connectorinit

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestConfigureAttachesSQLiteSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.db")
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`CREATE TABLE records(id INTEGER PRIMARY KEY); INSERT INTO records VALUES(1)`); err != nil {
		t.Fatal(err)
	}
	options := json.RawMessage(`{"sqliteAttachments":{"catalog":"` + path + `"}}`)
	if err = Configure(context.Background(), db, "sqlite", options); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.QueryRow(`SELECT COUNT(*) FROM catalog.records`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	if err = Configure(context.Background(), db, "sqlite", json.RawMessage(`{"sqliteAttachments":{"bad-name":"x"}}`)); err == nil {
		t.Fatal("invalid attachment name was accepted")
	}
}
