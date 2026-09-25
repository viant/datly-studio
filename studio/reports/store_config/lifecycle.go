package store_config

import (
	context "context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
)

// ReportConfigRules customizes role Input.Reports.
type ReportConfigRules struct{}

func ReportConfigRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*ReportConfigRules)(nil)).Elem()
}

var (
	ReportConfigRulesHooks = new(ReportConfigRules)
	ReportConfigRulesDatly = ReportConfigRulesDatlyType()
)

func (hooks *ReportConfigRules) Init(_ context.Context, entity *StoredReport, state xhandler.LifecycleContext[StoredReport, xhandler.NoParent, Output]) error {
	if entity == nil || entity.Id == "" || entity.Etag == nil || entity.UpdatedAt == nil ||
		entity.Has == nil || !entity.Has.Id || !entity.Has.Namespace || !entity.Has.Slug ||
		!entity.Has.Title || !entity.Has.Description || !entity.Has.OwnerId ||
		!entity.Has.Status || !entity.Has.DefaultConnectorName || !entity.Has.ComponentScope ||
		!entity.Has.ComponentName || !entity.Has.CurrentDraftVersion || !entity.Has.Etag ||
		!entity.Has.UpdatedAt {
		return fmt.Errorf("report config requires a complete snapshot and expected etag")
	}
	previous := state.Previous
	if previous == nil || previous.DeletedAt != nil {
		return &xhandler.Conflict{Entity: "report", Field: "id", Reason: "report is absent or deleted"}
	}
	if previous.Etag == nil || *previous.Etag != *entity.Etag {
		return &xhandler.Conflict{Entity: "report", Field: "etag", Reason: "expected etag does not match"}
	}
	if previous.OwnerId != entity.OwnerId {
		return &xhandler.Conflict{Entity: "report", Field: "owner_id", Reason: "report owner cannot change"}
	}
	next := *entity.Etag + 1
	entity.SetEtag(&next)
	return nil
}
func (hooks *ReportConfigRules) Validate(ctx context.Context, entity *StoredReport, state xhandler.LifecycleContext[StoredReport, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ReportConfigRules) AfterSequence(ctx context.Context, entity *StoredReport, state xhandler.LifecycleContext[StoredReport, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ReportConfigRules) AfterQueue(ctx context.Context, entity *StoredReport, state xhandler.LifecycleContext[StoredReport, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ReportConfigRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
