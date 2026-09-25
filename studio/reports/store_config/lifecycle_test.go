package store_config

import (
	"context"
	"errors"
	"testing"
	"time"

	xhandler "github.com/viant/xdatly/handler"
)

func TestReportConfigRulesMatchEtagAndPreserveOwner(t *testing.T) {
	now := time.Now().UTC()
	etag := int64(3)
	previous := &StoredReport{Id: "report-1", OwnerId: "owner-1", Etag: &etag}
	state := xhandler.LifecycleContext[StoredReport, xhandler.NoParent, Output]{
		EntityState: xhandler.EntityState[StoredReport, xhandler.NoParent]{Previous: previous}}
	row := func() *StoredReport {
		version := etag
		return &StoredReport{Id: "report-1", OwnerId: "owner-1", Etag: &version, UpdatedAt: &now,
			Has: &StoredReportHas{Id: true, Namespace: true, Slug: true, Title: true,
				Description: true, OwnerId: true, Status: true, DefaultConnectorName: true,
				ComponentScope: true, ComponentName: true, CurrentDraftVersion: true,
				Etag: true, UpdatedAt: true}}
	}
	rules := &ReportConfigRules{}
	valid := row()
	if err := rules.Init(context.Background(), valid, state); err != nil || *valid.Etag != etag+1 {
		t.Fatalf("valid update=%+v err=%v", valid, err)
	}
	stale := row()
	*stale.Etag--
	var conflict *xhandler.Conflict
	if err := rules.Init(context.Background(), stale, state); !errors.As(err, &conflict) {
		t.Fatalf("stale update error=%v", err)
	}
	otherOwner := row()
	otherOwner.OwnerId = "owner-2"
	if err := rules.Init(context.Background(), otherOwner, state); !errors.As(err, &conflict) {
		t.Fatalf("owner change error=%v", err)
	}
	previous.DeletedAt = &now
	if err := rules.Init(context.Background(), row(), state); !errors.As(err, &conflict) {
		t.Fatalf("deleted report error=%v", err)
	}
}
