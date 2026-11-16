package writer

import (
	"context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	"math"
	"reflect"
)

// CounterRules updates one component per incoming report, in the same native
// mutation graph as the original caller-numbered version rows.
type CounterRules struct {
	Input *Input `bind:"kind=input,required"`
}

func CounterRulesDatlyType() reflect.Type { return reflect.TypeFor[CounterRules]() }

var CounterRulesDatly = CounterRulesDatlyType()

func (r *CounterRules) Init(ctx context.Context, entity *VersionAllocation, state xhandler.LifecycleContext[VersionAllocation, ReportVersion, Output]) error {
	if entity == nil || entity.Id == nil || state.Previous == nil || entity.LastVersionNo == nil {
		return fmt.Errorf("version counter requires existing component")
	}
	next := *entity.LastVersionNo
	for _, version := range r.Input.Versions {
		if version != nil && version.ReportId != nil && *version.ReportId == *entity.Id && version.VersionNo != nil && !existingVersion(r.Input, *version.ReportId, *version.VersionNo) && *version.VersionNo > next {
			next = *version.VersionNo
		}
	}
	entity.SetLastVersionNo(&next)
	return nil
}
func (r *CounterRules) Validate(ctx context.Context, entity *VersionAllocation, state xhandler.LifecycleContext[VersionAllocation, ReportVersion, Output]) error {
	if entity.LastVersionNo == nil || state.Previous == nil || state.Previous.LastVersionNo == nil || *entity.LastVersionNo < *state.Previous.LastVersionNo || *entity.LastVersionNo > math.MaxInt32 {
		return fmt.Errorf("invalid version high-water")
	}
	return nil
}
func existingVersion(input *Input, reportID string, number int) bool {
	for _, version := range input.CurrentVersion {
		if version != nil && version.ReportId != nil && *version.ReportId == reportID && version.VersionNo != nil && *version.VersionNo == number {
			return true
		}
	}
	return false
}
