package store_state

import (
	"context"
	"errors"
	"testing"
	"time"

	xhandler "github.com/viant/xdatly/handler"
)

func TestVersionStateRulesMatchPublishedState(t *testing.T) {
	now := time.Now().UTC()
	previous := &StoredVersion{ReportId: "report", VersionNo: 1, State: "published"}
	state := xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]{
		EntityState: xhandler.EntityState[StoredVersion, xhandler.NoParent]{Previous: previous}}
	supersede := &VersionStateRules{Input: &Input{Operation: "supersede", TargetVersionNo: 2}}
	row := &StoredVersion{ReportId: "report", VersionNo: 1, State: "published",
		Has: &StoredVersionHas{ReportId: true, VersionNo: true, State: true}}
	if err := supersede.Init(context.Background(), row, state); err != nil || row.State != "superseded" {
		t.Fatalf("superseded row=%+v err=%v", row, err)
	}
	stale := &StoredVersion{ReportId: "report", VersionNo: 1, State: "published",
		Has: &StoredVersionHas{ReportId: true, VersionNo: true, State: true}}
	previous.State = "superseded"
	var conflict *xhandler.Conflict
	if err := supersede.Init(context.Background(), stale, state); !errors.As(err, &conflict) {
		t.Fatalf("stale state error=%v", err)
	}
	previous.VersionNo, previous.State = 2, "validated"
	publish := &VersionStateRules{Input: &Input{Operation: "publish", TargetVersionNo: 2}}
	target := &StoredVersion{ReportId: "report", VersionNo: 2, State: "validated", PublishedAt: &now,
		Has: &StoredVersionHas{ReportId: true, VersionNo: true, State: true, PublishedAt: true}}
	if err := publish.Init(context.Background(), target, state); err != nil || target.State != "published" || !target.Has.PublishedAt {
		t.Fatalf("published target=%+v err=%v", target, err)
	}
	previous.State = "published"
	unpublish := &VersionStateRules{Input: &Input{Operation: "unpublish", TargetVersionNo: 2}}
	withdrawn := &StoredVersion{ReportId: "report", VersionNo: 2, State: "published",
		Has: &StoredVersionHas{ReportId: true, VersionNo: true, State: true}}
	if err := unpublish.Init(context.Background(), withdrawn, state); err != nil || withdrawn.State != "superseded" {
		t.Fatalf("unpublished version=%+v err=%v", withdrawn, err)
	}
}
