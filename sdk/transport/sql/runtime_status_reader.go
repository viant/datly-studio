package sqltransport

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/runtime_generations/store_status"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

func (t *Transport) readActiveGeneration(ctx context.Context) (*stored.StoredGeneration, error) {
	resources := resource.New()
	if err := resources.Register(stored.GenerationDatlyResourceNamespace, stored.GenerationDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.GenerationComponent{}), "store_status",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target,
		Input: &stored.Input{Status: "active", Has: &stored.InputHas{Status: true}}})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("runtime status reader returned %T", value)
	}
	if len(output.Generations) == 0 {
		return nil, nil
	}
	if len(output.Generations) != 1 || output.Generations[0] == nil || output.Generations[0].Status != "active" {
		return nil, fmt.Errorf("runtime status reader returned an invalid active generation")
	}
	return output.Generations[0], nil
}
