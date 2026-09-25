package sqltransport

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/connectors/store_usage"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

func (t *Transport) connectorUsage(ctx context.Context, name string) (int64, error) {
	resources := resource.New()
	if err := resources.Register(stored.UsageDatlyResourceNamespace, stored.UsageDatlyResources); err != nil {
		return 0, err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return 0, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.UsageComponent{}), "store_usage",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return 0, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return 0, err
	}
	defer runtime.Shutdown(context.Background())
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target,
		Input: &stored.Input{Name: name, Has: &stored.InputHas{Name: true}}})
	if err != nil {
		return 0, err
	}
	output, ok := value.(*stored.Output)
	if !ok || len(output.Usages) != 1 || output.Usages[0] == nil || output.Usages[0].Used < 0 {
		return 0, fmt.Errorf("connector usage reader returned %T with unexpected rows", value)
	}
	return output.Usages[0].Used, nil
}
