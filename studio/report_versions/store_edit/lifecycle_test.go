package store_edit

import (
	"context"
	"errors"
	"strings"
	"testing"

	xhandler "github.com/viant/xdatly/handler"
)

func TestVersionEditRulesAdvanceMatchedSourceRevision(t *testing.T) {
	previousRevision := int64(3)
	previous := &StoredVersion{ReportId: "report", VersionNo: 2, SourceRevision: &previousRevision}
	state := xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]{
		EntityState: xhandler.EntityState[StoredVersion, xhandler.NoParent]{Previous: previous}}
	row := func() *StoredVersion {
		expected := previousRevision
		return &StoredVersion{ReportId: "report", VersionNo: 2, ComponentSpecJson: []byte(`{}`),
			SpecHash: strings.Repeat("a", 64), CompileStatus: "pending", CompileDiagnosticsJson: []byte(`[]`),
			SourceRevision: &expected, Has: &StoredVersionHas{ReportId: true, VersionNo: true,
				AuthoredSql: true, AuthoredDql: true, ComponentSpecJson: true, SpecHash: true,
				GeneratedDql: true, CompileStatus: true, CompileDiagnosticsJson: true, SourceRevision: true}}
	}
	rules := &VersionEditRules{}
	valid := row()
	if err := rules.Init(context.Background(), valid, state); err != nil || *valid.SourceRevision != 4 || !valid.Has.SourceRevision {
		t.Fatalf("matched edit=%+v err=%v", valid, err)
	}
	stale := row()
	*stale.SourceRevision--
	var conflict *xhandler.Conflict
	if err := rules.Init(context.Background(), stale, state); !errors.As(err, &conflict) {
		t.Fatalf("stale edit error=%v", err)
	}
	missing := row()
	state.Previous = nil
	if err := rules.Init(context.Background(), missing, state); !errors.As(err, &conflict) {
		t.Fatalf("missing version error=%v", err)
	}
	state.Previous = previous
	malformed := row()
	malformed.ComponentSpecJson = []byte(`bad`)
	if err := rules.Init(context.Background(), malformed, state); err == nil {
		t.Fatal("malformed spec accepted")
	}
}
