// Package namespaceaccess invokes the server-only transcribed namespace
// authorization reader used by the SDK authorizer.
package namespaceaccess

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/studio/namespaces/accesspredicate"
	stored "github.com/viant/datly-studio/studio/namespaces/store_access"
	"github.com/viant/datly-studio/studio/predicatecatalog"
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
		return nil, fmt.Errorf("namespace access database is unavailable")
	}
	resources := resource.New()
	if err := resources.Register(stored.NamespaceDatlyResourceNamespace, stored.NamespaceDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	catalog, err := predicatecatalog.New(predicatecatalog.Package{Path: "github.com/viant/datly-studio/studio/namespaces/accesspredicate",
		Types: []reflect.Type{reflect.TypeFor[accesspredicate.NamespaceAccess]()}})
	if err != nil {
		return nil, err
	}
	types, err := catalog.RuntimeTypes()
	if err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.NamespaceComponent{}), "store_access",
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

func (reader *Reader) Allowed(ctx context.Context, subject, name, permission string) (bool, error) {
	if reader == nil || reader.runtime == nil {
		return false, fmt.Errorf("namespace access reader is unavailable")
	}
	input := &stored.Input{Name: name, Subject: subject, Permission: permission,
		Has: &stored.InputHas{Name: true, Subject: true, Permission: true}}
	value, err := reader.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: reader.target, Input: input})
	if err != nil {
		return false, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return false, fmt.Errorf("namespace access reader returned %T", value)
	}
	if len(output.Namespaces) == 0 {
		return false, nil
	}
	if len(output.Namespaces) != 1 || output.Namespaces[0] == nil || output.Namespaces[0].Name != name {
		return false, fmt.Errorf("namespace access reader returned an ambiguous or mismatched namespace")
	}
	return true, nil
}

func (reader *Reader) Close(ctx context.Context) error {
	if reader == nil || reader.runtime == nil {
		return nil
	}
	return reader.runtime.Shutdown(ctx)
}
