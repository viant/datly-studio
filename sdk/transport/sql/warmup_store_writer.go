package sqltransport

import (
	"context"
	"fmt"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/report_warmup_runs/store_write"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	"reflect"
)

// writeWarmupRun invokes the transcribed server-only Datly v1 writer. The
// generated lifecycle hook owns transitions; the writer owns Previous and CAS.
func (t *Transport) writeWarmupRun(ctx context.Context, row *stored.StoredWarmupRun) error {
	resources := resource.New()
	if err := resources.Register(stored.WarmupRunDatlyResourceNamespace, stored.WarmupRunDatlyResources); err != nil {
		return err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return err
	}
	registration, target, err := readercomponent.CompileWriter(reflect.TypeOf(stored.WarmupRunComponent{}), "store_write", reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return err
	}
	defer runtime.Shutdown(context.Background())
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target,
		Input: &stored.Input{Runs: []*stored.StoredWarmupRun{row}, Has: &stored.InputHas{Runs: true}}})
	if err != nil {
		return err
	}
	if _, ok := value.(*stored.Output); !ok {
		return fmt.Errorf("warmup store writer returned %T", value)
	}
	return nil
}
