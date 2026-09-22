package writer

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
)

// FileRules authorizes new and existing report-scoped rows.
type FileRules struct {
	Input      *Input             `bind:"kind=input,required"`
	Connectors connector.Provider `bind:"kind=connector,required"`
}

var FileRulesHooks = new(FileRules)

func FileRulesDatlyType() reflect.Type { return reflect.TypeOf((*FileRules)(nil)).Elem() }

var FileRulesDatlyLinkedType = FileRulesDatlyType()

func (r *FileRules) Init(context.Context, *ReportResourceFile, xhandler.LifecycleContext[ReportResourceFile, xhandler.NoParent, Output]) error {
	return nil
}

func (r *FileRules) Validate(ctx context.Context, value *ReportResourceFile, _ xhandler.LifecycleContext[ReportResourceFile, xhandler.NoParent, Output]) error {
	if value == nil || value.ReportId == nil {
		return fmt.Errorf("report identity is required")
	}
	if r == nil || r.Input == nil {
		return fmt.Errorf("authorization input is unavailable")
	}
	return authorization.AuthorizeReport(ctx, r.Connectors, r.Input.Jwt, *value.ReportId, "can_edit")
}
