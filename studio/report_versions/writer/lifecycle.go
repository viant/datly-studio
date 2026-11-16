package writer

import (
	"context"
	"fmt"
	xresponse "github.com/viant/xdatly/response"
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
	counters   map[string]bool
}

// VersionRulesHooks keeps the local lifecycle type reachable for Datly package
// scanning without registration or init side effects.
var VersionRulesHooks = new(VersionRules)

func VersionRulesDatlyType() reflect.Type { return reflect.TypeOf((*VersionRules)(nil)).Elem() }

var VersionRulesDatlyLinkedType = VersionRulesDatlyType()

func (r *VersionRules) Init(ctx context.Context, version *ReportVersion, state xhandler.LifecycleContext[ReportVersion, xhandler.NoParent, Output]) error {
	if version == nil || version.ReportId == nil || r.Input == nil {
		return nil
	}
	// Only the first row for each report owns the counter mutation. The native
	// writer rebuilds frames after Init, including this server-owned relation.
	if r.counters == nil {
		r.counters = map[string]bool{}
	}
	for _, component := range r.Input.CurrentAllocation {
		if component.Id != *version.ReportId {
			continue
		}
		if version.NamespaceId == nil {
			id := component.NamespaceId
			version.SetNamespaceId(&id)
		}
		if !r.counters[*version.ReportId] {
			id, previous := component.Id, component.LastVersionNo
			version.SetAllocation([]*VersionAllocation{{Id: &id, LastVersionNo: &previous, Has: &VersionAllocationHas{Id: true, LastVersionNo: true}}})
			r.counters[*version.ReportId] = true
		}
		break
	}
	return nil
}

func (r *VersionRules) Validate(ctx context.Context, version *ReportVersion, _ xhandler.LifecycleContext[ReportVersion, xhandler.NoParent, Output]) error {
	if version == nil || version.ReportId == nil {
		return fmt.Errorf("report version report_id is required")
	}
	if r == nil || r.Input == nil {
		return fmt.Errorf("report version authorization input is unavailable")
	}
	if err := authorization.AuthorizeReport(ctx, r.Connectors, r.Input.Jwt, *version.ReportId, "can_edit"); err != nil {
		return err
	}
	if version.VersionNo != nil && !existingVersion(r.Input, *version.ReportId, *version.VersionNo) {
		for _, component := range r.Input.CurrentAllocation {
			if component.Id == *version.ReportId && *version.VersionNo <= component.LastVersionNo {
				return &xresponse.Error{Code: 409, Cause: fmt.Errorf("version identity has already been allocated")}
			}
		}
	}
	return nil
}
