package writer

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
)

// CubeRules authorizes new and existing report-scoped rows.
type CubeRules struct {
	Input      *Input             `bind:"kind=input,required"`
	Connectors connector.Provider `bind:"kind=connector,required"`
}

var CubeRulesHooks = new(CubeRules)

func CubeRulesDatlyType() reflect.Type { return reflect.TypeOf((*CubeRules)(nil)).Elem() }

var CubeRulesDatlyLinkedType = CubeRulesDatlyType()

func (r *CubeRules) Init(context.Context, *ReportCubeConfig, xhandler.LifecycleContext[ReportCubeConfig, xhandler.NoParent, Output]) error {
	return nil
}

func (r *CubeRules) Validate(ctx context.Context, value *ReportCubeConfig, _ xhandler.LifecycleContext[ReportCubeConfig, xhandler.NoParent, Output]) error {
	if value == nil || value.ReportId == nil {
		return fmt.Errorf("report identity is required")
	}
	if r == nil || r.Input == nil {
		return fmt.Errorf("authorization input is unavailable")
	}
	return authorization.AuthorizeReport(ctx, r.Connectors, r.Input.Jwt, *value.ReportId, "can_edit")
}
