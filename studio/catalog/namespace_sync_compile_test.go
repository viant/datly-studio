package catalog

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly/bootstrap"
	datlymcp "github.com/viant/datly/mcp"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/registry"
	dtag "github.com/viant/datly/tag"
)

func TestNamespaceSyncComponentsCompileWithOptionalAliasAndSharedWireFields(t *testing.T) {
	jwt := datatest.NewJWTFixture(t)
	resources := resource.New()
	db := datatest.OpenSQLite(t, "namespace_sync_schema", "studio")
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	for _, item := range []struct {
		holder, input, output reflect.Type
		handler               func() (rhandler.TypedHandler, error)
	}{
		{reflect.TypeFor[ListComponent](), reflect.TypeFor[ListInput](), reflect.TypeFor[ListOutput](), (ListComponent{}).DatlyHandler("NewList")},
		{reflect.TypeFor[GetComponent](), reflect.TypeFor[GetInput](), reflect.TypeFor[GetOutput](), (GetComponent{}).DatlyHandler("NewGet")},
	} {
		field, _ := item.holder.FieldByName("Contract")
		metadata, present, err := dtag.ParseComponent(field.Tag)
		require.NoError(t, err)
		require.True(t, present)
		component, err := (&bootstrap.RouteSource{HolderType: item.holder.Name(), FieldName: field.Name, PackageName: item.holder.Name(), PackagePath: item.holder.PkgPath(), Tag: metadata}).Resolve(item.input, item.output)
		require.NoError(t, err)
		handler, err := item.handler()
		require.NoError(t, err)
		artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: item.input, OutputType: item.output, Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory, Handler: handler, HandlerOwnedOutput: true})
		require.NoError(t, err)
		entry, err := artifact.Registration(registry.RegisteredComponent{})
		require.NoError(t, err)
		service, err := datlymcp.New(datlymcp.Config{Components: []*registry.RegisteredComponent{authEntry, entry}, Invoker: nativeSyncDirectory{}})
		require.NoError(t, err)
		if item.holder == reflect.TypeFor[GetComponent]() {
			tool, ok := service.Catalog().Tool("namespaces/get")
			require.True(t, ok)
			require.Contains(t, tool.Metadata().InputSchema.Properties, "namespace")
			require.Contains(t, tool.Metadata().InputSchema.Properties, "sync")
			require.NotContains(t, tool.Metadata().InputSchema.Required, "uri")
		}
	}
}
