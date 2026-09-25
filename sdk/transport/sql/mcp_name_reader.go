package sqltransport

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"github.com/viant/datly-studio/studio/report_versions/mcpnamepredicate"
	stored "github.com/viant/datly-studio/studio/report_versions/store_mcp_names"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

func (t *Transport) readMCPNameSources(ctx context.Context, reportID string) ([]*stored.NameSource, error) {
	resources := resource.New()
	if err := resources.Register(stored.VersionDatlyResourceNamespace, stored.VersionDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	catalog, err := predicatecatalog.New(predicatecatalog.Package{Path: "github.com/viant/datly-studio/studio/report_versions/mcpnamepredicate",
		Types: []reflect.Type{reflect.TypeFor[mcpnamepredicate.OtherReportCandidate]()}})
	if err != nil {
		return nil, err
	}
	types, err := catalog.RuntimeTypes()
	if err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.VersionComponent{}), "store_mcp_names",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector, types)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer runtime.Shutdown(context.Background())
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target,
		Input: &stored.Input{ReportId: reportID, Has: &stored.InputHas{ReportId: true}}})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("MCP name source reader returned %T", value)
	}
	for _, row := range output.Versions {
		if row == nil || row.ReportId == reportID || row.ReportId == "" || row.VersionNo <= 0 {
			return nil, fmt.Errorf("MCP name source reader returned a mismatched version")
		}
	}
	return output.Versions, nil
}
