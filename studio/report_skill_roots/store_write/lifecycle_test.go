package store_write

import (
	"context"
	"errors"
	"testing"

	xhandler "github.com/viant/xdatly/handler"
)

func TestSkillStoreRulesRequireFolderPathAndDeleteIdentity(t *testing.T) {
	rules := &SkillStoreRules{}
	row := func() *StoredSkill {
		return &StoredSkill{ReportId: "report", VersionNo: 2, SkillId: "skill",
			FolderId: "folder", SkillRoot: ".",
			Has: &StoredSkillHas{ReportId: true, VersionNo: true, SkillId: true,
				FolderId: true, SkillRoot: true, Ordinal: true, ShouldDelete: true}}
	}
	state := xhandler.LifecycleContext[StoredSkill, xhandler.NoParent, Output]{}
	if err := rules.Init(context.Background(), row(), state); err != nil {
		t.Fatalf("valid skill root rejected: %v", err)
	}
	invalid := row()
	invalid.SkillRoot = "../outside"
	if err := rules.Init(context.Background(), invalid, state); err == nil {
		t.Fatal("unsafe skill root accepted")
	}
	missing := &StoredSkill{ReportId: "report", VersionNo: 2, SkillId: "missing", ShouldDelete: true,
		Has: &StoredSkillHas{ReportId: true, VersionNo: true, SkillId: true, ShouldDelete: true}}
	var conflict *xhandler.Conflict
	if err := rules.Init(context.Background(), missing, state); !errors.As(err, &conflict) {
		t.Fatalf("missing skill delete error=%v", err)
	}
}
