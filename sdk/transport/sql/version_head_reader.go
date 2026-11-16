package sqltransport

import (
	"context"
	"database/sql"
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

// nextVersionNo reads the locked component high-water in the caller transaction.
func (t *Transport) nextVersionNo(ctx context.Context, tx *sql.Tx, reportID, namespaceID string) (int, error) {
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
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target,
		ReaderOptions: &dexec.ReaderOptions{ForUpdate: []string{dexec.RootView}}, Input: &stored.Input{ReportId: reportID, NamespaceId: namespaceID, Has: &stored.InputHas{ReportId: true, NamespaceId: true}}})
	if err != nil {
		return 0, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return 0, fmt.Errorf("version head reader returned %T", value)
	}
	if len(output.Heads) != 1 || output.Heads[0] == nil || output.Heads[0].LastVersionNo < 0 {
		return 0, fmt.Errorf("version head reader returned an invalid head for report %s", reportID)
	}
	return output.Heads[0].LastVersionNo + 1, nil
}
