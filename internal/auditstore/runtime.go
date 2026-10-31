// Package auditstore binds private generated Datly v1 audit components.
package auditstore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	control "github.com/viant/datly-studio/studio/tool_call_audit/control_read"
	read "github.com/viant/datly-studio/studio/tool_call_audit/store_read"
	write "github.com/viant/datly-studio/studio/tool_call_audit/store_write"

	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"

	"github.com/viant/datly/runtime/registry"

	dsql "github.com/viant/datly/sql"

	"github.com/viant/datly-studio/internal/writercomponent"
	controlwrite "github.com/viant/datly-studio/studio/tool_call_audit/control_write"
	"github.com/viant/datly/typecatalog"
	_ "github.com/viant/sqlx/metadata/product/mysql"
	_ "github.com/viant/sqlx/metadata/product/sqlite"
	"github.com/viant/x"
	"reflect"
)

type Store struct {
	runtime                            *druntime.Runtime
	read, write, control, controlWrite dexec.ComponentTarget
}

func New(db *sql.DB) (*Store, error) {
	resources := resource.New()
	if err := resources.Register(read.AuditDatlyResourceNamespace, read.AuditDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(write.AuditDatlyResourceNamespace, write.AuditDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(control.ControlDatlyResourceNamespace, control.ControlDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(controlwrite.ControlDatlyResourceNamespace, controlwrite.ControlDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		return nil, err
	}
	reader, readTarget, err := readercomponent.Compile(reflect.TypeFor[read.AuditComponent](), "store_read", reflect.TypeFor[read.Input](), reflect.TypeFor[read.Output](), resources, connector)
	if err != nil {
		return nil, err
	}
	controlReader, controlTarget, err := readercomponent.Compile(reflect.TypeFor[control.ControlComponent](), "control_read", reflect.TypeFor[control.Input](), reflect.TypeFor[control.Output](), resources, connector)
	if err != nil {
		return nil, err
	}
	writer, writeTarget, err := writercomponent.Compile(reflect.TypeFor[write.AuditComponent](), reflect.TypeFor[write.Input](), reflect.TypeFor[write.Output](), resources, connector, db, nil)
	if err != nil {
		return nil, err
	}
	types := typecatalog.NewCatalog()
	if err = types.LinkRuntimeAll(typecatalog.TypeOriginPackage, x.NewType(reflect.TypeFor[controlwrite.ScopeRules]()), x.NewType(reflect.TypeFor[controlwrite.ComponentRules]()), x.NewType(reflect.TypeFor[controlwrite.ToolRules]())); err != nil {
		return nil, err
	}
	controls, controlWriteTarget, err := writercomponent.Compile(reflect.TypeFor[controlwrite.ControlComponent](), reflect.TypeFor[controlwrite.Input](), reflect.TypeFor[controlwrite.Output](), resources, connector, db, types)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{reader, writer, controlReader, controls}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	return &Store{runtime: runtime, read: readTarget, write: writeTarget, control: controlTarget, controlWrite: controlWriteTarget}, nil
}
func (s *Store) Read(ctx context.Context, input *read.Input) ([]*read.Event, error) {
	value, err := s.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: s.read, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*read.Output)
	if !ok {
		return nil, fmt.Errorf("audit reader returned %T", value)
	}
	return output.Events, nil
}
func (s *Store) Insert(ctx context.Context, row *write.ToolCallAudit) error {
	input := &write.Input{}
	input.SetAudit([]*write.ToolCallAudit{row})
	value, err := s.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: s.write, Input: input})
	if err != nil {
		return err
	}
	if _, ok := value.(*write.Output); !ok {
		return fmt.Errorf("audit writer returned %T", value)
	}
	if row.Id <= 0 {
		return fmt.Errorf("audit writer did not allocate an identity")
	}
	return nil
}
func (s *Store) Close(ctx context.Context) error {
	if s == nil || s.runtime == nil {
		return nil
	}
	return s.runtime.Shutdown(ctx)
}

func (s *Store) Control(ctx context.Context, namespaceID, componentID, toolName string) (*control.AuditScope, error) {
	input := &control.Input{}
	input.SetNamespaceId(namespaceID)
	input.SetComponentId(componentID)
	input.SetToolName(toolName)
	value, err := s.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: s.control, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*control.Output)
	if !ok {
		return nil, fmt.Errorf("audit control reader returned %T", value)
	}
	if len(output.Data) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(output.Data) != 1 || output.Data[0] == nil {
		return nil, fmt.Errorf("ambiguous audit scope")
	}
	return output.Data[0], nil
}

func (s *Store) Configure(ctx context.Context, row *controlwrite.AuditScope) error {
	input := &controlwrite.Input{}
	input.SetScopes([]*controlwrite.AuditScope{row})
	value, err := s.runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: s.controlWrite, Input: input})
	if err != nil {
		return err
	}
	if _, ok := value.(*controlwrite.Output); !ok {
		return fmt.Errorf("audit control writer returned %T", value)
	}
	return nil
}
