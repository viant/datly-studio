package sqltransport

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/report_versions/store_head"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

// nextVersionNo allocates the next version number through the generated
// server-only head reader. The composite primary key on report_versions
// remains the authority: a concurrent allocation of the same number fails the
// insert and rolls the caller-owned import transaction back.
func (t *Transport) nextVersionNo(ctx context.Context, reportID string) (int, error) {
	resources := resource.New()
	if err := resources.Register(stored.HeadDatlyResourceNamespace, stored.HeadDatlyResources); err != nil {
		return 0, err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
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
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target,
		Input: &stored.Input{ReportId: reportID, Has: &stored.InputHas{ReportId: true}}})
	if err != nil {
		return 0, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return 0, fmt.Errorf("version head reader returned %T", value)
	}
	if len(output.Heads) != 1 || output.Heads[0] == nil || output.Heads[0].MaxVersionNo < 0 {
		return 0, fmt.Errorf("version head reader returned an invalid head for report %s", reportID)
	}
	return output.Heads[0].MaxVersionNo + 1, nil
}
