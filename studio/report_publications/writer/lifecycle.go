package writer

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
)

// PublicationRules authorizes new and existing report-scoped rows.
type PublicationRules struct {
	Input      *Input             `bind:"kind=input,required"`
	Connectors connector.Provider `bind:"kind=connector,required"`
}

var PublicationRulesHooks = new(PublicationRules)

func PublicationRulesDatlyType() reflect.Type { return reflect.TypeOf((*PublicationRules)(nil)).Elem() }

var PublicationRulesDatlyLinkedType = PublicationRulesDatlyType()

func (r *PublicationRules) Init(context.Context, *ReportPublication, xhandler.LifecycleContext[ReportPublication, xhandler.NoParent, Output]) error {
	return nil
}

func (r *PublicationRules) Validate(ctx context.Context, value *ReportPublication, _ xhandler.LifecycleContext[ReportPublication, xhandler.NoParent, Output]) error {
	if value == nil || value.ReportId == nil {
		return fmt.Errorf("report identity is required")
	}
	if r == nil || r.Input == nil {
		return fmt.Errorf("authorization input is unavailable")
	}
	return authorization.AuthorizeReport(ctx, r.Connectors, r.Input.Jwt, *value.ReportId, "can_publish")
}
