package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/runtime_generations/store_catalog"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

func (t *Transport) readGenerationCatalog(ctx context.Context, tx *sql.Tx, generationNo int64, status string) ([]*stored.CatalogGeneration, error) {
	resources := resource.New()
	if err := resources.Register(stored.GenerationDatlyResourceNamespace, stored.GenerationDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB, Tx: tx}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.GenerationComponent{}), "store_catalog",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	input := &stored.Input{GenerationNo: generationNo, Status: status,
		Has: &stored.InputHas{GenerationNo: generationNo > 0, Status: status != ""}}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("generation catalog returned %T", value)
	}
	for _, row := range output.Generations {
		if row == nil || row.GenerationNo <= 0 || generationNo > 0 && row.GenerationNo != generationNo || status != "" && row.Status != status {
			return nil, fmt.Errorf("generation catalog returned a mismatched row")
		}
	}
	return output.Generations, nil
}
