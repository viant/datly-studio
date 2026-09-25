package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/resource_namespaces/store_usage"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

func (t *Transport) readForeignResourceNamespaceUsage(ctx context.Context, tx *sql.Tx, namespace, excludeReportID string) ([]*stored.NamespaceUsage, error) {
	resources := resource.New()
	if err := resources.Register(stored.UsageDatlyResourceNamespace, stored.UsageDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB, Tx: tx}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.UsageComponent{}), "store_usage",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	input := &stored.Input{Namespace: namespace, ExcludeReportId: excludeReportID,
		Has: &stored.InputHas{Namespace: true, ExcludeReportId: true}}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("resource namespace usage returned %T", value)
	}
	if len(output.Usages) > 2 {
		return nil, fmt.Errorf("resource namespace usage exceeded requested limit")
	}
	for _, row := range output.Usages {
		if row == nil || row.Namespace != namespace || row.ReportId == "" || row.ReportId == excludeReportID {
			return nil, fmt.Errorf("resource namespace usage returned a mismatched row")
		}
	}
	return output.Usages, nil
}
