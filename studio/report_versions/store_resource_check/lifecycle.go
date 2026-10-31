package store_resource_check

import (
	context "context"
	"encoding/json"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	"strings"
)

// ResourceCheckRules customizes role Input.Versions.
type ResourceCheckRules struct{}

func ResourceCheckRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*ResourceCheckRules)(nil)).Elem()
}

var (
	ResourceCheckRulesHooks = new(ResourceCheckRules)
	ResourceCheckRulesDatly = ResourceCheckRulesDatlyType()
)

func (hooks *ResourceCheckRules) Init(_ context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	if entity == nil || strings.TrimSpace(entity.ReportId) == "" || entity.VersionNo <= 0 ||
		entity.SourceRevision == nil || *entity.SourceRevision <= 0 ||
		entity.Has == nil || !entity.Has.ReportId || !entity.Has.VersionNo || !entity.Has.SourceRevision {
		return fmt.Errorf("resource mutation requires exact report, version and source revision")
	}
	if !entity.Has.ResourceCheckEnabled {
		return fmt.Errorf("resource_check_enabled must be explicitly supplied")
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
func (hooks *ResourceCheckRules) Validate(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ResourceCheckRules) AfterSequence(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ResourceCheckRules) AfterQueue(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ResourceCheckRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
