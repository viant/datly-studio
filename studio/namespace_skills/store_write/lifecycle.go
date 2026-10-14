package store_write

import (
	context "context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
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

func (hooks *SkillStoreRules) Init(ctx context.Context, entity *StoredSkill, state xhandler.LifecycleContext[StoredSkill, xhandler.NoParent, Output]) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if entity == nil || entity.NamespaceId == "" || entity.SkillId == "" || entity.Name == "" || entity.Title == "" || entity.Description == "" || entity.Etag == nil {
		return fmt.Errorf("complete namespace skill identity and metadata are required")
	}
	if entity.CreatedAt.IsZero() || entity.UpdatedAt.IsZero() || entity.CreatedBy == "" || entity.UpdatedBy == "" {
		return fmt.Errorf("namespace skill audit fields are required")
	}
	if entity.LatestRevision < 1 || entity.PublishedRevision < 0 || entity.PublishedRevision > entity.LatestRevision {
		return fmt.Errorf("invalid namespace skill revision")
	}
	if entity.Status != "active" && entity.Status != "archived" {
		return fmt.Errorf("invalid namespace skill status")
	}
	if entity.DeletedAt != nil && (entity.Status != "archived" || entity.PublishedRevision != 0) {
		return fmt.Errorf("archived skill must be unpublished")
	}
	previous := state.Previous
	if previous == nil {
		if *entity.Etag != 1 || entity.LatestRevision != 1 || entity.PublishedRevision != 0 || entity.Status != "active" || entity.DeletedAt != nil {
			return fmt.Errorf("new namespace skill requires revision 1, etag 1 and unpublished active state")
		}
		return requireRevision(entity)
	}
	if previous.Etag == nil || *entity.Etag != *previous.Etag {
		return &xhandler.Conflict{Entity: "namespace skill", Field: "etag", Reason: "expected token does not match"}
	}
	if previous.DeletedAt != nil {
		return fmt.Errorf("archived namespace skill cannot be changed")
	}
	if entity.Name != previous.Name || entity.CreatedBy != previous.CreatedBy || !entity.CreatedAt.Equal(previous.CreatedAt) {
		return fmt.Errorf("namespace skill name and creation audit are immutable")
	}
	if entity.LatestRevision == previous.LatestRevision+1 {
		if err := requireRevision(entity); err != nil {
			return err
		}
	} else if entity.LatestRevision != previous.LatestRevision || len(entity.Revisions) != 0 {
		return fmt.Errorf("skill edits must append exactly one new revision")
	} else if entity.Title != previous.Title || entity.Description != previous.Description {
		return fmt.Errorf("skill metadata edits require a new revision")
	}
	next := *entity.Etag + 1
	if next <= 0 {
		return fmt.Errorf("namespace skill token exhausted")
	}
	entity.SetEtag(&next)
	return nil
}

func requireRevision(entity *StoredSkill) error {
	if len(entity.Revisions) != 1 || entity.Revisions[0] == nil || entity.Revisions[0].Revision != entity.LatestRevision {
		return fmt.Errorf("skill edit requires its exact new content revision")
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

// SkillRevisionRules customizes role Input.Skills.Revisions.
type SkillRevisionRules struct{}

func SkillRevisionRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*SkillRevisionRules)(nil)).Elem()
}

var (
	SkillRevisionRulesHooks = new(SkillRevisionRules)
	SkillRevisionRulesDatly = SkillRevisionRulesDatlyType()
)

func (hooks *SkillRevisionRules) Init(ctx context.Context, entity *StoredRevision, state xhandler.LifecycleContext[StoredRevision, StoredSkill, Output]) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if entity == nil || state.Parent == nil || state.Previous != nil {
		return fmt.Errorf("namespace skill revisions are append-only")
	}
	if entity.NamespaceId != state.Parent.NamespaceId || entity.SkillId != state.Parent.SkillId || entity.Revision != state.Parent.LatestRevision {
		return fmt.Errorf("revision must belong to its namespace skill and new revision number")
	}
	if !json.Valid(entity.BundleJson) {
		return fmt.Errorf("skill revision bundle must be valid JSON")
	}
	digest := sha256.Sum256(entity.BundleJson)
	if entity.ContentSha256 != hex.EncodeToString(digest[:]) {
		return fmt.Errorf("skill revision content digest does not match")
	}
	if entity.CreatedAt.IsZero() || entity.CreatedBy == "" || !entity.UpdatedAt.Equal(entity.CreatedAt) || entity.UpdatedBy != entity.CreatedBy {
		return fmt.Errorf("immutable skill revision requires consistent creation audit")
	}
	return nil
}

func (hooks *SkillRevisionRules) Validate(ctx context.Context, entity *StoredRevision, state xhandler.LifecycleContext[StoredRevision, StoredSkill, Output]) error {
	return nil
}
func (hooks *SkillRevisionRules) AfterSequence(ctx context.Context, entity *StoredRevision, state xhandler.LifecycleContext[StoredRevision, StoredSkill, Output]) error {
	return nil
}
func (hooks *SkillRevisionRules) AfterQueue(ctx context.Context, entity *StoredRevision, state xhandler.LifecycleContext[StoredRevision, StoredSkill, Output]) error {
	return nil
}
