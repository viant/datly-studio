package publications

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"github.com/viant/datly-studio/studio/report_publications/otheractivepredicate"
	stored "github.com/viant/datly-studio/studio/report_publications/store_active_others"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

func ReadOtherActive(ctx context.Context, db *sql.DB, tx *sql.Tx, excludeReportID string) ([]*stored.ActivePublication, error) {
	resources := resource.New()
	if err := resources.Register(stored.PublicationDatlyResourceNamespace, stored.PublicationDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: db, Tx: tx}
	if err := connector.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	catalog, err := predicatecatalog.New(predicatecatalog.Package{Path: "github.com/viant/datly-studio/studio/report_publications/otheractivepredicate",
		Types: []reflect.Type{reflect.TypeFor[otheractivepredicate.OtherActive]()}})
	if err != nil {
		return nil, err
	}
	types, err := catalog.RuntimeTypes()
	if err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.PublicationComponent{}), "store_active_others",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector, types)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	input := &stored.Input{ExcludeReportId: excludeReportID, Has: &stored.InputHas{ExcludeReportId: true}}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("other active publications reader returned %T", value)
	}
	for _, row := range output.Publications {
		if row == nil || row.ReportId == "" || row.ReportId == excludeReportID || row.PublicationStatus != "active" {
			return nil, fmt.Errorf("other active publications reader returned a mismatched row")
		}
	}
	return output.Publications, nil
}
