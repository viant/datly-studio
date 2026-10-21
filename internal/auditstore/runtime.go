// Package auditstore binds private generated Datly v1 audit components.
package auditstore

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/viant/bindly/locator"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	read "github.com/viant/datly-studio/studio/tool_call_audit/store_read"
	write "github.com/viant/datly-studio/studio/tool_call_audit/store_write"
	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	writerhandler "github.com/viant/datly/runtime/handler/writer"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/sql/dml"
	viewprovider "github.com/viant/datly/sql/reader/provider"
	dtag "github.com/viant/datly/tag"
	_ "github.com/viant/sqlx/metadata/product/mysql"
	_ "github.com/viant/sqlx/metadata/product/sqlite"
	"reflect"
)

type Store struct {
	runtime     *druntime.Runtime
	read, write dexec.ComponentTarget
}

func New(db *sql.DB) (*Store, error) {
	resources := resource.New()
	if err := resources.Register(read.AuditDatlyResourceNamespace, read.AuditDatlyResources); err != nil {
		return nil, err
	}
	if err := resources.Register(write.AuditDatlyResourceNamespace, write.AuditDatlyResources); err != nil {
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
	holder := reflect.TypeFor[write.AuditComponent]()
	contract, _ := holder.FieldByName("Contract")
	metadata, _, err := dtag.ParseComponent(contract.Tag)
	if err != nil {
		return nil, err
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: contract.Name, PackageName: "store_write", PackagePath: holder.PkgPath(), Tag: metadata, InputType: "Input", OutputType: "Output"}).Resolve(reflect.TypeFor[write.Input](), reflect.TypeFor[write.Output]())
	if err != nil {
		return nil, err
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeFor[write.Input](), OutputType: reflect.TypeFor[write.Output](), Resources: resources})
	if err != nil {
		return nil, err
	}
	var providers []locator.Provider
	if len(artifact.ViewDependencies) > 0 {
		views, err := viewprovider.New(viewprovider.Config{Dependencies: artifact.ViewDependencies, Input: artifact.Input, SQL: connector})
		if err != nil {
			return nil, err
		}
		providers = append(providers, views)
	}
	handler, err := writerhandler.New(component, reflect.TypeFor[write.Input](), reflect.TypeFor[write.Output](), "post")
	if err != nil {
		return nil, err
	}
	writer := &registry.RegisteredComponent{Component: artifact.Component, Input: artifact.Input, Output: artifact.Output, OutputType: reflect.TypeFor[write.Output](), Handler: handler, Providers: providers, DataSource: dml.Source{DB: db}}
	writer.Capabilities.Connector = connector
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{reader, writer}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	return &Store{runtime: runtime, read: readTarget, write: dexec.ComponentTarget{Component: component.Key, Route: spec.RouteRef{Method: "POST", Path: "/_studio/tool-call-audit/write"}}}, nil
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
