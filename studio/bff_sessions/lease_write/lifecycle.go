package lease_write

import (
	context "context"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
)

// LeaseRules customizes role Input.Sessions.
type LeaseRules struct{}

func LeaseRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*LeaseRules)(nil)).Elem()
}

var (
	LeaseRulesHooks = new(LeaseRules)
	LeaseRulesDatly = LeaseRulesDatlyType()
)

func (hooks *LeaseRules) Init(ctx context.Context, entity *LeaseSession, state xhandler.LifecycleContext[LeaseSession, xhandler.NoParent, Output]) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if state.Previous == nil {
		return &xhandler.Conflict{Entity: "session", Reason: "refresh session no longer exists"}
	}
	return nil
}

func (input *Input) LeaseConditionScope() (string, string, int64) {
	return input.Operation, input.LeaseOwner, input.NowUnix
}
