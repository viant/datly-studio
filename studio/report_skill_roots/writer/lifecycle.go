package writer

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
)

// SkillRules authorizes new and existing report-scoped rows.
type SkillRules struct {
	Input      *Input             `bind:"kind=input,required"`
	Connectors connector.Provider `bind:"kind=connector,required"`
}

var SkillRulesHooks = new(SkillRules)

func SkillRulesDatlyType() reflect.Type { return reflect.TypeOf((*SkillRules)(nil)).Elem() }

var SkillRulesDatlyLinkedType = SkillRulesDatlyType()

func (r *SkillRules) Init(context.Context, *ReportSkillRoot, xhandler.LifecycleContext[ReportSkillRoot, xhandler.NoParent, Output]) error {
	return nil
}

func (r *SkillRules) Validate(ctx context.Context, value *ReportSkillRoot, _ xhandler.LifecycleContext[ReportSkillRoot, xhandler.NoParent, Output]) error {
	if value == nil || value.ReportId == nil {
		return fmt.Errorf("report identity is required")
	}
	if r == nil || r.Input == nil {
		return fmt.Errorf("authorization input is unavailable")
	}
	return authorization.AuthorizeReport(ctx, r.Connectors, r.Input.Jwt, *value.ReportId, "can_edit")
}
