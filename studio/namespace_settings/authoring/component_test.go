package authoring

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	read "github.com/viant/datly-studio/studio/namespace_settings/store_read"
	write "github.com/viant/datly-studio/studio/namespace_settings/store_write"
	namespaces "github.com/viant/datly-studio/studio/namespaces/store_read"
	"github.com/viant/datly/bootstrap"
	gateway "github.com/viant/datly/gateway/http"
	druntime "github.com/viant/datly/runtime"
	rh "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/writer"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/sql/dml"
	viewprovider "github.com/viant/datly/sql/reader/provider"
	dtag "github.com/viant/datly/tag"
)

func TestNativeHTTPSettingsAuthorizePersistAndRejectStaleWrite(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "settings_native_http", "studio")
	jwt := datatest.NewJWTFixture(t)
	require.NoError(t, datatest.Hydrate(ctx, db, datatest.Table{Name: "namespaces", Rows: []datatest.Row{{"owner_id": "owner", "name": "alpha", "title": "Alpha", "status": "active", "etag": 1, "created_at": "2026-10-28 00:00:00", "updated_at": "2026-10-28 00:00:00"}}}))
	resources := resource.New()
	require.NoError(t, resources.Register(namespaces.NamespaceDatlyResourceNamespace, namespaces.NamespaceDatlyResources))
	require.NoError(t, resources.Register(read.SettingsDatlyResourceNamespace, read.SettingsDatlyResources))
	require.NoError(t, resources.Register(write.SettingsDatlyResourceNamespace, write.SettingsDatlyResources))
	connector := &dsql.SQLComponent{DB: db}
	require.NoError(t, connector.RegisterConnector("studio", db))
	types := datatest.StudioAuthorizationTypes(t)
	compile := func(holder, in, out reflect.Type, handler rh.TypedHandler, mutation bool) *registry.RegisteredComponent {
		field, ok := holder.FieldByName("Contract")
		require.True(t, ok)
		metadata, present, err := dtag.ParseComponent(field.Tag)
		require.NoError(t, err)
		require.True(t, present)
		component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: holder.Name(), PackagePath: holder.PkgPath(), Tag: metadata, InputType: in.Name(), OutputType: out.Name()}).Resolve(in, out)
		require.NoError(t, err)
		artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: in, OutputType: out, Resources: resources, Types: types, CodecFactory: jwt.Factory})
		require.NoError(t, err)
		entry := registry.RegisteredComponent{Handler: handler, DataSource: dml.Source{DB: db}}
		if mutation {
			entry.Handler, err = writer.New(component, in, out, "patch")
			require.NoError(t, err)
			providers, e := viewprovider.New(viewprovider.Config{Dependencies: artifact.ViewDependencies, Input: artifact.Input, SQL: connector})
			require.NoError(t, e)
			entry.Providers = append(entry.Providers, providers)
		} else if handler == nil {
			entry.Reader, err = artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: connector})
			require.NoError(t, err)
		}
		registration, err := artifact.Registration(entry)
		require.NoError(t, err)
		registration.Capabilities.Connector = connector
		return registration
	}
	getHandler, err := (GetComponent{}).DatlyHandler("NewGet")()
	require.NoError(t, err)
	updateHandler, err := (UpdateComponent{}).DatlyHandler("NewUpdate")()
	require.NoError(t, err)
	entries := []*registry.RegisteredComponent{datatest.AuthRegistration(t, db, jwt.Factory, resources), compile(reflect.TypeFor[namespaces.NamespaceComponent](), reflect.TypeFor[namespaces.Input](), reflect.TypeFor[namespaces.Output](), nil, false), compile(reflect.TypeFor[read.SettingsComponent](), reflect.TypeFor[read.Input](), reflect.TypeFor[read.Output](), nil, false), compile(reflect.TypeFor[write.SettingsComponent](), reflect.TypeFor[write.Input](), reflect.TypeFor[write.Output](), nil, true), compile(reflect.TypeFor[GetComponent](), reflect.TypeFor[Input](), reflect.TypeFor[Output](), getHandler, false), compile(reflect.TypeFor[UpdateComponent](), reflect.TypeFor[Input](), reflect.TypeFor[Output](), updateHandler, false)}
	runtime, err := druntime.NewRuntime(entries, druntime.WithResources(resources))
	require.NoError(t, err)
	defer runtime.Shutdown(ctx)
	httpHandler := gateway.NewHandler(runtime, nil, "test")
	call := func(subject, path, body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/namespaces."+path, bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", jwt.Bearer(t, subject))
		response := httptest.NewRecorder()
		httpHandler.ServeHTTP(response, request)
		return response
	}
	response := call("owner", "settings_update", `{"name":"alpha","input":{"etag":1,"settings":{"resourceLimits":{"maxBytesProcessed":123}}}}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	var output Output
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &output))
	require.EqualValues(t, 2, output.ETag)
	require.NotNil(t, output.Settings, response.Body.String())
	require.NotNil(t, output.Settings.ResourceLimits, response.Body.String())
	require.EqualValues(t, 123, *output.Settings.ResourceLimits.MaxBytesProcessed)
	response = call("owner", "settings_update", `{"name":"alpha","input":{"etag":1,"settings":{"resourceLimits":{"maxBytesProcessed":456}}}}`)
	require.Equal(t, 409, response.Code, response.Body.String())
	response = call("other", "settings_get", `{"name":"alpha"}`)
	require.Equal(t, 403, response.Code, response.Body.String())
	var bytesProcessed string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT value_json FROM namespace_settings WHERE owner_id='owner' AND namespace='alpha' AND setting_key='resourceLimits'").Scan(&bytesProcessed))
	require.Contains(t, bytesProcessed, "123")
}
