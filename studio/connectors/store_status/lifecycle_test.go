package store_status

import (
	"context"
	"errors"
	"testing"
	"time"

	xhandler "github.com/viant/xdatly/handler"
)

func TestConnectorStatusRulesRequireCurrentEtagAndPassedProbe(t *testing.T) {
	now := time.Now().UTC()
	etag := int64(3)
	passed := "passed"
	previous := &StoredConnector{Name: "source", Status: "draft", Etag: &etag, LastTestStatus: &passed}
	state := xhandler.LifecycleContext[StoredConnector, xhandler.NoParent, Output]{EntityState: xhandler.EntityState[StoredConnector, xhandler.NoParent]{Previous: previous}}
	row := func() *StoredConnector {
		current := etag
		return &StoredConnector{Name: "source", Etag: &current, UpdatedAt: &now,
			Has: &StoredConnectorHas{Name: true, Etag: true, UpdatedAt: true}}
	}
	activate := &ConnectorStatusRules{Input: &Input{Operation: "activate"}}
	active := row()
	if err := activate.Init(context.Background(), active, state); err != nil || active.Status != "active" || *active.Etag != etag+1 || !active.Has.Status || !active.Has.Etag {
		t.Fatalf("activated row=%+v err=%v", active, err)
	}
	previous.LastTestStatus = nil
	if err := activate.Init(context.Background(), row(), state); !errors.Is(err, ErrProbeRequired) {
		t.Fatalf("activation without passed probe: %v", err)
	}
	disable := &ConnectorStatusRules{Input: &Input{Operation: "disable"}}
	disabled := row()
	if err := disable.Init(context.Background(), disabled, state); err != nil || disabled.Status != "disabled" || *disabled.Etag != etag+1 {
		t.Fatalf("disabled row=%+v err=%v", disabled, err)
	}
	stale := row()
	*stale.Etag--
	var conflict *xhandler.Conflict
	if err := disable.Init(context.Background(), stale, state); !errors.As(err, &conflict) {
		t.Fatalf("stale etag error=%v", err)
	}
	previous.DeletedAt = &now
	if err := disable.Init(context.Background(), row(), state); !errors.As(err, &conflict) {
		t.Fatalf("deleted connector error=%v", err)
	}
}
