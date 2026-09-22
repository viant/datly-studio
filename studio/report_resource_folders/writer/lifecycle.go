package writer

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
)

// FolderRules authorizes new and existing report-scoped rows.
type FolderRules struct {
	Input      *Input             `bind:"kind=input,required"`
	Connectors connector.Provider `bind:"kind=connector,required"`
}

var FolderRulesHooks = new(FolderRules)

func FolderRulesDatlyType() reflect.Type { return reflect.TypeOf((*FolderRules)(nil)).Elem() }

var FolderRulesDatlyLinkedType = FolderRulesDatlyType()

func (r *FolderRules) Init(context.Context, *ReportResourceFolder, xhandler.LifecycleContext[ReportResourceFolder, xhandler.NoParent, Output]) error {
	return nil
}

func (r *FolderRules) Validate(ctx context.Context, value *ReportResourceFolder, _ xhandler.LifecycleContext[ReportResourceFolder, xhandler.NoParent, Output]) error {
	if value == nil || value.ReportId == nil {
		return fmt.Errorf("report identity is required")
	}
	if r == nil || r.Input == nil {
		return fmt.Errorf("authorization input is unavailable")
	}
	return authorization.AuthorizeReport(ctx, r.Connectors, r.Input.Jwt, *value.ReportId, "can_edit")
}
