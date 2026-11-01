package authoring

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/runtime/accessprovider"
	"github.com/viant/datly-studio/studio/namespaces/change"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

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
	t.Setenv("STUDIO_NAMESPACE_ADMIN_ROLE", "NAMESPACE_ADMIN")
	ctx := accessprovider.WithProvider(context.Background(), settingsProvider{})
	db := datatest.OpenSQLite(t, "settings_native_http", "studio")
	notifications := 0
	refreshFailure := false
	ctx = change.WithObserver(ctx, change.ObserverFunc(func(ctx context.Context, identities []change.Identity) error {
		notifications++
		require.Len(t, identities, 1)
		require.Equal(t, "owner", identities[0].OwnerID)
		require.Equal(t, "alpha", identities[0].Name)
		require.NotEmpty(t, identities[0].NamespaceID)
		require.EqualValues(t, notifications+1, identities[0].Revision)
		require.True(t, identities[0].SettingsChanged)
		require.True(t, identities[0].SecurityChanged)
		if notifications == 1 || notifications == 3 {
			require.NotNil(t, identities[0].Security)
			require.True(t, identities[0].Security.RequireInternalComponents)
		} else {
			require.Nil(t, identities[0].Security)
		}
		var etag int
		require.NoError(t, db.QueryRowContext(ctx, "SELECT etag FROM namespaces WHERE owner_id='owner' AND name='alpha'").Scan(&etag))
		require.Equal(t, notifications+1, etag, "notification must observe the committed token")
		if refreshFailure {
			return errors.New("namespace settings committed but runtime refresh failed")
		}
		return nil
	}))
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
		request = request.WithContext(ctx)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", jwt.Bearer(t, subject))
		response := httptest.NewRecorder()
		httpHandler.ServeHTTP(response, request)
		return response
	}
	response := call("owner", "settings_update", `{"name":"alpha","input":{"etag":1,"settings":{"security":{"requireInternalComponents":true},"resourceLimits":{"maxBytesProcessed":123}}}}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	require.Equal(t, 1, notifications)
	var output Output
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &output))
	require.EqualValues(t, 2, output.ETag)
	require.NotNil(t, output.Settings, response.Body.String())
	require.NotNil(t, output.Settings.ResourceLimits, response.Body.String())
	require.EqualValues(t, 123, *output.Settings.ResourceLimits.MaxBytesProcessed)
	require.NotNil(t, output.Settings.Security)
	require.True(t, output.Settings.Security.RequireInternalComponents)
	response = call("owner", "settings_get", `{"name":"alpha"}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	var persisted Output
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &persisted))
	require.NotNil(t, persisted.Settings.Security)
	require.True(t, persisted.Settings.Security.RequireInternalComponents)
	response = call("owner", "settings_update", `{"name":"alpha","input":{"etag":1,"settings":{"resourceLimits":{"maxBytesProcessed":456}}}}`)
	require.Equal(t, 409, response.Code, response.Body.String())
	require.Equal(t, 1, notifications, "stale write must not notify")
	response = call("other", "settings_get", `{"name":"alpha"}`)
	require.Equal(t, 403, response.Code, response.Body.String())
	var bytesProcessed string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT value_json FROM namespace_settings WHERE owner_id='owner' AND namespace='alpha' AND setting_key='resourceLimits'").Scan(&bytesProcessed))
	require.Contains(t, bytesProcessed, "123")
	response = call("owner", "settings_update", `{"name":"alpha","input":{"etag":2,"removeKeys":["resourceLimits","security"],"settings":{}}}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	var count int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM namespace_settings WHERE owner_id='owner' AND namespace='alpha'").Scan(&count))
	require.Zero(t, count)
	require.Equal(t, 2, notifications)
	refreshFailure = true
	response = call("owner", "settings_update", `{"name":"alpha","input":{"etag":3,"settings":{"security":{"requireInternalComponents":true}}}}`)
	require.GreaterOrEqual(t, response.Code, 400, response.Body.String())
	response = call("owner", "settings_get", `{"name":"alpha"}`)
	require.Equal(t, 200, response.Code, response.Body.String())
	var afterFailure Output
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &afterFailure))
	require.EqualValues(t, 4, afterFailure.ETag)
	require.True(t, afterFailure.Settings.Security.RequireInternalComponents)
	require.Equal(t, 3, notifications, "committed write must not be reported as rolled back")

}

type settingsProvider struct{}

func (settingsProvider) Resolve(context.Context) (authz.Facts, error) {
	return authz.Facts{Subject: "owner", Issuer: "settings-test", Tenant: "settings-test", Roles: []string{"NAMESPACE_ADMIN"}, ValidUntil: time.Now().Add(time.Minute)}, nil
}
