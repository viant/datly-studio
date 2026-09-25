package store_touch

import (
	"context"
	"errors"
	"testing"

	xhandler "github.com/viant/xdatly/handler"
)

func TestVersionTouchRulesRequireExactEditableRevision(t *testing.T) {
	revision := int64(4)
	previous := &StoredVersion{ReportId: "report", VersionNo: 2, State: "validated", SourceRevision: &revision}
	state := xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]{
		EntityState: xhandler.EntityState[StoredVersion, xhandler.NoParent]{Previous: previous}}
	row := func() *StoredVersion {
		expected := revision
		return &StoredVersion{ReportId: "report", VersionNo: 2, SourceRevision: &expected,
			Has: &StoredVersionHas{ReportId: true, VersionNo: true, SourceRevision: true}}
	}
	rules := &VersionTouchRules{}
	valid := row()
	if err := rules.Init(context.Background(), valid, state); err != nil || valid.State != "draft" ||
		valid.CompileStatus != "pending" || string(valid.CompileDiagnosticsJson) != "[]" ||
		valid.SourceRevision == nil || *valid.SourceRevision != revision+1 ||
		!valid.Has.ValidatedAt || valid.ValidatedAt != nil {
		t.Fatalf("touched version=%+v err=%v", valid, err)
	}
	stale := row()
	*stale.SourceRevision--
	var conflict *xhandler.Conflict
	if err := rules.Init(context.Background(), stale, state); !errors.As(err, &conflict) {
		t.Fatalf("stale revision error=%v", err)
	}
	previous.State = "published"
	if err := rules.Init(context.Background(), row(), state); !errors.As(err, &conflict) {
		t.Fatalf("published version mutation error=%v", err)
	}
}
