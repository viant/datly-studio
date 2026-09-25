package store_recover

import (
	"context"
	"errors"
	"testing"

	xhandler "github.com/viant/xdatly/handler"
)

func TestPublicationRecoverRulesRestoreAndFail(t *testing.T) {
	expected := int64(3)
	active := int64(1)
	revision := "rev-active"
	diagnostics := `[{"code":"expired"}]`
	previous := &StoredPublication{ReportId: "report", DesiredGeneration: &expected,
		ActiveGeneration: &active, ActiveVersionNo: 2, PublicationStatus: "unpublishing"}
	state := xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]{
		EntityState: xhandler.EntityState[StoredPublication, xhandler.NoParent]{Previous: previous}}
	row := func() *StoredPublication {
		generation := expected
		return &StoredPublication{ReportId: "report", DesiredGeneration: &generation,
			PublicationStatus: "unpublishing", FailureJson: &diagnostics,
			Has: &StoredPublicationHas{ReportId: true, DesiredGeneration: true,
				PublicationStatus: true, FailureJson: true}}
	}
	restore := &PublicationRecoverRules{Input: &Input{Operation: "restore_active"}}
	valid := row()
	valid.RuntimeRevision = &revision
	valid.SpecHash = "active-spec"
	valid.Has.RuntimeRevision, valid.Has.SpecHash = true, true
	if err := restore.Init(context.Background(), valid, state); err != nil || valid.PublicationStatus != "active" ||
		valid.DesiredVersionNo == nil || *valid.DesiredVersionNo != 2 ||
		valid.DesiredGeneration == nil || *valid.DesiredGeneration != 1 {
		t.Fatalf("restored row=%+v err=%v", valid, err)
	}
	stale := row()
	stale.PublicationStatus = "pending"
	stale.RuntimeRevision, stale.SpecHash = &revision, "active-spec"
	stale.Has.RuntimeRevision, stale.Has.SpecHash = true, true
	var conflict *xhandler.Conflict
	if err := restore.Init(context.Background(), stale, state); !errors.As(err, &conflict) {
		t.Fatalf("stale publication status error=%v", err)
	}
	previous.ActiveGeneration = nil
	previous.PublicationStatus = "pending"
	fail := &PublicationRecoverRules{Input: &Input{Operation: "fail"}}
	failed := row()
	failed.PublicationStatus = "pending"
	if err := fail.Init(context.Background(), failed, state); err != nil || failed.PublicationStatus != "failed" {
		t.Fatalf("failed row=%+v err=%v", failed, err)
	}
	failed = row()
	failed.PublicationStatus = "pending"
	badJSON := "not-json"
	failed.FailureJson = &badJSON
	if err := fail.Init(context.Background(), failed, state); err == nil {
		t.Fatal("recovery accepted invalid failure evidence")
	}
}
