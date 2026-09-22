package writer

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
)

// GenerationRules restricts global runtime-generation changes to publishers.
type GenerationRules struct {
	Input      *Input             `bind:"kind=input,required"`
	Connectors connector.Provider `bind:"kind=connector,required"`
}

var GenerationRulesHooks = new(GenerationRules)

func GenerationRulesDatlyType() reflect.Type { return reflect.TypeOf((*GenerationRules)(nil)).Elem() }

var GenerationRulesDatlyLinkedType = GenerationRulesDatlyType()

func (r *GenerationRules) Init(context.Context, *RuntimeGeneration, xhandler.LifecycleContext[RuntimeGeneration, xhandler.NoParent, Output]) error {
	return nil
}

func (r *GenerationRules) Validate(ctx context.Context, _ *RuntimeGeneration, _ xhandler.LifecycleContext[RuntimeGeneration, xhandler.NoParent, Output]) error {
	if r == nil || r.Input == nil {
		return fmt.Errorf("authorization input is unavailable")
	}
	return authorization.AuthorizeGlobal(ctx, r.Connectors, r.Input.Jwt, "can_publish")
}
