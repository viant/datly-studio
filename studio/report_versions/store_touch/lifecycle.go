package store_touch

import (
	context "context"
	"encoding/json"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	"strings"
)

// VersionTouchRules customizes role Input.Versions.
type VersionTouchRules struct{}

func VersionTouchRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*VersionTouchRules)(nil)).Elem()
}

var (
	VersionTouchRulesHooks = new(VersionTouchRules)
	VersionTouchRulesDatly = VersionTouchRulesDatlyType()
)

func (hooks *VersionTouchRules) Init(_ context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	if entity == nil || strings.TrimSpace(entity.ReportId) == "" || entity.VersionNo <= 0 ||
		entity.SourceRevision == nil || *entity.SourceRevision <= 0 ||
		entity.Has == nil || !entity.Has.ReportId || !entity.Has.VersionNo || !entity.Has.SourceRevision {
		return fmt.Errorf("resource mutation requires exact report, version and source revision")
	}
	previous := state.Previous
	if previous == nil || previous.SourceRevision == nil || *previous.SourceRevision != *entity.SourceRevision {
		return &xhandler.Conflict{Entity: "report_version", Field: "source_revision", Reason: "expected source revision does not match"}
	}
	if previous.State != "draft" && previous.State != "validated" && previous.State != "failed" {
		return &xhandler.Conflict{Entity: "report_version", Field: "state", Reason: "published versions cannot be mutated"}
	}
	entity.SetState("draft")
	entity.SetCompileStatus("pending")
	entity.SetCompileDiagnosticsJson(json.RawMessage(`[]`))
	entity.SetValidatedAt(nil)
	next := *entity.SourceRevision + 1
	entity.SetSourceRevision(&next)
	return nil
}
func (hooks *VersionTouchRules) Validate(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *VersionTouchRules) AfterSequence(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *VersionTouchRules) AfterQueue(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *VersionTouchRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
