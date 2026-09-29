package store_write

import (
	"context"
	"errors"
	"strings"
	"testing"

	xhandler "github.com/viant/xdatly/handler"
)

func TestSkillStoreRulesRequireFolderPathAndDeleteIdentity(t *testing.T) {
	rules := &SkillStoreRules{}
	row := func() *StoredSkill {
		return &StoredSkill{NamespaceId: strings.Repeat("a", 64), ReportId: "report", VersionNo: 2, SkillId: "skill",
			FolderId: "folder", SkillRoot: ".",
			Has: &StoredSkillHas{NamespaceId: true, ReportId: true, VersionNo: true, SkillId: true,
				FolderId: true, SkillRoot: true, Ordinal: true, ShouldDelete: true}}
	}
	state := xhandler.LifecycleContext[StoredSkill, xhandler.NoParent, Output]{}
	if err := rules.Init(context.Background(), row(), state); err != nil {
		t.Fatalf("valid skill root rejected: %v", err)
	}
	foreignState := xhandler.LifecycleContext[StoredSkill, xhandler.NoParent, Output]{
		EntityState: xhandler.EntityState[StoredSkill, xhandler.NoParent]{Previous: &StoredSkill{NamespaceId: strings.Repeat("b", 64)}}}
	if err := rules.Init(context.Background(), row(), foreignState); err == nil {
		t.Fatal("resource ownership move was accepted")
	}
	invalid := row()
	invalid.SkillRoot = "../outside"
	if err := rules.Init(context.Background(), invalid, state); err == nil {
		t.Fatal("unsafe skill root accepted")
	}
	missing := &StoredSkill{NamespaceId: strings.Repeat("a", 64), ReportId: "report", VersionNo: 2, SkillId: "missing", ShouldDelete: true,
		Has: &StoredSkillHas{NamespaceId: true, ReportId: true, VersionNo: true, SkillId: true, ShouldDelete: true}}
	var conflict *xhandler.Conflict
	if err := rules.Init(context.Background(), missing, state); !errors.As(err, &conflict) {
		t.Fatalf("missing skill delete error=%v", err)
	}
}
