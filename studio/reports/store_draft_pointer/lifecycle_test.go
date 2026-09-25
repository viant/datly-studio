package store_draft_pointer

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	xhandler "github.com/viant/xdatly/handler"
)

func pointer(id string, draft int, etag int64) *DraftPointer {
	now := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	row := &DraftPointer{Id: id, CurrentDraftVersion: &draft, Etag: &etag, UpdatedAt: &now, Has: &DraftPointerHas{}}
	return row
}

func previous(draft *int, etag int64) xhandler.LifecycleContext[DraftPointer, xhandler.NoParent, Output] {
	state := xhandler.LifecycleContext[DraftPointer, xhandler.NoParent, Output]{}
	state.Previous = &DraftPointer{Id: "r1", CurrentDraftVersion: draft, Etag: &etag}
	return state
}

func TestDraftPointerRulesAdvanceEtagByExactlyOne(t *testing.T) {
	hooks := new(DraftPointerRules)
	ctx := context.Background()
	one := 1
	row := pointer("r1", 2, 7)
	state := previous(&one, 7)
	if err := hooks.Init(ctx, row, state); err != nil {
		t.Fatal(err)
	}
	if row.Etag == nil || *row.Etag != 8 || !row.Has.Etag {
		t.Fatalf("etag=%v has=%+v", row.Etag, row.Has)
	}
	if err := hooks.Validate(ctx, row, state); err != nil {
		t.Fatal(err)
	}
	first := pointer("r1", 1, 1)
	unset := previous(nil, 1)
	if err := hooks.Init(ctx, first, unset); err != nil {
		t.Fatal(err)
	}
	if err := hooks.Validate(ctx, first, unset); err != nil {
		t.Fatalf("first draft pointer denied: %v", err)
	}
}

func TestDraftPointerRulesDenyMissingStaleAndRegressingPointers(t *testing.T) {
	hooks := new(DraftPointerRules)
	ctx := context.Background()
	one, two := 1, 2
	var conflict *xhandler.Conflict
	if err := hooks.Init(ctx, pointer("r1", 2, 7), xhandler.LifecycleContext[DraftPointer, xhandler.NoParent, Output]{}); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("missing report error=%v", err)
	}
	if err := hooks.Init(ctx, pointer(" ", 2, 7), previous(&one, 7)); err == nil || !strings.Contains(err.Error(), "report id is required") {
		t.Fatalf("blank id error=%v", err)
	}
	noEtag := pointer("r1", 2, 7)
	noEtag.Etag = nil
	if err := hooks.Init(ctx, noEtag, previous(&one, 7)); !errors.As(err, &conflict) {
		t.Fatalf("missing etag error=%v", err)
	}
	if err := hooks.Init(ctx, pointer("r1", 0, 7), previous(&one, 7)); err == nil || !strings.Contains(err.Error(), "draft version is required") {
		t.Fatalf("zero draft error=%v", err)
	}
	noTime := pointer("r1", 2, 7)
	noTime.UpdatedAt = nil
	if err := hooks.Init(ctx, noTime, previous(&one, 7)); err == nil || !strings.Contains(err.Error(), "updated_at is required") {
		t.Fatalf("missing updated_at error=%v", err)
	}
	stale := pointer("r1", 2, 5)
	state := previous(&one, 7)
	if err := hooks.Init(ctx, stale, state); err != nil {
		t.Fatal(err)
	}
	if err := hooks.Validate(ctx, stale, state); !errors.As(err, &conflict) || conflict.Reason != "report etag is stale" {
		t.Fatalf("stale etag error=%v", err)
	}
	regress := pointer("r1", 2, 7)
	state = previous(&two, 7)
	if err := hooks.Init(ctx, regress, state); err != nil {
		t.Fatal(err)
	}
	if err := hooks.Validate(ctx, regress, state); !errors.As(err, &conflict) || !strings.Contains(conflict.Reason, "must advance") {
		t.Fatalf("regressing pointer error=%v", err)
	}
}
