package writer

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
)

// ACLRules authorizes new and existing report-scoped rows.
type ACLRules struct {
	Input      *Input             `bind:"kind=input,required"`
	Connectors connector.Provider `bind:"kind=connector,required"`
}

var ACLRulesHooks = new(ACLRules)

func ACLRulesDatlyType() reflect.Type { return reflect.TypeOf((*ACLRules)(nil)).Elem() }

var ACLRulesDatlyLinkedType = ACLRulesDatlyType()

func (r *ACLRules) Init(context.Context, *ReportACL, xhandler.LifecycleContext[ReportACL, xhandler.NoParent, Output]) error {
	return nil
}

func (r *ACLRules) Validate(ctx context.Context, value *ReportACL, _ xhandler.LifecycleContext[ReportACL, xhandler.NoParent, Output]) error {
	if value == nil || value.ReportId == nil {
		return fmt.Errorf("report identity is required")
	}
	if value.SubjectType == nil || strings.TrimSpace(*value.SubjectType) != "user" {
		return fmt.Errorf("ACL subject type must be user")
	}
	if value.SubjectId == nil || strings.TrimSpace(*value.SubjectId) == "" {
		return fmt.Errorf("ACL subject ID must be a verified JWT sub")
	}
	if r == nil || r.Input == nil {
		return fmt.Errorf("authorization input is unavailable")
	}
	return authorization.AuthorizeReport(ctx, r.Connectors, r.Input.Jwt, *value.ReportId, "can_publish")
}
