package sqltransport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	stored "github.com/viant/datly-studio/studio/report_versions/store_edit"
	xhandler "github.com/viant/xdatly/handler"
	_ "modernc.org/sqlite"
)

func TestVersionEditWriterRejectsStaleSourceRevision(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}}
	client, err := sdk.NewClient(transport)
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
	report, err := client.Reports().Create(owner, sdk.CreateReportInput{Slug: "edit", Title: "Edit", DefaultConnectorName: connector.Name})
	if err != nil {
		t.Fatal(err)
	}
	version, err := client.Versions().Create(owner, report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
	if err != nil {
		t.Fatal(err)
	}
	expected := version.SourceRevision
	updatedDQL := "SELECT 2"
	row := &stored.StoredVersion{ReportId: report.ID, VersionNo: version.VersionNo,
		AuthoredDql: &updatedDQL, GeneratedDql: &updatedDQL,
		ComponentSpecJson: json.RawMessage(`{}`),
		SpecHash:          hashVersion(report.ID, version.VersionNo, "dql", "", updatedDQL, json.RawMessage(`{}`)),
		CompileStatus:     "pending", CompileDiagnosticsJson: json.RawMessage(`[]`), SourceRevision: &expected,
		Has: &stored.StoredVersionHas{ReportId: true, VersionNo: true,
			AuthoredSql: true, AuthoredDql: true, ComponentSpecJson: true, SpecHash: true,
			GeneratedDql: true, CompileStatus: true, CompileDiagnosticsJson: true, SourceRevision: true}}
	if err := transport.writeVersionEdit(owner, nil, row); err != nil {
		t.Fatal(err)
	}
	current, err := transport.getVersionValue(owner, report.ID, version.VersionNo)
	if err != nil || current.SourceRevision != 2 || current.AuthoredDQL != updatedDQL {
		t.Fatalf("edited version=%+v err=%v", current, err)
	}
	staleExpected := version.SourceRevision
	stale := *row
	stale.SourceRevision = &staleExpected
	var conflict *xhandler.Conflict
	if err := transport.writeVersionEdit(owner, nil, &stale); !errors.As(err, &conflict) {
		t.Fatalf("stale writer error=%v", err)
	}
	current, err = transport.getVersionValue(owner, report.ID, version.VersionNo)
	if err != nil || current.SourceRevision != 2 || current.AuthoredDQL != updatedDQL {
		t.Fatalf("stale writer changed version=%+v err=%v", current, err)
	}
	_, err = client.Versions().Apply(owner, report.ID, version.VersionNo, sdk.EditCommand{
		Kind: "set_dql", ExpectedSourceRevision: version.SourceRevision,
		Payload: json.RawMessage(`{"authoredDql":"SELECT 3"}`),
	})
	var sdkErr *sdk.Error
	if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorConflict {
		t.Fatalf("stale SDK edit error=%v", err)
	}
	for _, revision := range []int64{0, -1} {
		_, err = client.Versions().Apply(owner, report.ID, version.VersionNo, sdk.EditCommand{
			Kind: "set_dql", ExpectedSourceRevision: revision,
			Payload: json.RawMessage(`{"authoredDql":"SELECT 3"}`),
		})
		if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorInvalidArgument || !strings.Contains(sdkErr.Message, "expectedSourceRevision") {
			t.Fatalf("revision %d SDK edit error=%v", revision, err)
		}
	}
	current, err = client.Versions().Get(owner, report.ID, version.VersionNo)
	if err != nil || current.SourceRevision != 2 || current.AuthoredDQL != updatedDQL {
		t.Fatalf("rejected edits changed version=%+v err=%v", current, err)
	}
}
