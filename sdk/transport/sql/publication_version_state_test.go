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

func TestVersionActivationSupersedesAllPagesAndRollsBack(t *testing.T) {
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
	report, err := client.Reports().Create(owner, sdk.CreateReportInput{Slug: "version-state-pages", Title: "Version state pages", DefaultConnectorName: connector.Name})
	if err != nil {
		t.Fatal(err)
	}
	const targetNo = 504
	now := time.Now().UTC()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for number := 1; number <= targetNo; number++ {
		state := "published"
		if number == targetNo {
			state = "validated"
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO report_versions
			(report_id,version_no,state,authoring_mode,authored_dql,component_spec_json,
			 spec_format_version,spec_hash,type_manifest_json,compile_status,
			 datly_version,compiler_version,source_revision,created_by,created_at)
			 VALUES(?,?,?,'dql','SELECT 1','{}','studio.v1',?,'{}','valid','v1','studio.v1',1,'owner',?)`,
			report.ID, number, state, fmt.Sprintf("%064d", number), now); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	activation, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer activation.Rollback()
	if err := transport.activateVersionState(owner, activation, report.ID, targetNo, now); err != nil {
		t.Fatal(err)
	}
	var superseded int
	if err := activation.QueryRowContext(ctx, `SELECT COUNT(*) FROM report_versions WHERE report_id=? AND state='superseded'`, report.ID).Scan(&superseded); err != nil || superseded != targetNo-1 {
		t.Fatalf("superseded versions=%d err=%v", superseded, err)
	}
	target, err := transport.readVersionCatalogTx(owner, activation, versionCatalogRequest{ReportID: report.ID, VersionNo: targetNo, Limit: 2})
	if err != nil || len(target) != 1 || target[0].State != "published" || target[0].PublishedAt == nil {
		t.Fatalf("published target=%+v err=%v", target, err)
	}
	if err := activation.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM report_versions WHERE report_id=? AND state='published'`, report.ID).Scan(&superseded); err != nil || superseded != targetNo-1 {
		t.Fatalf("rolled-back published versions=%d err=%v", superseded, err)
	}
}
