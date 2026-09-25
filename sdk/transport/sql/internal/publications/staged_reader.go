package publications

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/report_publications/store_staged"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

func ReadStaged(ctx context.Context, db *sql.DB, tx *sql.Tx, generation int64) ([]*stored.StagedPublication, error) {
	resources := resource.New()
	if err := resources.Register(stored.PublicationDatlyResourceNamespace, stored.PublicationDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: db, Tx: tx}
	if err := connector.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.PublicationComponent{}), "store_staged",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	input := &stored.Input{DesiredGeneration: generation, Has: &stored.InputHas{DesiredGeneration: true}}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("staged publication reader returned %T", value)
	}
	for _, row := range output.Publications {
		if row == nil || row.ReportId == "" || row.DesiredGeneration != generation {
			return nil, fmt.Errorf("staged publication reader returned a mismatched row")
		}
	}
	return output.Publications, nil
}
