package preview

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	stored "github.com/viant/datly-studio/studio/report_versions/store_preview_definition"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
)

// DefinitionReader resolves exact draft/version metadata via a server-only
// Datly v1 reader. A Studio API host may reuse one instance for all previews.
type DefinitionReader struct {
	runtime *druntime.Runtime
	target  dexec.ComponentTarget
}

func NewDefinitionReader(db *sql.DB) (*DefinitionReader, error) {
	resources := resource.New()
	if err := resources.Register(stored.DefinitionDatlyResourceNamespace, stored.DefinitionDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(stored.DefinitionComponent{}), "store_preview_definition",
		reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	return &DefinitionReader{runtime: runtime, target: target}, nil
}

func (r *DefinitionReader) Get(ctx context.Context, reportID string, versionNo int) (*stored.PreviewDefinition, error) {
	if r == nil || r.runtime == nil || reportID == "" || versionNo <= 0 {
		return nil, fmt.Errorf("preview definition reader requires an exact report version")
	}
	input := &stored.Input{ReportId: reportID, VersionNo: versionNo,
		Has: &stored.InputHas{ReportId: true, VersionNo: true}}
	value, err := r.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: r.target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*stored.Output)
	if !ok {
		return nil, fmt.Errorf("preview definition reader returned %T", value)
	}
	if len(output.Definitions) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(output.Definitions) != 1 || output.Definitions[0] == nil || output.Definitions[0].ReportId != reportID || output.Definitions[0].VersionNo != versionNo {
		return nil, fmt.Errorf("preview definition reader returned an ambiguous or mismatched version")
	}
	if output.Definitions[0].DsnTemplate == nil {
		return nil, fmt.Errorf("preview definition %s v%d has no connector DSN", reportID, versionNo)
	}
	return output.Definitions[0], nil
}

func (r *DefinitionReader) Close(ctx context.Context) error {
	if r == nil || r.runtime == nil {
		return nil
	}
	return r.runtime.Shutdown(ctx)
}
