package writer

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
)

// VersionRules authorizes both existing and newly inserted report versions.
// The predicate protects source reads; this hook protects a new child row that
// has no previous record to constrain.
type VersionRules struct {
	Input      *Input             `bind:"kind=input,required"`
	Connectors connector.Provider `bind:"kind=connector,required"`
}

// VersionRulesHooks keeps the local lifecycle type reachable for Datly package
// scanning without registration or init side effects.
var VersionRulesHooks = new(VersionRules)

func VersionRulesDatlyType() reflect.Type { return reflect.TypeOf((*VersionRules)(nil)).Elem() }

var VersionRulesDatlyLinkedType = VersionRulesDatlyType()

func (r *VersionRules) Init(context.Context, *ReportVersion, xhandler.LifecycleContext[ReportVersion, xhandler.NoParent, Output]) error {
	return nil
}

func (r *VersionRules) Validate(ctx context.Context, version *ReportVersion, _ xhandler.LifecycleContext[ReportVersion, xhandler.NoParent, Output]) error {
	if version == nil || version.ReportId == nil {
		return fmt.Errorf("report version report_id is required")
	}
	if r == nil || r.Input == nil {
		return fmt.Errorf("report version authorization input is unavailable")
	}
	return authorization.AuthorizeReport(ctx, r.Connectors, r.Input.Jwt, *version.ReportId, "can_edit")
}
