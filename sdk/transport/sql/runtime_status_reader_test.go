package sqltransport

import (
	"context"
	"database/sql"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
)

func TestRuntimeStatusUsesLatestActiveGeneratedReader(t *testing.T) {
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
	idle, err := client.Runtime().Status(owner)
	if err != nil || idle.Status != "idle" || idle.ActiveGeneration != 0 {
		t.Fatalf("idle status=%+v err=%v", idle, err)
	}
	now := time.Now().UTC()
	for _, row := range []struct {
		generation  int64
		status      string
		count       int
		diagnostics any
	}{
		{1, "active", 1, nil}, {2, "retired", 2, nil},
		{3, "active", 3, `[{"code":"check","message":"warning"}]`},
	} {
		if _, err := db.ExecContext(ctx, `INSERT INTO runtime_generations
			(generation_no,source_revision,status,report_count,build_manifest_json,diagnostics_json,requested_by,requested_at,activated_at)
			VALUES(?,?,?,?, '{}', ?, 'owner', ?, ?)`, row.generation, strconv.FormatInt(row.generation, 10), row.status, row.count, row.diagnostics, now, now); err != nil {
			t.Fatal(err)
		}
	}
	active, err := transport.readActiveGeneration(owner)
	if err != nil || active == nil || active.GenerationNo != 3 || active.ReportCount != 3 || active.ActivatedAt == nil {
		t.Fatalf("active generated row=%+v err=%v", active, err)
	}
	status, err := client.Runtime().Status(owner)
	if err != nil || status.ActiveGeneration != 3 || status.Status != "active" || status.ReportCount != 3 ||
		status.ActivatedAt == nil || len(status.Diagnostics) != 1 || status.Diagnostics[0].Code != "check" ||
		status.Host == nil || status.Host.Status != "unknown" {
		t.Fatalf("runtime status=%+v err=%v", status, err)
	}
}
