package store_state

import (
	"context"
	"errors"
	"testing"
	"time"

	xhandler "github.com/viant/xdatly/handler"
)

func TestGenerationStateRulesActivateAndRetireWithStatusToken(t *testing.T) {
	now := time.Now().UTC()
	count := 2
	state := xhandler.LifecycleContext[StoredGeneration, xhandler.NoParent, Output]{
		EntityState: xhandler.EntityState[StoredGeneration, xhandler.NoParent]{Previous: &StoredGeneration{GenerationNo: 2, Status: "building"}}}
	activate := &GenerationStateRules{Input: &Input{Operation: "activate", TargetGeneration: 2}}
	row := &StoredGeneration{GenerationNo: 2, Status: "building", ReportCount: &count, ActivatedAt: &now,
		Has: &StoredGenerationHas{GenerationNo: true, Status: true, ReportCount: true, ActivatedAt: true}}
	if err := activate.Init(context.Background(), row, state); err != nil || row.Status != "active" || !row.Has.Status {
		t.Fatalf("activated row=%+v err=%v", row, err)
	}
	stale := &StoredGeneration{GenerationNo: 2, Status: "building", ReportCount: &count, ActivatedAt: &now,
		Has: &StoredGenerationHas{GenerationNo: true, Status: true, ReportCount: true, ActivatedAt: true}}
	state.Previous = &StoredGeneration{GenerationNo: 2, Status: "active"}
	var conflict *xhandler.Conflict
	if err := activate.Init(context.Background(), stale, state); !errors.As(err, &conflict) {
		t.Fatalf("stale status error=%v", err)
	}
	retire := &GenerationStateRules{Input: &Input{Operation: "retire", TargetGeneration: 2}}
	state.Previous = &StoredGeneration{GenerationNo: 1, Status: "active"}
	older := &StoredGeneration{GenerationNo: 1, Status: "active", RetiredAt: &now,
		Has: &StoredGenerationHas{GenerationNo: true, Status: true, RetiredAt: true}}
	if err := retire.Init(context.Background(), older, state); err != nil || older.Status != "retired" || !older.Has.RetiredAt {
		t.Fatalf("retired row=%+v err=%v", older, err)
	}
}
