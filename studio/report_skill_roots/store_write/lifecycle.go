package store_write

import (
	context "context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	"io/fs"
	reflect "reflect"
	"strings"
)

// SkillStoreRules customizes role Input.Skills.
type SkillStoreRules struct{}

func SkillStoreRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*SkillStoreRules)(nil)).Elem()
}

var (
	SkillStoreRulesHooks = new(SkillStoreRules)
	SkillStoreRulesDatly = SkillStoreRulesDatlyType()
)

func (hooks *SkillStoreRules) Init(_ context.Context, entity *StoredSkill, state xhandler.LifecycleContext[StoredSkill, xhandler.NoParent, Output]) error {
	if entity == nil || strings.TrimSpace(entity.ReportId) == "" || entity.VersionNo <= 0 ||
		strings.TrimSpace(entity.SkillId) == "" || entity.Has == nil ||
		!entity.Has.ReportId || !entity.Has.VersionNo || !entity.Has.SkillId ||
		!entity.Has.ShouldDelete {
		return fmt.Errorf("skill root mutation requires complete identity and explicit delete marker")
	}
	if entity.ShouldDelete {
		if state.Previous == nil {
			return &xhandler.Conflict{Entity: "skill_root", Field: "skill_id", Reason: "skill root does not exist"}
		}
		return nil
	}
	if !entity.Has.FolderId || !entity.Has.SkillRoot || !entity.Has.Ordinal ||
		strings.TrimSpace(entity.FolderId) == "" ||
		entity.SkillRoot != "." && (!fs.ValidPath(entity.SkillRoot) || strings.Contains(entity.SkillRoot, "\\")) {
		return fmt.Errorf("skill root requires a complete valid folder and relative path")
	}
	return nil
}
func (hooks *SkillStoreRules) Validate(ctx context.Context, entity *StoredSkill, state xhandler.LifecycleContext[StoredSkill, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *SkillStoreRules) AfterSequence(ctx context.Context, entity *StoredSkill, state xhandler.LifecycleContext[StoredSkill, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *SkillStoreRules) AfterQueue(ctx context.Context, entity *StoredSkill, state xhandler.LifecycleContext[StoredSkill, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *SkillStoreRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
