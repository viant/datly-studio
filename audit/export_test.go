package audit

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"github.com/viant/datly-studio/schema"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCSVExportPredicatesPaginationAndPermissions(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "audit.db")+"?_txlock=immediate&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	sink := &SQLSink{DB: db}
	defer sink.Close(ctx)
	started := time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 126; i++ {
		subject := "=1+1"
		if i == 125 {
			subject = "other"
		}
		record := Record{Identity: Identity{Subject: subject}, ToolName: "tool,\"quoted\"", Status: "OK", StartedAt: started.Add(time.Duration(i) * time.Second), RequestCompression: "none", ResponseCompression: "none"}
		if err = sink.Write(ctx, record); err != nil {
			t.Fatal(err)
		}
	}
	provider := &syntheticProvider{subject: "auditor", roles: []string{"E2E_AUDIT_VIEWER"}}
	service := &Service{DB: db, Provider: provider, ViewerRole: "E2E_AUDIT_VIEWER"}
	defer service.Close(ctx)
	filter := Filter{Subject: "=1+1", ToolName: "tool,\"quoted\"", From: started.Format(time.RFC3339), To: started.Add(time.Hour).Format(time.RFC3339)}
	result, err := service.ExportCSV(ctx, filter, 0)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(bytes.NewReader(result.Archive)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if result.Count != 125 || result.Truncated || len(rows) != 126 || rows[1][3] != "tool,\"quoted\"" || rows[1][7] != "'=1+1" {
		t.Fatalf("CSV result=%+v rows=%d", result, len(rows))
	}
	if bytes.Contains(result.Archive, []byte("response_original_bytes")) {
		t.Fatal("reserved field exported")
	}
	limited, err := service.ExportCSV(ctx, filter, 10)
	if err != nil || limited.Count != 10 || !limited.Truncated || limited.NextID == 0 {
		t.Fatal("bounded export", limited, err)
	}
	filter.BeforeID = limited.NextID
	rest, err := service.ExportCSV(ctx, filter, 0)
	if err != nil || rest.Count != 115 || rest.Truncated {
		t.Fatal("continuation", rest, err)
	}
	provider.roles = nil
	if _, err = service.ExportCSV(ctx, filter, 0); !errors.Is(err, ErrForbidden) {
		t.Fatal("export bypassed viewer permission", err)
	}
}
func TestCSVCellLiteral(t *testing.T) {
	for _, input := range []string{"=1+1", " +formula", "-1", "@name", "\tdata", "\rdata"} {
		if !strings.HasPrefix(csvCell(input), "'") {
			t.Fatal("unsafe spreadsheet cell", input)
		}
	}
	if csvCell("ordinary") != "ordinary" {
		t.Fatal("ordinary value changed")
	}
}
