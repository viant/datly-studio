package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
)

func TestPublicationEventStoreReads(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:publication_event_store_reads?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE reports (id TEXT PRIMARY KEY, owner_id TEXT NOT NULL, deleted_at TEXT)`,
		`CREATE TABLE report_publication_events (event_id TEXT PRIMARY KEY, report_id TEXT, owner_id TEXT, operation TEXT, version_no INTEGER, generation_no INTEGER, status TEXT, requested_by TEXT, reason TEXT, failure_code TEXT, failure_message TEXT, occurred_at TIMESTAMP)`,
		`INSERT INTO reports(id,owner_id) VALUES ('r1','alice'),('r2','bob')`,
		`INSERT INTO reports(id,owner_id,deleted_at) VALUES ('deleted','alice','2026-01-01')`,
		`INSERT INTO report_publication_events VALUES ('e1','r1','alice','publish',NULL,NULL,'succeeded','alice',NULL,NULL,NULL,'2026-01-01T00:00:00Z')`,
		`INSERT INTO report_publication_events VALUES ('e2','r1','alice','rollback',2,20,'failed','alice','retry','timeout','failed','2026-01-02T00:00:00Z')`,
		`INSERT INTO report_publication_events VALUES ('e3','r1','alice','publish',3,30,'succeeded','alice','published',NULL,NULL,'2026-01-02T00:00:00Z')`,
		`INSERT INTO report_publication_events VALUES ('other','r2','bob','publish',1,10,'succeeded','bob',NULL,NULL,NULL,'2026-01-03T00:00:00Z')`,
	} {
		if _, err = db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	transport := &Transport{DB: db}
	for _, test := range []struct {
		name          string
		input         sdk.ListPublicationEventsInput
		ids           []string
		limit, offset int
	}{
		{"default and stable order", sdk.ListPublicationEventsInput{}, []string{"e3", "e2", "e1"}, 20, 0},
		{"page", sdk.ListPublicationEventsInput{Limit: 1, Offset: 1}, []string{"e2"}, 1, 1},
		{"filters", sdk.ListPublicationEventsInput{Operation: " rollback ", Status: " failed "}, []string{"e2"}, 20, 0},
		{"cap", sdk.ListPublicationEventsInput{Limit: 101}, []string{"e3", "e2", "e1"}, 100, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			var page sdk.PublicationEventPage
			if err := transport.listPublicationEvents(ctx, publicationEventListRequest{ReportID: "r1", Input: test.input}, &page); err != nil {
				t.Fatal(err)
			}
			if page.Limit != test.limit || page.Offset != test.offset || len(page.Items) != len(test.ids) {
				t.Fatalf("page=%+v", page)
			}
			for i, id := range test.ids {
				if page.Items[i].EventID != id || page.Items[i].ReportID != "r1" {
					t.Fatalf("item %d=%+v", i, page.Items[i])
				}
			}
		})
	}
	var page sdk.PublicationEventPage
	if err := transport.listPublicationEvents(ctx, publicationEventListRequest{ReportID: "r1", Input: sdk.ListPublicationEventsInput{Status: "succeeded"}}, &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Items[1].VersionNo != nil || page.Items[1].GenerationNo != nil ||
		page.Items[1].Reason != "" || page.Items[0].OccurredAt.IsZero() {
		t.Fatalf("nullable event fields=%+v", page.Items)
	}
	page = sdk.PublicationEventPage{}
	if err := transport.listPublicationEvents(ctx, publicationEventListRequest{ReportID: "r1", Input: sdk.ListPublicationEventsInput{Status: "failed"}}, &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].VersionNo == nil || *page.Items[0].VersionNo != 2 ||
		page.Items[0].GenerationNo == nil || *page.Items[0].GenerationNo != 20 ||
		page.Items[0].Reason != "retry" || page.Items[0].FailureCode != "timeout" || page.Items[0].FailureMessage != "failed" {
		t.Fatalf("failed event fields=%+v", page.Items)
	}
	for _, input := range []publicationEventListRequest{
		{ReportID: "", Input: sdk.ListPublicationEventsInput{}},
		{ReportID: "r1", Input: sdk.ListPublicationEventsInput{Offset: -1}},
		{ReportID: "r1", Input: sdk.ListPublicationEventsInput{Operation: "delete"}},
		{ReportID: "r1", Input: sdk.ListPublicationEventsInput{Status: "pending"}},
	} {
		var result sdk.PublicationEventPage
		var sdkErr *sdk.Error
		if err := transport.listPublicationEvents(ctx, input, &result); !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorInvalidArgument {
			t.Fatalf("input=%+v error=%v", input, err)
		}
	}
	if actual, err := transport.publicationEventOwner(ctx, "r1"); err != nil || actual != "alice" {
		t.Fatalf("owner=%q error=%v", actual, err)
	}
	for _, id := range []string{"missing", "deleted"} {
		var sdkErr *sdk.Error
		if _, err := transport.publicationEventOwner(ctx, id); !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorNotFound || sdkErr.Message != "report not found" {
			t.Fatalf("owner %q error=%v", id, err)
		}
	}
	if transport.publicationReader == nil {
		t.Fatal("publication reader was not retained for reuse")
	}
	if err := transport.Close(ctx); err != nil || transport.publicationReader != nil {
		t.Fatalf("publication reader close: err=%v retained=%v", err, transport.publicationReader != nil)
	}
}

func TestPublicationEventNativeWriteUsesCallerTransaction(t *testing.T) {
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "alice"})
	db, err := sql.Open("sqlite", "file:publication_event_native_write?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	if _, err := db.ExecContext(ctx, `CREATE TABLE report_publication_events (
		event_id TEXT PRIMARY KEY, report_id TEXT NOT NULL, owner_id TEXT NOT NULL,
		operation TEXT NOT NULL, version_no INTEGER, generation_no INTEGER,
		status TEXT NOT NULL, requested_by TEXT NOT NULL, reason TEXT,
		failure_code TEXT, failure_message TEXT, occurred_at TIMESTAMP NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db}
	event := publicationEventRecord{ReportID: "r1", OwnerID: "alice", Operation: "publish",
		Status: "succeeded", RequestedBy: "spoofed"}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := transport.appendPublicationEventTx(ctx, tx, event); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM report_publication_events`).Scan(&count); err != nil || count != 1 {
		_ = tx.Rollback()
		t.Fatalf("caller transaction count=%d err=%v", count, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM report_publication_events`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("rolled-back count=%d err=%v", count, err)
	}
	if err := transport.appendPublicationEvent(ctx, event); err != nil {
		t.Fatal(err)
	}
	var version, reason sql.NullString
	var requestedBy string
	if err := db.QueryRowContext(ctx, `SELECT version_no, reason, requested_by FROM report_publication_events`).Scan(&version, &reason, &requestedBy); err != nil {
		t.Fatal(err)
	}
	if version.Valid || reason.Valid || requestedBy != "alice" {
		t.Fatalf("event fields: version=%v reason=%v requestedBy=%q", version, reason, requestedBy)
	}
	if err := transport.appendPublicationEvent(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM report_publication_events WHERE requested_by=?`, sdk.SystemPrincipal().Subject).Scan(&count); err != nil || count != 1 {
		t.Fatalf("system event count=%d err=%v", count, err)
	}
}
