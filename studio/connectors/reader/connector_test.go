package reader

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	requestprovider "github.com/viant/bindly/provider/request"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly/bootstrap"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	dtag "github.com/viant/datly/tag"
	"github.com/viant/xdatly/response"
)

func TestConnectorReaderMinimumContract(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "reader", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{
			{"name": "alpha", "driver": "sqlite", "dsn_template": "sqlite://private-token", "description": "Primary analytics", "owner_id": "owner-a", "status": "active", "etag": 1, "created_at": "2026-09-17 10:00:00", "updated_at": "2026-09-17 10:00:00"},
			{"name": "beta", "driver": "mysql", "description": "Archive", "owner_id": "owner-b", "status": "disabled", "etag": 2, "created_at": "2026-09-17 11:00:00", "updated_at": "2026-09-17 11:00:00"},
			{"name": "gamma", "driver": "mysql", "description": "Gamma analytics", "owner_id": "owner-a", "status": "active", "etag": 3, "created_at": "2026-09-17 12:00:00", "updated_at": "2026-09-17 12:00:00"},
			{"name": "removed", "driver": "mysql", "description": "Deleted", "owner_id": "owner-a", "status": "deleted", "etag": 4, "created_at": "2026-09-17 13:00:00", "updated_at": "2026-09-17 13:00:00", "deleted_at": "2026-09-17 13:30:00"},
		}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{
			{"owner_id": "owner-a", "name": "general", "title": "General", "status": "active", "etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"},
			{"owner_id": "owner-b", "name": "general", "title": "General", "status": "active", "etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"},
		}},
		datatest.Table{Name: "reports", Rows: []datatest.Row{
			{"id": "r-alpha", "slug": "alpha", "title": "Alpha", "owner_id": "owner-a", "status": "active", "default_connector_name": "alpha", "component_scope": "reports", "component_name": "alpha", "created_at": "2026-09-17 10:00:00", "updated_at": "2026-09-17 10:00:00"},
			{"id": "r-beta", "slug": "beta", "title": "Beta", "owner_id": "owner-b", "status": "active", "default_connector_name": "beta", "component_scope": "reports", "component_name": "beta", "created_at": "2026-09-17 10:00:00", "updated_at": "2026-09-17 10:00:00"},
			{"id": "r-gamma", "slug": "gamma", "title": "Gamma", "owner_id": "owner-a", "status": "active", "default_connector_name": "gamma", "component_scope": "reports", "component_name": "gamma", "created_at": "2026-09-17 10:00:00", "updated_at": "2026-09-17 10:00:00"},
		}},
		datatest.Table{Name: "report_acl", Rows: []datatest.Row{
			{"report_id": "r-alpha", "subject_type": "user", "subject_id": "viewer", "can_view": true},
			{"report_id": "r-beta", "subject_type": "user", "subject_id": "viewer", "can_view": true},
			{"report_id": "r-gamma", "subject_type": "user", "subject_id": "viewer", "can_view": true},
		}},
	); err != nil {
		t.Fatal(err)
	}

	holder := reflect.TypeOf(ConnectorComponent{})
	field, ok := holder.FieldByName("Contract1")
	if !ok {
		t.Fatal("missing reader component holder")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatal(err)
	}
	source := &bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: "reader", PackagePath: holder.PkgPath(), Tag: metadata, InputType: "ConnectorsInput", OutputType: "ConnectorsOutput"}
	component, err := source.Resolve(reflect.TypeOf(Input{}), reflect.TypeOf(Output{}))
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(ConnectorDatlyResourceNamespace, ConnectorDatlyResources); err != nil {
		t.Fatal(err)
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeOf(Input{}), OutputType: reflect.TypeOf(Output{}), Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory})
	if err != nil {
		t.Fatal(err)
	}
	execution, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: &dsql.SQLComponent{DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := artifact.Registration(registry.RegisteredComponent{Reader: execution})
	if err != nil {
		t.Fatal(err)
	}
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{authEntry, entry}, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })

	invoke := func(path string, authorization ...string) (*Output, error) {
		request := httptest.NewRequest("GET", path, nil)
		token := jwt.Bearer(t, "viewer")
		if len(authorization) > 0 {
			token = authorization[0]
		}
		if token != "" {
			request.Header.Set("Authorization", token)
		}
		routePath := "/v1/studio/connectors"
		pathParams := map[string]string(nil)
		if request.URL.Path != routePath {
			routePath += "/{name}"
			pathParams = map[string]string{"name": strings.TrimPrefix(request.URL.Path, "/v1/studio/connectors/")}
		}
		scope, scopeErr := requestprovider.New(request, requestprovider.WithPathParams(pathParams))
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		actual, invokeErr := runtime.ExecuteRoute(ctx, "GET", routePath, scope)
		if invokeErr != nil {
			return nil, invokeErr
		}
		return actual.(*Output), nil
	}
	names := func(output *Output) []string {
		result := make([]string, 0, len(output.Connectors))
		for _, connector := range output.Connectors {
			if connector != nil && connector.Name != nil {
				result = append(result, *connector.Name)
			}
		}
		return result
	}

	cases := []struct {
		name string
		path string
		want []string
	}{
		{"all predicates absent", "/v1/studio/connectors?orderBy=name", []string{"alpha", "beta", "gamma"}},
		{"by name", "/v1/studio/connectors/" + url.PathEscape("beta"), []string{"beta"}},
		{"by name missing", "/v1/studio/connectors/" + url.PathEscape("missing"), []string{}},
		{"search OR group", "/v1/studio/connectors?q=analytics&orderBy=name", []string{"alpha", "gamma"}},
		{"status predicate", "/v1/studio/connectors?status=active&orderBy=name", []string{"alpha", "gamma"}},
		{"owner predicate", "/v1/studio/connectors?owner=owner-b", []string{"beta"}},
		{"driver predicate", "/v1/studio/connectors?driver=mysql&orderBy=name", []string{"beta", "gamma"}},
		{"AND predicate group", "/v1/studio/connectors?status=active&owner=owner-a&driver=mysql", []string{"gamma"}},
		{"combined groups", "/v1/studio/connectors?q=analytics&status=active&owner=owner-a&orderBy=name", []string{"alpha", "gamma"}},
		{"explicit empty activates predicate", "/v1/studio/connectors?status=", []string{}},
		{"pagination", "/v1/studio/connectors?orderBy=name&limit=1&offset=1", []string{"beta"}},
		{"descending order", "/v1/studio/connectors?orderBy=" + url.QueryEscape("name DESC"), []string{"gamma", "beta", "alpha"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			output, err := invoke(test.path)
			if err != nil {
				t.Fatal(err)
			}
			if actual := names(output); !reflect.DeepEqual(actual, test.want) {
				t.Fatalf("%s names=%v want=%v", test.path, actual, test.want)
			}
		})
	}

	t.Run("missing JWT is rejected before protected read", func(t *testing.T) {
		if _, err := invoke("/v1/studio/connectors", ""); err == nil || response.ErrorStatusCode(err, 500) != 401 {
			t.Fatalf("missing JWT error=%v, want 401", err)
		}
	})

	t.Run("field projection", func(t *testing.T) {
		output, err := invoke("/v1/studio/connectors?fields=name&fields=status&orderBy=name&limit=1")
		if err != nil {
			t.Fatal(err)
		}
		if len(output.Connectors) != 1 || output.Connectors[0].Name == nil || output.Connectors[0].Status == nil || output.Connectors[0].Driver != nil {
			t.Fatalf("projected connector=%+v", output.Connectors)
		}
	})

	t.Run("DSN template is not part of the route contract", func(t *testing.T) {
		output, err := invoke("/v1/studio/connectors/alpha")
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := reflect.TypeOf(Connector{}).FieldByName("DsnTemplate"); exists {
			t.Fatal("connector reader shape exposes DsnTemplate")
		}
		if _, exists := reflect.TypeOf(Connector{}).FieldByName("SecretRef"); exists {
			t.Fatal("connector reader shape exposes SecretRef")
		}
		payload, err := json.Marshal(output)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(payload), "sqlite://private-token") || strings.Contains(string(payload), "dsn_template") || strings.Contains(string(payload), "secret_ref") {
			t.Fatalf("serialized connector response exposed connection material: %s", payload)
		}
	})

	t.Run("disallowed order", func(t *testing.T) {
		if _, err := invoke("/v1/studio/connectors?orderBy=secret_ref"); err == nil {
			t.Fatal("disallowed order column was accepted")
		}
	})

	t.Run("predicate presence markers", func(t *testing.T) {
		input := Input{}
		input.SetStatus("")
		if input.Has == nil || !input.Has.Status || input.Has.Query || input.Has.Driver {
			t.Fatalf("predicate presence=%+v", input.Has)
		}
	})
}
