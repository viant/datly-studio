package store_recover

import (
	"context"
	"errors"
	"testing"
	"time"

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

func TestPublicationRecoverRulesCompensation(t *testing.T) {
	stagedGeneration, oldGeneration := int64(3), int64(1)
	desiredVersion := 1
	now := time.Now().UTC()
	revision := "rev-old"
	failure := `[{"code":"reload"}]`
	previous := &StoredPublication{ReportId: "report", DesiredGeneration: &stagedGeneration,
		ActiveGeneration: &oldGeneration, ActiveVersionNo: 1, PublicationStatus: "pending"}
	state := xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]{
		EntityState: xhandler.EntityState[StoredPublication, xhandler.NoParent]{Previous: previous}}
	row := func() *StoredPublication {
		generation := oldGeneration
		return &StoredPublication{ReportId: "report", ActiveVersionNo: 1,
			DesiredVersionNo: &desiredVersion, DesiredGeneration: &generation,
			ActiveGeneration: &oldGeneration, PublicationStatus: "pending",
			RuntimeRevision: &revision, SpecHash: "old-spec", PublishedBy: "owner",
			PublishedAt: &now, ActivatedAt: &now, FailureJson: &failure,
			Has: &StoredPublicationHas{ReportId: true, ActiveVersionNo: true,
				DesiredVersionNo: true, DesiredGeneration: true, ActiveGeneration: true,
				PublicationStatus: true, RuntimeRevision: true, SpecHash: true,
				PublishedBy: true, PublishedAt: true, ActivatedAt: true, FailureJson: true}}
	}
	rules := &PublicationRecoverRules{Input: &Input{Operation: "compensate_restore",
		StagedGeneration: stagedGeneration, RestoreStatus: "active"}}
	valid := row()
	if err := rules.Init(context.Background(), valid, state); err != nil || valid.PublicationStatus != "active" ||
		valid.DesiredGeneration == nil || *valid.DesiredGeneration != oldGeneration {
		t.Fatalf("compensated publication=%+v err=%v", valid, err)
	}
	rules.Input.StagedGeneration++
	var conflict *xhandler.Conflict
	if err := rules.Init(context.Background(), row(), state); !errors.As(err, &conflict) {
		t.Fatalf("stale compensation generation error=%v", err)
	}
	rules.Input.StagedGeneration = stagedGeneration
	incomplete := row()
	incomplete.Has.PublishedBy = false
	if err := rules.Init(context.Background(), incomplete, state); err == nil {
		t.Fatal("compensation accepted an incomplete previous snapshot")
	}
	previous.ActiveGeneration = nil
	initial := &PublicationRecoverRules{Input: &Input{Operation: "compensate_initial", StagedGeneration: stagedGeneration}}
	initialRow := &StoredPublication{ReportId: "report", DesiredGeneration: &stagedGeneration,
		PublicationStatus: "pending", FailureJson: &failure,
		Has: &StoredPublicationHas{ReportId: true, DesiredGeneration: true,
			PublicationStatus: true, FailureJson: true}}
	if err := initial.Init(context.Background(), initialRow, state); err != nil || initialRow.PublicationStatus != "failed" {
		t.Fatalf("initial compensation=%+v err=%v", initialRow, err)
	}
	previous.ActiveGeneration = &oldGeneration
	initialRow.PublicationStatus = "pending"
	if err := initial.Init(context.Background(), initialRow, state); err == nil {
		t.Fatal("initial compensation accepted an active publication")
	}
}
