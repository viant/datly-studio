package writer

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
)

// ParameterRules authorizes new and existing report-scoped rows.
type ParameterRules struct {
	Input      *Input             `bind:"kind=input,required"`
	Connectors connector.Provider `bind:"kind=connector,required"`
}

var ParameterRulesHooks = new(ParameterRules)

func ParameterRulesDatlyType() reflect.Type { return reflect.TypeOf((*ParameterRules)(nil)).Elem() }

var ParameterRulesDatlyLinkedType = ParameterRulesDatlyType()

func (r *ParameterRules) Init(context.Context, *ReportParameter, xhandler.LifecycleContext[ReportParameter, xhandler.NoParent, Output]) error {
	return nil
}

func (r *ParameterRules) Validate(ctx context.Context, value *ReportParameter, _ xhandler.LifecycleContext[ReportParameter, xhandler.NoParent, Output]) error {
	if value == nil || value.ReportId == nil {
		return fmt.Errorf("report identity is required")
	}
	if r == nil || r.Input == nil {
		return fmt.Errorf("authorization input is unavailable")
	}
	return authorization.AuthorizeReport(ctx, r.Connectors, r.Input.Jwt, *value.ReportId, "can_edit")
}
