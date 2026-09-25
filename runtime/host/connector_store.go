package host

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/viant/bindly/resource"
	stored "github.com/viant/datly-studio/studio/connectors/store_active"
	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	dtag "github.com/viant/datly/tag"
)

type connectorDefinition struct {
	name, driver, dsn, secretRef string
	options                      json.RawMessage
}

type activeConnectorStore struct {
	db      *sql.DB
	mu      sync.Mutex
	runtime *druntime.Runtime
	target  dexec.ComponentTarget
}

func (s *activeConnectorStore) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runtime == nil {
		return nil
	}
	err := s.runtime.Shutdown(ctx)
	s.runtime = nil
	return err
}

// Read invokes a fresh query through the unexposed Datly v1 component while
// reusing its compiled runtime. The route is internal dispatch metadata only.
func (s *activeConnectorStore) Read(ctx context.Context) (map[string]connectorDefinition, error) {
	runtime, target, err := s.components()
	if err != nil {
		return nil, err
	}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: &stored.Input{}})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("active connector store returned %T", value)
	}
	result := make(map[string]connectorDefinition, len(output.Connectors))
	for _, row := range output.Connectors {
		if row == nil || row.Name == "" || row.Driver == "" {
			return nil, fmt.Errorf("active connector store returned an incomplete row")
		}
		if _, exists := result[row.Name]; exists {
			return nil, fmt.Errorf("active connector store returned duplicate connector %q", row.Name)
		}
		item := connectorDefinition{name: row.Name, driver: row.Driver, secretRef: row.SecretRef, options: row.OptionsJson}
		if row.DsnTemplate != nil {
			item.dsn = *row.DsnTemplate
		}
		result[row.Name] = item
	}
	return result, nil
}

func (s *activeConnectorStore) components() (*druntime.Runtime, dexec.ComponentTarget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runtime != nil {
		return s.runtime, s.target, nil
	}
	resources := resource.New()
	if err := resources.Register(stored.ConnectorDatlyResourceNamespace, stored.ConnectorDatlyResources); err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	connector := &dsql.SQLComponent{DB: s.db}
	if err := connector.RegisterConnector("studio", s.db); err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	holder := reflect.TypeOf(stored.ConnectorComponent{})
	field, ok := holder.FieldByName("Contract")
	if !ok {
		return nil, dexec.ComponentTarget{}, fmt.Errorf("active connector store has no generated component contract")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil {
		return nil, dexec.ComponentTarget{}, fmt.Errorf("active connector store metadata: %w", err)
	}
	if !present {
		return nil, dexec.ComponentTarget{}, fmt.Errorf("active connector store has no component metadata")
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "store_active", PackagePath: holder.PkgPath(), Tag: metadata,
		InputType: "Input", OutputType: "Output"}).Resolve(reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}))
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeOf(stored.Input{}), OutputType: reflect.TypeOf(stored.Output{}), Resources: resources})
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	execution, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: connector})
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	registration, err := artifact.Registration(registry.RegisteredComponent{Reader: execution})
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	target := dexec.ComponentTarget{Component: component.Key}
	if len(component.Routes) > 0 && component.Routes[0] != nil {
		target.Route = spec.RouteRef{Method: component.Routes[0].Method, Path: component.Routes[0].Path}
	}
	s.runtime, s.target = runtime, target
	return runtime, target, nil
}
