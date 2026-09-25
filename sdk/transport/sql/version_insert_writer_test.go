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

func TestVersionInsertWriterSupportsSQLAndStructuredModes(t *testing.T) {
	ctx := context.Background()
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
	owner := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"})
	connector, err := client.Connectors().Create(owner, sdk.CreateConnectorInput{Name: "main", Driver: "sqlite"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE connectors SET status='active' WHERE name=?`, connector.Name); err != nil {
		t.Fatal(err)
	}
	report, err := client.Reports().Create(owner, sdk.CreateReportInput{Slug: "modes", Title: "Modes", DefaultConnectorName: connector.Name})
	if err != nil {
		t.Fatal(err)
	}
	sqlVersion, err := client.Versions().Create(owner, report.ID, sdk.CreateVersionInput{
		AuthoringMode: "sql", AuthoredSQL: "SELECT 1", Notes: "first",
		ComponentSpec: json.RawMessage(`{"views":[]}`),
	})
	if err != nil || sqlVersion.VersionNo != 1 || sqlVersion.AuthoringMode != "sql" ||
		sqlVersion.AuthoredSQL != "SELECT 1" || sqlVersion.GeneratedDQL != "SELECT 1" ||
		sqlVersion.Notes != "first" || sqlVersion.CreatedBy != "owner" ||
		string(sqlVersion.ComponentSpec) != `{"views":[]}` || sqlVersion.SourceRevision != 1 {
		t.Fatalf("SQL version=%+v err=%v", sqlVersion, err)
	}
	structured, err := client.Versions().Create(owner, report.ID, sdk.CreateVersionInput{
		AuthoringMode: "structured", ComponentSpec: json.RawMessage(`{"views":[{"name":"v"}]}`),
	})
	if err != nil || structured.VersionNo != 2 || structured.GeneratedDQL != "" ||
		structured.AuthoredSQL != "" || structured.AuthoredDQL != "" ||
		string(structured.ComponentSpec) != `{"views":[{"name":"v"}]}` {
		t.Fatalf("structured version=%+v err=%v", structured, err)
	}
	_, err = client.Versions().Create(owner, report.ID, sdk.CreateVersionInput{AuthoringMode: "sql", CreatedBy: "other", AuthoredSQL: "SELECT 2"})
	var sdkErr *sdk.Error
	if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorForbidden {
		t.Fatalf("spoofed version author error=%v", err)
	}
}
