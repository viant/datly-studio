package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/runtime_generations/store_head"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

// nextGenerationNo reads through Datly on the caller's transaction. The
// generation primary key remains authoritative if another host races it.
func (t *Transport) nextGenerationNo(ctx context.Context, tx *sql.Tx) (int64, error) {
	resources := resource.New()
	if err := resources.Register(stored.HeadDatlyResourceNamespace, stored.HeadDatlyResources); err != nil {
		return 0, err
	}
	connector := &dsql.SQLComponent{DB: t.DB, Tx: tx}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return 0, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.HeadComponent{}), "store_head",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return 0, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return 0, err
	}
	defer runtime.Shutdown(context.Background())
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: &stored.Input{}})
	if err != nil {
		return 0, err
	}
	output, ok := value.(*stored.Output)
	if !ok || len(output.Heads) != 1 || output.Heads[0] == nil || output.Heads[0].MaxGeneration < 0 {
		return 0, fmt.Errorf("runtime generation head returned invalid result %T", value)
	}
	return output.Heads[0].MaxGeneration + 1, nil
}
