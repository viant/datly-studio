package sqltransport

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/bindly/locator"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/sdk"
	stored "github.com/viant/datly-studio/studio/report_acl/store_write"
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
)

// writeACL invokes only the private Datly writer. The SDK authorizes the actor
// and validates the subject and capability invariants before this call.
func (t *Transport) writeACL(ctx context.Context, operation string, row *stored.StoredACL) error {
	resources := resource.New()
	if err := resources.Register(stored.AclDatlyResourceNamespace, stored.AclDatlyResources); err != nil {
		return err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return err
	}
	holder := reflect.TypeOf(stored.AclComponent{})
	contract, ok := holder.FieldByName("Contract")
	if !ok {
		return fmt.Errorf("ACL store writer has no component contract")
	}
	metadata, present, err := dtag.ParseComponent(contract.Tag)
	if err != nil {
		return err
	}
	if !present {
		return fmt.Errorf("ACL store writer has no component metadata")
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: contract.Name,
		PackageName: "store_write", PackagePath: holder.PkgPath(), Tag: metadata,
		InputType: "Input", OutputType: "Output"}).Resolve(reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}))
	if err != nil {
		return err
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeOf(stored.Input{}), OutputType: reflect.TypeOf(stored.Output{}), Resources: resources})
	if err != nil {
		return err
	}
	views, err := viewprovider.New(viewprovider.Config{Dependencies: artifact.ViewDependencies, Input: artifact.Input, SQL: connector})
	if err != nil {
		return err
	}
	handler, err := writerhandler.New(component, reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), operation)
	if err != nil {
		return err
	}
	registration := &registry.RegisteredComponent{Component: artifact.Component, Input: artifact.Input,
		Output: artifact.Output, OutputType: reflect.TypeOf(stored.Output{}), Handler: handler,
		Providers: []locator.Provider{views}, DataSource: dml.Source{DB: t.DB}}
	registration.Capabilities.Connector = connector
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return err
	}
	defer runtime.Shutdown(context.Background())
	target := dexec.ComponentTarget{Component: component.Key}
	if len(component.Routes) > 0 && component.Routes[0] != nil {
		target.Route = spec.RouteRef{Method: component.Routes[0].Method, Path: component.Routes[0].Path}
	}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target,
		Input: &stored.Input{Access: []*stored.StoredACL{row}, Has: &stored.InputHas{Access: true}}})
	if err != nil {
		return err
	}
	if _, ok := value.(*stored.Output); !ok {
		return fmt.Errorf("ACL store writer returned %T", value)
	}
	return nil
}

func aclWriteRow(value sdk.ReportACL, deleting bool) *stored.StoredACL {
	reportID, subjectType, subjectID := value.ReportID, value.SubjectType, value.SubjectID
	token := int(value.ETag)
	if token == 0 && !deleting {
		token = 1
	}
	row := &stored.StoredACL{ReportId: &reportID, SubjectType: &subjectType, SubjectId: &subjectID,
		Etag: &token, Has: &stored.StoredACLHas{ReportId: true, SubjectType: true, SubjectId: true, Etag: true}}
	if deleting {
		row.ShouldDelete = true
		row.Has.ShouldDelete = true
		return row
	}
	canView, canRun, canEdit := aclInt(value.CanView), aclInt(value.CanRun), aclInt(value.CanEdit)
	canPublish, canUseDQL := aclInt(value.CanPublish), aclInt(value.CanUseDQL)
	row.CanView, row.CanRun, row.CanEdit, row.CanPublish, row.CanUseDql = &canView, &canRun, &canEdit, &canPublish, &canUseDQL
	row.Has.CanView, row.Has.CanRun, row.Has.CanEdit, row.Has.CanPublish, row.Has.CanUseDql = true, true, true, true, true
	return row
}

func aclInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
