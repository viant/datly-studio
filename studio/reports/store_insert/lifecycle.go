package store_insert

import (
	context "context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
)

// ReportInsertRules customizes role Input.Reports.
type ReportInsertRules struct{}

func ReportInsertRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*ReportInsertRules)(nil)).Elem()
}

var (
	ReportInsertRulesHooks = new(ReportInsertRules)
	ReportInsertRulesDatly = ReportInsertRulesDatlyType()
)

func (hooks *ReportInsertRules) Init(_ context.Context, entity *StoredReport, _ xhandler.LifecycleContext[StoredReport, xhandler.NoParent, Output]) error {
	if entity == nil || entity.Id == "" || entity.Namespace == "" || entity.Slug == "" ||
		entity.Title == "" || entity.OwnerId == "" || entity.DefaultConnectorName == "" ||
		entity.ComponentScope == "" || entity.ComponentName == "" || entity.Status != "draft" ||
		entity.Etag != 1 || entity.CreatedAt.IsZero() || entity.UpdatedAt.IsZero() {
		return fmt.Errorf("new report requires identity, owner, connector, draft state, timestamps and etag 1")
	}
	return nil
}
func (hooks *ReportInsertRules) Validate(ctx context.Context, entity *StoredReport, state xhandler.LifecycleContext[StoredReport, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ReportInsertRules) AfterSequence(ctx context.Context, entity *StoredReport, state xhandler.LifecycleContext[StoredReport, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ReportInsertRules) AfterQueue(ctx context.Context, entity *StoredReport, state xhandler.LifecycleContext[StoredReport, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ReportInsertRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
