package sqltransport

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/authoring/readerbuilder"
	_ "modernc.org/sqlite"
)

func TestBuilderCubeUsesExactVersionSQLResource(t *testing.T) {
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"})
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}})
	if err != nil {
		t.Fatal(err)
	}
	connector, err := client.Connectors().Create(ctx, sdk.CreateConnectorInput{Name: "main", Driver: "sqlite"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE connectors SET status='active' WHERE name=?`, connector.Name); err != nil {
		t.Fatal(err)
	}
	component, err := client.Components().Create(ctx, sdk.CreateComponentInput{Slug: "embedded", Title: "Embedded", DefaultConnectorName: connector.Name})
	if err != nil {
		t.Fatal(err)
	}
	source := `#setting($_ = $route('/embedded','GET'))
SELECT forecast.*, groupable(forecast), tag(forecast.country,'groupable:"true"')
FROM (${embed:sql/forecast.sql}) forecast
JOIN (SELECT ID, NAME FROM dictionary) channel ON channel.ID = forecast.channel_id AND 1=1`
	version, err := client.Versions().Create(ctx, component.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: source})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := client.Resources().UpsertFile(ctx, sdk.ResourceFile{ReportID: component.ID, VersionNo: version.VersionNo, Namespace: component.OwnerPackage + ".sql", ResourcePath: "sql/forecast.sql", Content: "SELECT country,channel_id,SUM(avails) AS avails FROM events GROUP BY country,channel_id\n", ExpectedSourceRevision: version.SourceRevision})
	if err != nil {
		t.Fatal(err)
	}
	operation, err := json.Marshal(readerbuilder.Operation{Type: readerbuilder.OperationSetSetting, Setting: &readerbuilder.SettingMutation{Name: "cube"}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Versions().ApplyReaderCommand(ctx, component.ID, version.VersionNo, sdk.ReaderBuilderCommand{ExpectedSourceRevision: snapshot.Version.SourceRevision, Operation: operation})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Applied || !strings.Contains(result.Inspection.DQL, "${embed:sql/forecast.sql}") {
		t.Fatalf("result=%+v", result)
	}
	current, err := client.Versions().Get(ctx, component.ID, version.VersionNo)
	if err != nil {
		t.Fatal(err)
	}
	if current.SourceRevision != snapshot.Version.SourceRevision+1 {
		t.Fatalf("revision=%d", current.SourceRevision)
	}
}
