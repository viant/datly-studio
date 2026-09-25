package store_activate

import (
	"context"
	"errors"
	"testing"
	"time"

	xhandler "github.com/viant/xdatly/handler"
)

func TestPublicationActivationRulesRequireExactPendingVersion(t *testing.T) {
	generation := int64(2)
	version := 2
	failure := `[{"code":"old"}]`
	previous := &StoredPublication{ReportId: "report", DesiredGeneration: &generation,
		DesiredVersionNo: &version, PublicationStatus: "pending", FailureJson: &failure}
	state := xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]{
		EntityState: xhandler.EntityState[StoredPublication, xhandler.NoParent]{Previous: previous}}
	now := time.Now().UTC()
	row := func() *StoredPublication {
		expected := generation
		return &StoredPublication{ReportId: "report", DesiredGeneration: &expected, ActivatedAt: &now,
			Has: &StoredPublicationHas{ReportId: true, DesiredGeneration: true, ActivatedAt: true}}
	}
	rules := &PublicationActivationRules{Input: &Input{VersionNo: version, Generation: generation}}
	valid := row()
	if err := rules.Init(context.Background(), valid, state); err != nil ||
		valid.ActiveVersionNo == nil || *valid.ActiveVersionNo != version ||
		valid.ActiveGeneration == nil || *valid.ActiveGeneration != generation ||
		valid.PublicationStatus != "active" || valid.FailureJson != nil ||
		!valid.Has.ActiveVersionNo || !valid.Has.ActiveGeneration || !valid.Has.PublicationStatus || !valid.Has.FailureJson {
		t.Fatalf("activated publication=%+v err=%v", valid, err)
	}
	wrongVersion := row()
	rules.Input.VersionNo++
	var conflict *xhandler.Conflict
	if err := rules.Init(context.Background(), wrongVersion, state); !errors.As(err, &conflict) {
		t.Fatalf("wrong version error=%v", err)
	}
	rules.Input.VersionNo = version
	previous.PublicationStatus = "active"
	if err := rules.Init(context.Background(), row(), state); !errors.As(err, &conflict) {
		t.Fatalf("not-pending error=%v", err)
	}
}
