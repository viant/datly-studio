package connectors

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
)

func TestOpenBuildsTrustedNamedDatlyConnectors(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "source.db")
	seed, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := seed.ExecContext(ctx, "CREATE TABLE records(id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	if err := seed.Close(); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(ctx, "source", []*sdk.Connector{{Name: "source", Status: "active", Driver: "sqlite", DSNTemplate: dsn}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if opened.SQL == nil || opened.Connections["source"] == nil {
		t.Fatalf("opened connector = %+v", opened)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(ctx, "missing", []*sdk.Connector{{Name: "source", Status: "active", Driver: "sqlite", DSNTemplate: dsn}}, nil); err == nil {
		t.Fatal("missing primary connector was accepted")
	}
	if _, err := Open(ctx, "source", []*sdk.Connector{{Name: "source", Status: "disabled", Driver: "sqlite", DSNTemplate: dsn}}, nil); err == nil {
		t.Fatal("disabled connector was accepted")
	}
	if _, err := Open(ctx, "source", []*sdk.Connector{{Name: "source", Status: "active", Driver: "sqlite", DSNTemplate: dsn}, {Name: "source", Status: "active", Driver: "sqlite", DSNTemplate: dsn}}, nil); err == nil {
		t.Fatal("duplicate connector was accepted")
	}
}
