package sqltransport

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/report_warmup_runs/store_expired"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

func (t *Transport) expiredWarmupRuns(ctx context.Context, before time.Time, limit int) (rows []*stored.ExpiredWarmupRun, err error) {
	resources := resource.New()
	if err := resources.Register(stored.WarmupRunDatlyResourceNamespace, stored.WarmupRunDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.WarmupRunComponent{}), "store_expired",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, runtime.Shutdown(context.Background())) }()
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target,
		Input: &stored.Input{Before: before, PageLimit: limit, Has: &stored.InputHas{Before: true, PageLimit: true}}})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok || len(output.WarmupRuns) > limit {
		return nil, fmt.Errorf("expired warmup reader returned %T with unexpected rows", value)
	}
	for _, row := range output.WarmupRuns {
		if row == nil || row.RunId == "" || row.UpdatedAt == nil || row.Status != "accepted" && row.Status != "running" {
			return nil, errors.New("expired warmup reader returned an invalid row")
		}
	}
	return output.WarmupRuns, nil
}
