// Package globalaccess invokes the server-only transcribed global-publish
// authorization reader used by the SDK authorizer.
package globalaccess

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"github.com/viant/datly-studio/studio/reports/globalpredicate"
	stored "github.com/viant/datly-studio/studio/reports/store_global_access"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

type Reader struct {
	runtime *druntime.Runtime
	target  dexec.ComponentTarget
}

func New(db *sql.DB) (*Reader, error) {
	if db == nil {
		return nil, fmt.Errorf("global access database is unavailable")
	}
	resources := resource.New()
	if err := resources.Register(stored.ReportDatlyResourceNamespace, stored.ReportDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	catalog, err := predicatecatalog.New(predicatecatalog.Package{Path: "github.com/viant/datly-studio/studio/reports/globalpredicate",
		Types: []reflect.Type{reflect.TypeFor[globalpredicate.GlobalPublish]()}})
	if err != nil {
		return nil, err
	}
	types, err := catalog.RuntimeTypes()
	if err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.ReportComponent{}), "store_global_access",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector, types)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	return &Reader{runtime: runtime, target: target}, nil
}

func (reader *Reader) Allowed(ctx context.Context, subject string) (bool, error) {
	if reader == nil || reader.runtime == nil {
		return false, fmt.Errorf("global access reader is unavailable")
	}
	input := &stored.Input{Subject: subject, Has: &stored.InputHas{Subject: true}}
	value, err := reader.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: reader.target, Input: input})
	if err != nil {
		return false, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return false, fmt.Errorf("global access reader returned %T", value)
	}
	if len(output.Reports) == 0 {
		return false, nil
	}
	if len(output.Reports) != 1 || output.Reports[0] == nil || output.Reports[0].Id == "" {
		return false, fmt.Errorf("global access reader returned an invalid report")
	}
	return true, nil
}

func (reader *Reader) Close(ctx context.Context) error {
	if reader == nil || reader.runtime == nil {
		return nil
	}
	return reader.runtime.Shutdown(ctx)
}
