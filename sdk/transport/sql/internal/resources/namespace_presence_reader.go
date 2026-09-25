package resources

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/resource_namespaces/store_presence"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

func NamespaceHasResources(ctx context.Context, db *sql.DB, tx *sql.Tx, reportID, namespace string) (bool, error) {
	resources := resource.New()
	if err := resources.Register(stored.UsageDatlyResourceNamespace, stored.UsageDatlyResources); err != nil {
		return false, err
	}
	connector := &dsql.SQLComponent{DB: db, Tx: tx}
	if err := connector.RegisterConnector("studio", db); err != nil {
		return false, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.UsageComponent{}), "store_presence",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return false, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return false, err
	}
	defer runtime.Shutdown(context.Background())
	input := &stored.Input{ReportId: reportID, Namespace: namespace,
		Has: &stored.InputHas{ReportId: true, Namespace: true}}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return false, err
	}
	output, ok := value.(*stored.Output)
	if !ok || output == nil || len(output.Usages) > 1 {
		return false, fmt.Errorf("resource namespace presence returned invalid output %T", value)
	}
	for _, row := range output.Usages {
		if row == nil || row.ReportId != reportID || row.Namespace != namespace {
			return false, fmt.Errorf("resource namespace presence returned mismatched row")
		}
	}
	return len(output.Usages) == 1, nil
}
