package store_repoint

import (
	"context"
	"errors"
	"testing"

	xhandler "github.com/viant/xdatly/handler"
)

func TestPublicationRepointRulesMatchOtherActiveRow(t *testing.T) {
	previousGeneration := int64(1)
	previous := &StoredPublication{ReportId: "other", ActiveGeneration: &previousGeneration, PublicationStatus: "active"}
	state := xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]{
		EntityState: xhandler.EntityState[StoredPublication, xhandler.NoParent]{Previous: previous}}
	row := func() *StoredPublication {
		expected := previousGeneration
		return &StoredPublication{ReportId: "other", ActiveGeneration: &expected,
			Has: &StoredPublicationHas{ReportId: true, ActiveGeneration: true}}
	}
	rules := &PublicationRepointRules{Input: &Input{ExcludeReportId: "current", Generation: 2}}
	valid := row()
	if err := rules.Init(context.Background(), valid, state); err != nil ||
		valid.ActiveGeneration == nil || *valid.ActiveGeneration != 2 || !valid.Has.ActiveGeneration {
		t.Fatalf("repointed row=%+v err=%v", valid, err)
	}
	stale := row()
	*stale.ActiveGeneration--
	var conflict *xhandler.Conflict
	if err := rules.Init(context.Background(), stale, state); !errors.As(err, &conflict) {
		t.Fatalf("stale other publication error=%v", err)
	}
	previous.PublicationStatus = "pending"
	if err := rules.Init(context.Background(), row(), state); !errors.As(err, &conflict) {
		t.Fatalf("not-active other publication error=%v", err)
	}
	rules.Input.ExcludeReportId = "other"
	if err := rules.Init(context.Background(), row(), state); err == nil {
		t.Fatal("current report was accepted as another publication")
	}
}
