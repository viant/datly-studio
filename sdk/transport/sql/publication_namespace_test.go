package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
)

func TestPublicationRepointLeavesAnotherNamespaceUnchanged(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}}
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	owner := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"})
	a, err := client.Namespaces().Create(owner, sdk.CreateNamespaceInput{Name: "alpha", Title: "Alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Namespaces().Create(owner, sdk.CreateNamespaceInput{Name: "beta", Title: "Beta"}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Connectors().Create(owner, sdk.CreateConnectorInput{Name: "shared", Driver: "sqlite"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE connectors SET status='active' WHERE name='shared'`); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, generation := range []int64{1, 2} {
		if err := transport.insertBuildingGeneration(owner, nil, generation, fmt.Sprintf("fixture-%d", generation), "owner", now); err != nil {
			t.Fatal(err)
		}
	}
	for id, namespace := range map[string]string{"current": "alpha", "alpha-other": "alpha", "beta-other": "beta"} {
		if _, err := client.Components().Create(owner, sdk.CreateComponentInput{ID: id, Namespace: namespace, Slug: id, Title: id, DefaultConnectorName: "shared"}); err != nil {
			t.Fatal(err)
		}
		version, err := client.Versions().Create(owner, id, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO component_publications(report_id,active_version_no,active_generation,desired_generation,publication_status,runtime_revision,spec_hash,published_by,published_at) VALUES(?,1,1,1,'active','fixture',?,'owner',CURRENT_TIMESTAMP)`, id, version.SpecHash); err != nil {
			t.Fatal(err)
		}
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	count, err := transport.repointActivePublications(sdk.WithNamespaceSelection(owner, a.NamespaceID), tx, "current", 2)
	if err != nil || count != 1 {
		t.Fatalf("scoped repoint count=%d err=%v", count, err)
	}
	for id, want := range map[string]int64{"alpha-other": 2, "beta-other": 1} {
		var generation int64
		if err := tx.QueryRowContext(ctx, `SELECT active_generation FROM component_publications WHERE report_id=?`, id).Scan(&generation); err != nil || generation != want {
			t.Fatalf("publication %s generation=%d want=%d err=%v", id, generation, want, err)
		}
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	var restored int64
	if err := db.QueryRowContext(ctx, `SELECT active_generation FROM component_publications WHERE report_id='alpha-other'`).Scan(&restored); err != nil || restored != 1 {
		t.Fatalf("repoint rollback generation=%d err=%v", restored, err)
	}
}
