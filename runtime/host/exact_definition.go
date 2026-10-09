package host

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	exact "github.com/viant/datly-studio/studio/runtime_generations/store_exact"
	"github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	"reflect"
)

func (s *Service) exactDefinition(ctx context.Context, id string, version int) (definition, error) {
	reader := s.metadataDefinitions
	if reader == nil {
		var err error
		reader, err = newExactDefinitionReader(s.studio)
		if err != nil {
			return definition{}, err
		}
		defer reader.runtime.Shutdown(context.Background())
	}
	return reader.read(ctx, id, version, s.config.NamespaceID)
}

type exactDefinitionReader struct {
	runtime *druntime.Runtime
	target  exec.ComponentTarget
}

func newExactDefinitionReader(db *sql.DB) (*exactDefinitionReader, error) {
	resources := resource.New()
	if err := resources.Register(exact.ResourceNamespace, exact.Resources); err != nil {
		return nil, err
	}
	sqlSource := &dsql.SQLComponent{DB: db}
	if err := sqlSource.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	registered, target, err := readercomponent.Compile(reflect.TypeFor[exact.Component](), "store_exact", reflect.TypeFor[exact.Input](), reflect.TypeFor[exact.Output](), resources, sqlSource)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registered}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	return &exactDefinitionReader{runtime: runtime, target: target}, nil
}
func (r *exactDefinitionReader) read(ctx context.Context, id string, version int, namespace string) (definition, error) {
	value, err := r.runtime.InvokeComponent(ctx, exec.ComponentRequest{Target: r.target, Input: &exact.Input{ReportID: id, VersionNo: version, NamespaceID: namespace, Has: &exact.InputHas{ReportID: true, VersionNo: true, NamespaceID: true}}})
	if err != nil {
		return definition{}, err
	}
	output, ok := value.(*exact.Output)
	if !ok || output == nil || len(output.Definitions) > 1 {
		return definition{}, fmt.Errorf("exact component reader returned invalid metadata")
	}
	if len(output.Definitions) == 0 {
		return definition{}, sql.ErrNoRows
	}
	row := output.Definitions[0]
	if row == nil || row.ReportID != id || row.VersionNo != version {
		return definition{}, fmt.Errorf("exact component identity mismatch")
	}
	return mapPublishedDefinition(row.ReportID, row.VersionNo, row.ComponentScope, row.ComponentName, row.DefaultConnectorName, row.Driver, row.DSNTemplate, row.SecretRef, row.GeneratedDQL, row.AuthoredDQL)
}
