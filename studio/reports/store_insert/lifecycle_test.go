package store_insert

import (
	"context"
	"testing"
	"time"

	xhandler "github.com/viant/xdatly/handler"
)

func TestReportInsertRulesRequireDraftIdentity(t *testing.T) {
	now := time.Now().UTC()
	row := &StoredReport{Id: "report-1", Namespace: "general", Slug: "report-1", Title: "Report 1",
		OwnerId: "owner-1", Status: "draft", DefaultConnectorName: "main",
		ComponentScope: "reports/owner-1/report-1", ComponentName: "reader",
		Etag: 1, CreatedAt: now, UpdatedAt: now}
	rules := &ReportInsertRules{}
	state := xhandler.LifecycleContext[StoredReport, xhandler.NoParent, Output]{}
	if err := rules.Init(context.Background(), row, state); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*StoredReport){
		func(value *StoredReport) { value.OwnerId = "" },
		func(value *StoredReport) { value.Status = "active" },
		func(value *StoredReport) { value.Etag = 2 },
		func(value *StoredReport) { value.DefaultConnectorName = "" },
	} {
		invalid := *row
		mutate(&invalid)
		if err := rules.Init(context.Background(), &invalid, state); err == nil {
			t.Fatalf("invalid report accepted: %+v", invalid)
		}
	}
}
