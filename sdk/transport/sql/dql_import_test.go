package sqltransport

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/authoring/readerbuilder"
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
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}}
	client, err := sdk.NewClient(transport)
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
	report, err := client.Components().Create(ctx, sdk.CreateComponentInput{Slug: "imported", Title: "Imported", DefaultConnectorName: connector.Name})
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
	if _, err = db.Exec(`CREATE TRIGGER reject_import BEFORE INSERT ON component_resource_files BEGIN SELECT RAISE(ABORT,'test import failure'); END`); err != nil {
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
	current, err := client.Components().Get(ctx, report.ID)
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
	if _, err = db.Exec(`INSERT INTO component_acl(report_id,subject_type,subject_id,can_view,can_edit,can_use_dql)
		VALUES(?,'user','editor',1,1,0)`, report.ID); err != nil {
		t.Fatal(err)
	}
	delegated := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "editor"})
	redacted, err := client.Versions().LoadDQL(delegated, report.ID, sdk.LoadDQLInput{DQL: "SELECT delegated_source"})
	if err != nil || redacted.Version == nil || redacted.Version.AuthoredDQL != "" || redacted.Version.GeneratedDQL != "" {
		t.Fatalf("delegated import response=%+v err=%v", redacted, err)
	}
	var storedDQL string
	if err = db.QueryRow(`SELECT authored_dql FROM component_versions WHERE report_id=? AND version_no=?`, report.ID, redacted.Version.VersionNo).Scan(&storedDQL); err != nil || storedDQL != "SELECT delegated_source" {
		t.Fatalf("delegated import stored DQL=%q err=%v", storedDQL, err)
	}
	edited, err := client.Versions().Apply(delegated, report.ID, redacted.Version.VersionNo, sdk.EditCommand{
		Kind: "set_dql", ExpectedSourceRevision: 1,
		Payload: json.RawMessage(`{"authoredDql":"SELECT delegated_edit"}`),
	})
	if err != nil || edited.Version == nil || edited.Version.SourceRevision != 2 ||
		edited.Version.AuthoredDQL != "" || edited.Version.GeneratedDQL != "" {
		t.Fatalf("delegated edit response=%+v err=%v", edited, err)
	}
	if err = db.QueryRow(`SELECT authored_dql FROM component_versions WHERE report_id=? AND version_no=?`, report.ID, redacted.Version.VersionNo).Scan(&storedDQL); err != nil || storedDQL != "SELECT delegated_edit" {
		t.Fatalf("delegated edit stored DQL=%q err=%v", storedDQL, err)
	}
	validated, err := client.Versions().Validate(delegated, report.ID, redacted.Version.VersionNo, edited.Version.SourceRevision)
	if err != nil || validated.Version == nil || !validated.Valid ||
		validated.Version.AuthoredDQL != "" || validated.Version.GeneratedDQL != "" || len(validated.Version.CompileDiagnostics) != 0 {
		t.Fatalf("delegated validation response=%+v err=%v", validated, err)
	}
	transport.Validator = validationStub{err: errors.New("SELECT delegated_edit failed")}
	invalid, err := client.Versions().Validate(delegated, report.ID, redacted.Version.VersionNo, edited.Version.SourceRevision)
	if err != nil || invalid.Valid || invalid.Version == nil || invalid.Version.AuthoredDQL != "" ||
		len(invalid.Version.CompileDiagnostics) != 0 || len(invalid.Diagnostics) != 1 ||
		invalid.Diagnostics[0].Code != "runtime_contract" || invalid.Diagnostics[0].Message != "" {
		t.Fatalf("delegated invalid validation response=%+v err=%v", invalid, err)
	}
	transport.Validator = nil
	operation, err := json.Marshal(readerbuilder.Operation{Type: readerbuilder.OperationAddField,
		Field: &readerbuilder.Field{Name: "FilterID", Type: "int", SourceKind: "query", SourceName: "filterId"}})
	if err != nil {
		t.Fatal(err)
	}
	built, err := client.Versions().ApplyReaderCommand(delegated, report.ID, loaded.Version.VersionNo,
		sdk.ReaderBuilderCommand{ExpectedSourceRevision: loaded.Version.SourceRevision, Operation: operation})
	if err != nil || built.Inspection == nil || built.Inspection.Version == nil || !built.Applied ||
		built.Inspection.Version.AuthoredDQL != "" || built.Inspection.Version.GeneratedDQL != "" || built.Inspection.DQL != "" {
		t.Fatalf("delegated builder response=%+v err=%v", built, err)
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
