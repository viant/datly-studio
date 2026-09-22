package writer

import (
	"context"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/bindly/locator"
	requestprovider "github.com/viant/bindly/provider/request"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly/bootstrap"
	druntime "github.com/viant/datly/runtime"
	writerhandler "github.com/viant/datly/runtime/handler/writer"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/sql/dml"
	viewprovider "github.com/viant/datly/sql/reader/provider"
	dtag "github.com/viant/datly/tag"
	"github.com/viant/xdatly/response"
)

func TestConnectorWriterMinimumContract(t *testing.T) {
	datatest.AssertFilesAbsent(t, ".", obsoleteWriterFiles()...)
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "writer", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db, datatest.Table{Name: "connectors", Rows: []datatest.Row{
		{"name": "alpha", "driver": "sqlite", "description": "before", "owner_id": "owner-a", "status": "active", "etag": 2, "created_at": "2026-09-17 10:00:00", "updated_at": "2026-09-17 10:00:00"},
	}}); err != nil {
		t.Fatal(err)
	}

	holder := reflect.TypeOf(ConnectorComponent{})
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("missing writer component holder")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatal(err)
	}
	source := &bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: "writer", PackagePath: holder.PkgPath(), Tag: metadata, InputType: "ConnectorPatchInput", OutputType: "ConnectorPatchOutput"}
	component, err := source.Resolve(reflect.TypeOf(Input{}), reflect.TypeOf(Output{}))
	if err != nil {
		t.Fatal(err)
	}
	if component.Settings == nil || component.Settings.Output == nil || !reflect.DeepEqual(component.Settings.Output.Exclude, []string{"Data.DsnTemplate", "Data.SecretRef"}) {
		t.Fatalf("connector output redaction=%+v", component.Settings)
	}
	resources := resource.New()
	if err = resources.Register(ConnectorDatlyResourceNamespace, ConnectorDatlyResources); err != nil {
		t.Fatal(err)
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeOf(Input{}), OutputType: reflect.TypeOf(Output{}), Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory})
	if err != nil {
		t.Fatal(err)
	}
	sqlComponent := &dsql.SQLComponent{DB: db}
	if err = sqlComponent.RegisterConnector("studio", db); err != nil {
		t.Fatal(err)
	}
	views, err := viewprovider.New(viewprovider.Config{Dependencies: artifact.ViewDependencies, Input: artifact.Input, SQL: sqlComponent})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := writerhandler.New(component, reflect.TypeOf(Input{}), reflect.TypeOf(Output{}), "patch")
	if err != nil {
		t.Fatal(err)
	}
	registered := &druntime.RegisteredComponent{Component: artifact.Component, Input: artifact.Input, Output: artifact.Output, OutputType: reflect.TypeOf(Output{}), Handler: handler, Providers: []locator.Provider{views}, DataSource: dml.Source{DB: db}}
	registered.Capabilities.Connector = sqlComponent
	runtime, err := druntime.NewRuntime([]*druntime.RegisteredComponent{registered}, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	invoke := func(body string, authorization ...string) (*Output, error) {
		request := httptest.NewRequest("PATCH", "/v1/studio/connectors", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		token := jwt.Bearer(t, "owner-a")
		if len(authorization) > 0 {
			token = authorization[0]
		}
		request.Header.Set("Authorization", token)
		scope, scopeErr := requestprovider.New(request)
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		actual, invokeErr := runtime.ExecuteRoute(ctx, "PATCH", "/v1/studio/connectors", scope)
		if invokeErr != nil {
			return nil, invokeErr
		}
		return actual.(*Output), nil
	}

	t.Run("sparse update preserves omitted fields", func(t *testing.T) {
		output, err := invoke(`{"Data":[{"name":"alpha","etag":2,"description":"changed"}]}`)
		if err != nil {
			t.Fatal(err)
		}
		if output.Status.Status != "ok" {
			t.Fatalf("output=%+v", output)
		}
		datatest.AssertRows(t, ctx, db, "SELECT name,driver,description,owner_id,status,etag FROM connectors WHERE name='alpha'", nil,
			datatest.Row{"name": "alpha", "driver": "sqlite", "description": "changed", "owner_id": "owner-a", "status": "active", "etag": 2})
	})

	t.Run("stale concurrency token rolls back", func(t *testing.T) {
		_, err := invoke(`{"Data":[{"name":"alpha","etag":1,"description":"stale"}]}`)
		if err == nil || response.ErrorStatusCode(err, 500) != 409 && response.ErrorStatusCode(err, 500) != 422 {
			t.Fatalf("stale etag error=%v", err)
		}
		datatest.AssertRows(t, ctx, db, "SELECT description,etag FROM connectors WHERE name='alpha'", nil,
			datatest.Row{"description": "changed", "etag": 2})
	})

	t.Run("explicit null differs from omission", func(t *testing.T) {
		if _, err := invoke(`{"Data":[{"name":"alpha","etag":2,"description":null}]}`); err != nil {
			t.Fatal(err)
		}
		datatest.AssertRows(t, ctx, db, "SELECT description,driver,status FROM connectors WHERE name='alpha'", nil,
			datatest.Row{"description": nil, "driver": "sqlite", "status": "active"})
	})

	t.Run("insert", func(t *testing.T) {
		output, err := invoke(`{"Data":[{"name":"beta","driver":"mysql","dsnTemplate":"user:pass@tcp(localhost:3306)/studio","ownerId":"owner-a","status":"draft","etag":1,"createdAt":"2026-09-17T11:00:00Z","updatedAt":"2026-09-17T11:00:00Z"}]}`)
		if err != nil {
			t.Fatal(err)
		}
		if output.Status.Status != "ok" {
			t.Fatalf("insert output=%+v", output)
		}
		datatest.AssertRows(t, ctx, db, "SELECT name,driver,owner_id,status,etag FROM connectors WHERE name='beta'", nil,
			datatest.Row{"name": "beta", "driver": "mysql", "owner_id": "owner-a", "status": "draft", "etag": 1})
	})

	t.Run("cross owner insert is forbidden", func(t *testing.T) {
		_, err := invoke(`{"Data":[{"name":"forbidden","driver":"sqlite","dsnTemplate":"file:forbidden.db","ownerId":"owner-b","status":"draft","etag":1,"createdAt":"2026-09-17T11:00:00Z","updatedAt":"2026-09-17T11:00:00Z"}]}`)
		if err == nil || response.ErrorStatusCode(err, 500) != 403 {
			t.Fatalf("cross owner insert error=%v, want 403", err)
		}
		datatest.AssertRows(t, ctx, db, "SELECT name FROM connectors WHERE name='forbidden'", nil)
	})

	t.Run("generated connector validation", func(t *testing.T) {
		for _, test := range []struct {
			name string
			body string
		}{
			{"name required", `{"Data":[{"driver":"sqlite","dsnTemplate":"file:missing.db","ownerId":"owner-a","status":"draft","etag":1,"createdAt":"2026-09-17T11:00:00Z","updatedAt":"2026-09-17T11:00:00Z"}]}`},
			{"driver supported", `{"Data":[{"name":"oracle","driver":"oracle","dsnTemplate":"oracle://localhost/studio","ownerId":"owner-a","status":"draft","etag":1,"createdAt":"2026-09-17T11:00:00Z","updatedAt":"2026-09-17T11:00:00Z"}]}`},
			{"dsn matches driver", `{"Data":[{"name":"mismatch","driver":"sqlite","dsnTemplate":"bigquery://project/dataset","ownerId":"owner-a","status":"draft","etag":1,"createdAt":"2026-09-17T11:00:00Z","updatedAt":"2026-09-17T11:00:00Z"}]}`},
		} {
			t.Run(test.name, func(t *testing.T) {
				if _, err := invoke(test.body); err == nil || response.ErrorStatusCode(err, 500) != 422 {
					t.Fatalf("validation error=%v", err)
				}
			})
		}
		datatest.AssertRows(t, ctx, db, "SELECT name FROM connectors WHERE name IN ('oracle','mismatch') ORDER BY name", nil)
	})

	t.Run("false delete marker preserves row", func(t *testing.T) {
		if _, err := invoke(`{"Data":[{"name":"beta","etag":1,"shouldDelete":false}]}`); err != nil {
			t.Fatal(err)
		}
		datatest.AssertRows(t, ctx, db, "SELECT name FROM connectors WHERE name='beta'", nil, datatest.Row{"name": "beta"})
	})

	t.Run("explicit delete", func(t *testing.T) {
		if _, err := invoke(`{"Data":[{"name":"beta","etag":1,"shouldDelete":true}]}`); err != nil {
			t.Fatal(err)
		}
		datatest.AssertRows(t, ctx, db, "SELECT name FROM connectors WHERE name='beta'", nil)
	})
}

func obsoleteWriterFiles() []string {
	return []string{"actions.go", "entities.go", "frames.go", "invariants.go", "layout.go", "links.go", "mutation.go", "output_projection.go", "previous.go", "validation.go", "lifecycle_dispatch.go"}
}
