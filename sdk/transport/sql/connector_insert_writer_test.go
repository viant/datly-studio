package sqltransport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
)

func TestConnectorInsertWriterPreservesDefaultsAndRejectsDuplicate(t *testing.T) {
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"})
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}})
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.Connectors().Create(ctx, sdk.CreateConnectorInput{
		Name: "source", Driver: "sqlite", DSNTemplate: "file:source.db",
		Options: json.RawMessage(`{"busy_timeout_ms":1000}`),
	})
	if err != nil || created.OwnerID != "owner" || created.Status != "draft" ||
		created.ETag != 1 || !created.DSNConfigured ||
		string(created.Options) != `{"busy_timeout_ms":1000}` {
		t.Fatalf("created=%+v err=%v", created, err)
	}
	var status, options string
	var etag int64
	if err := db.QueryRowContext(ctx, `SELECT status,options_json,etag FROM connectors WHERE name=?`, created.Name).
		Scan(&status, &options, &etag); err != nil {
		t.Fatal(err)
	}
	if status != "draft" || options != string(created.Options) || etag != 1 {
		t.Fatalf("stored status=%q options=%q etag=%d", status, options, etag)
	}
	_, err = client.Connectors().Create(ctx, sdk.CreateConnectorInput{Name: created.Name, Driver: "sqlite"})
	var sdkErr *sdk.Error
	if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorConflict {
		t.Fatalf("duplicate insert error=%v, want conflict", err)
	}
}
