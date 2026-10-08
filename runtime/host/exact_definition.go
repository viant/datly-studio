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
	resources := resource.New()
	if err := resources.Register(exact.ResourceNamespace, exact.Resources); err != nil {
		return definition{}, err
	}
	sqlSource := &dsql.SQLComponent{DB: s.studio}
	if err := sqlSource.RegisterConnector("studio", s.studio); err != nil {
		return definition{}, err
	}
	registered, target, err := readercomponent.Compile(reflect.TypeFor[exact.Component](), "store_exact", reflect.TypeFor[exact.Input](), reflect.TypeFor[exact.Output](), resources, sqlSource)
	if err != nil {
		return definition{}, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registered}, druntime.WithResources(resources))
	if err != nil {
		return definition{}, err
	}
	defer runtime.Shutdown(context.Background())
	value, err := runtime.InvokeComponent(ctx, exec.ComponentRequest{Target: target, Input: &exact.Input{ReportID: id, VersionNo: version, NamespaceID: s.config.NamespaceID, Has: &exact.InputHas{ReportID: true, VersionNo: true, NamespaceID: true}}})
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
