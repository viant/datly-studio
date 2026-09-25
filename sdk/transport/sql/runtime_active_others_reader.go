package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"github.com/viant/datly-studio/studio/runtime_generations/otheractivepredicate"
	stored "github.com/viant/datly-studio/studio/runtime_generations/store_active_others"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

func (t *Transport) readOtherActiveGenerations(ctx context.Context, tx *sql.Tx, exclude int64) ([]*stored.ActiveGeneration, error) {
	resources := resource.New()
	if err := resources.Register(stored.GenerationDatlyResourceNamespace, stored.GenerationDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB, Tx: tx}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	catalog, err := predicatecatalog.New(predicatecatalog.Package{Path: "github.com/viant/datly-studio/studio/runtime_generations/otheractivepredicate",
		Types: []reflect.Type{reflect.TypeFor[otheractivepredicate.OtherActive]()}})
	if err != nil {
		return nil, err
	}
	types, err := catalog.RuntimeTypes()
	if err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.GenerationComponent{}), "store_active_others",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector, types)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	input := &stored.Input{ExcludeGeneration: exclude, Has: &stored.InputHas{ExcludeGeneration: true}}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("other active generations reader returned %T", value)
	}
	for _, row := range output.Generations {
		if row == nil || row.GenerationNo <= 0 || row.GenerationNo == exclude || row.Status != "active" {
			return nil, fmt.Errorf("other active generations reader returned a mismatched row")
		}
	}
	return output.Generations, nil
}
