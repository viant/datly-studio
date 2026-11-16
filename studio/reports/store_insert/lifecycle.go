package store_insert

import (
	context "context"
	"fmt"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
)

// ReportInsertRules customizes role Input.Reports.
type ReportInsertRules struct {
	Input *Input `bind:"kind=input,required"`
}

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
	if hooks.Input == nil {
		return fmt.Errorf("component creation requires receipt admission")
	}
	covered := false
	for _, id := range hooks.Input.ComponentIds {
		if id == entity.Id {
			covered = true
			break
		}
	}
	if !covered {
		return fmt.Errorf("component creation requires receipt admission for its identity")
	}
	for _, receipt := range hooks.Input.Deletions {
		if receipt != nil && receipt.ComponentId == entity.Id {
			return &xhandler.Conflict{Entity: "component", Field: "id", Reason: "component identity is permanently reserved"}
		}
	}
	if entity.NamespaceId != namespaceaccess.ID(entity.OwnerId, entity.Namespace) {
		return fmt.Errorf("component namespace ownership must match its owner and namespace")
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
