package store_insert

import (
	"context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	"math"
	"reflect"
)

// AllocationRules advances the component high-water in the version mutation graph.
type AllocationRules struct{}

func AllocationRulesDatlyType() reflect.Type { return reflect.TypeFor[AllocationRules]() }

var AllocationRulesDatly = AllocationRulesDatlyType()

func (*AllocationRules) Init(ctx context.Context, entity *VersionAllocation, state xhandler.LifecycleContext[VersionAllocation, xhandler.NoParent, Output]) error {
	if entity == nil || state.Previous == nil || entity.LastVersionNo == nil || *entity.LastVersionNo < 0 || *entity.LastVersionNo >= math.MaxInt32 {
		return fmt.Errorf("version allocation requires an existing component and valid high-water")
	}
	if entity.NamespaceId != state.Previous.NamespaceId || len(entity.Draft) != 1 || entity.Draft[0] == nil || entity.Draft[0].VersionNo != *entity.LastVersionNo+1 {
		return fmt.Errorf("version allocation requires one next component version")
	}
	entity.SetLastVersionNo(versionNumber(*entity.LastVersionNo + 1))
	return nil
}
func (*AllocationRules) Validate(ctx context.Context, entity *VersionAllocation, state xhandler.LifecycleContext[VersionAllocation, xhandler.NoParent, Output]) error {
	return nil
}

func versionNumber(value int) *int { return &value }
