package writer

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
)

// ExposureRules authorizes new and existing report-scoped rows.
type ExposureRules struct {
	Input      *Input             `bind:"kind=input,required"`
	Connectors connector.Provider `bind:"kind=connector,required"`
}

var ExposureRulesHooks = new(ExposureRules)

func ExposureRulesDatlyType() reflect.Type { return reflect.TypeOf((*ExposureRules)(nil)).Elem() }

var ExposureRulesDatlyLinkedType = ExposureRulesDatlyType()

func (r *ExposureRules) Init(context.Context, *ReportMCPExposure, xhandler.LifecycleContext[ReportMCPExposure, xhandler.NoParent, Output]) error {
	return nil
}

func (r *ExposureRules) Validate(ctx context.Context, value *ReportMCPExposure, _ xhandler.LifecycleContext[ReportMCPExposure, xhandler.NoParent, Output]) error {
	if value == nil || value.ReportId == nil {
		return fmt.Errorf("report identity is required")
	}
	if r == nil || r.Input == nil {
		return fmt.Errorf("authorization input is unavailable")
	}
	return authorization.AuthorizeReport(ctx, r.Connectors, r.Input.Jwt, *value.ReportId, "can_edit")
}
