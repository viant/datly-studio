package preview

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	active "github.com/viant/datly-studio/studio/connectors/store_active"
	scoped "github.com/viant/datly-studio/studio/connectors/store_preview_scoped"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

// ConnectorReader hosts the public-mode and subject-scoped preview connector
// readers in-process. Its routes never reach the Studio HTTP/MCP gateways.
type ConnectorReader struct {
	runtime *druntime.Runtime
	all     dexec.ComponentTarget
	scoped  dexec.ComponentTarget
}

func NewConnectorReader(db *sql.DB) (*ConnectorReader, error) {
	resources := resource.New()
	if err := resources.Register(active.ConnectorDatlyResourceNamespace, active.ConnectorDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(scoped.ConnectorDatlyResourceNamespace, scoped.ConnectorDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	allRegistration, allTarget, err := readercomponent.Compile(reflect.TypeOf(active.ConnectorComponent{}), "store_active",
		reflect.TypeOf(active.Input{}), reflect.TypeOf(active.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	scopedRegistration, scopedTarget, err := readercomponent.Compile(reflect.TypeOf(scoped.ConnectorComponent{}), "store_preview_scoped",
		reflect.TypeOf(scoped.Input{}), reflect.TypeOf(scoped.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{allRegistration, scopedRegistration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	return &ConnectorReader{runtime: runtime, all: allTarget, scoped: scopedTarget}, nil
}

func (r *ConnectorReader) ListAll(ctx context.Context) ([]connectorDefinition, error) {
	if r == nil || r.runtime == nil {
		return nil, fmt.Errorf("preview connector reader is unavailable")
	}
	value, err := r.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: r.all, Input: &active.Input{}})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*active.Output)
	if !ok {
		return nil, fmt.Errorf("active connector reader returned %T", value)
	}
	result := make([]connectorDefinition, 0, len(output.Connectors))
	for _, row := range output.Connectors {
		if row == nil {
			return nil, fmt.Errorf("active connector reader returned nil row")
		}
		item, err := previewConnector(row.Name, row.Driver, row.DsnTemplate, row.SecretRef, row.OptionsJson)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (r *ConnectorReader) ListForPrincipal(ctx context.Context, subject string) ([]connectorDefinition, error) {
	if r == nil || r.runtime == nil || subject == "" {
		return nil, fmt.Errorf("preview connector reader requires a verified subject")
	}
	input := &scoped.Input{Subject: subject, Has: &scoped.InputHas{Subject: true}}
	value, err := r.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: r.scoped, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*scoped.Output)
	if !ok {
		return nil, fmt.Errorf("scoped connector reader returned %T", value)
	}
	result := make([]connectorDefinition, 0, len(output.Connectors))
	for _, row := range output.Connectors {
		if row == nil {
			return nil, fmt.Errorf("scoped connector reader returned nil row")
		}
		item, err := previewConnector(row.Name, row.Driver, row.DsnTemplate, row.SecretRef, row.OptionsJson)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func previewConnector(name, driver string, dsn *string, secretRef string, options json.RawMessage) (connectorDefinition, error) {
	if name == "" || driver == "" || dsn == nil {
		return connectorDefinition{}, fmt.Errorf("preview connector reader returned incomplete connector %q", name)
	}
	return connectorDefinition{Name: name, Driver: driver, DSN: *dsn, SecretRef: secretRef,
		Options: append(json.RawMessage(nil), options...)}, nil
}

func (r *ConnectorReader) Close(ctx context.Context) error {
	if r == nil || r.runtime == nil {
		return nil
	}
	return r.runtime.Shutdown(ctx)
}
