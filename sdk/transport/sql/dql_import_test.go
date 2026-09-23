package sqltransport

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
)

func TestDQLImportPersistsAndExecutesDependencies(t *testing.T) {
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"})
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	sourceDSN := "file:" + filepath.Join(t.TempDir(), "source.db")
	source, err := sql.Open("sqlite", sourceDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if _, err = source.Exec(`CREATE TABLE records(id INTEGER PRIMARY KEY,name TEXT); INSERT INTO records VALUES(1,'Imported')`); err != nil {
		t.Fatal(err)
	}
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}})
	if err != nil {
		t.Fatal(err)
	}
	connector, err := client.Connectors().Create(ctx, sdk.CreateConnectorInput{Name: "source", Driver: "sqlite", DSNTemplate: sourceDSN})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE connectors SET status='active' WHERE name=?`, connector.Name); err != nil {
		t.Fatal(err)
	}
	report, err := client.Reports().Create(ctx, sdk.CreateReportInput{Slug: "imported", Title: "Imported", DefaultConnectorName: connector.Name})
	if err != nil {
		t.Fatal(err)
	}
	dql := `#setting($_ = $connector('source'))
#setting($_ = $route('/imported','GET'))
#define($_ = $Rows<[]*Record>(output/view))
SELECT records.*, type(records,'Record') FROM (${embed:sql/records.dql}) records`
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for name, content := range map[string]string{"main.dql": dql, "sql/records.dql": "SELECT id,name FROM records", "other.dql": "SELECT 2"} {
		file, e := writer.Create(name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = file.Write([]byte(content)); e != nil {
			t.Fatal(e)
		}
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	// Ambiguity must fail before creating any draft.
	if _, err = client.Versions().LoadArchive(ctx, report.ID, sdk.LoadArchiveInput{Archive: archive.Bytes(), Format: "zip"}); err == nil {
		t.Fatal("accepted ambiguous entry")
	}
	loaded, err := client.Versions().LoadArchive(ctx, report.ID, sdk.LoadArchiveInput{Archive: archive.Bytes(), Format: "zip", EntryDQL: "main.dql"})
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version.VersionNo != 1 || len(loaded.Entries) != 2 || len(loaded.Files) != 3 {
		t.Fatalf("import=%+v", loaded)
	}
	result, err := (preview.Dynamic{StudioDB: db, RootDir: "../../.."}).Execute(ctx, report.ID, 1, sdk.PreviewInput{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result.Data), "Imported") {
		t.Fatalf("preview=%s", result.Data)
	}
	download, err := client.Versions().Download(ctx, report.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := sdk.ReadDQLArchive(bytes.NewReader(download.Archive), "zip")
	if err != nil {
		t.Fatal(err)
	}
	if string(bundle.Files["sql/records.dql"]) != "SELECT id,name FROM records" {
		t.Fatal("download lost dependency")
	}
	// A resource insert failure must roll back the version and draft pointer.
	if _, err = db.Exec(`CREATE TRIGGER reject_import BEFORE INSERT ON report_resource_files BEGIN SELECT RAISE(ABORT,'test import failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Versions().LoadDQL(ctx, report.ID, sdk.LoadDQLInput{DQL: "SELECT 1"}); err == nil {
		t.Fatal("expected resource failure")
	}
	versions, err := client.Versions().List(ctx, report.ID, sdk.ListVersionsInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(versions.Items) != 1 {
		t.Fatal("partial draft persisted")
	}
	current, err := client.Reports().Get(ctx, report.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.CurrentDraftVersion == nil || *current.CurrentDraftVersion != 1 {
		t.Fatal("draft pointer changed")
	}
	if _, err = db.Exec(`DROP TRIGGER reject_import`); err != nil {
		t.Fatal(err)
	}
	single, err := client.Versions().LoadDQL(ctx, report.ID, sdk.LoadDQLInput{DQL: "SELECT 1"})
	if err != nil {
		t.Fatal(err)
	}
	if single.Version.VersionNo != 2 || single.Version.AuthoredDQL != "SELECT 1" {
		t.Fatalf("single=%+v", single)
	}
	inline := strings.Replace(dql, "${embed:sql/records.dql}", "SELECT id,name FROM records", 1)
	inlineVersion, err := client.Versions().LoadDQL(ctx, report.ID, sdk.LoadDQLInput{DQL: inline})
	if err != nil {
		t.Fatal(err)
	}
	download, err = client.Versions().Download(ctx, report.ID, inlineVersion.Version.VersionNo)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err = sdk.ReadDQLArchive(bytes.NewReader(download.Archive), "zip")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bundle.Files[download.EntryDQL]), "${embed:sql/") {
		t.Fatal("SQL was not delegated")
	}
	reloaded, err := client.Versions().LoadArchive(ctx, report.ID, sdk.LoadArchiveInput{Archive: download.Archive, Format: "zip", EntryDQL: download.EntryDQL})
	if err != nil {
		t.Fatal(err)
	}
	result, err = (preview.Dynamic{StudioDB: db, RootDir: "../../.."}).Execute(ctx, report.ID, reloaded.Version.VersionNo, sdk.PreviewInput{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(result.Data), "Imported") {
		t.Fatalf("round trip result=%s", result.Data)
	}
}
