package store_validation

import (
	"context"
	"errors"
	"testing"
	"time"

	xhandler "github.com/viant/xdatly/handler"
)

func TestVersionValidationRulesMatchSourceWithoutAdvancingIt(t *testing.T) {
	expected := int64(4)
	now := time.Now().UTC()
	previous := &StoredVersion{ReportId: "report", VersionNo: 2, SourceRevision: &expected}
	state := xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]{
		EntityState: xhandler.EntityState[StoredVersion, xhandler.NoParent]{Previous: previous}}
	row := func() *StoredVersion {
		version := expected
		return &StoredVersion{ReportId: "report", VersionNo: 2, CompileStatus: "valid",
			CompileDiagnosticsJson: []byte(`[]`), ValidatedAt: &now, SourceRevision: &version,
			Has: &StoredVersionHas{ReportId: true, VersionNo: true, CompileStatus: true,
				CompileDiagnosticsJson: true, ValidatedAt: true, SourceRevision: true}}
	}
	rules := &VersionValidationRules{}
	valid := row()
	if err := rules.Init(context.Background(), valid, state); err != nil || *valid.SourceRevision != expected {
		t.Fatalf("validation row=%+v err=%v", valid, err)
	}
	stale := row()
	*stale.SourceRevision--
	var conflict *xhandler.Conflict
	if err := rules.Init(context.Background(), stale, state); !errors.As(err, &conflict) {
		t.Fatalf("stale validation error=%v", err)
	}
	malformed := row()
	malformed.CompileDiagnosticsJson = []byte(`invalid`)
	if err := rules.Init(context.Background(), malformed, state); err == nil {
		t.Fatal("malformed diagnostics accepted")
	}
}
