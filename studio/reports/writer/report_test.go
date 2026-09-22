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

func TestReportWriterMinimumContract(t *testing.T) {
	datatest.AssertFilesAbsent(t, ".", "actions.go", "entities.go", "frames.go", "invariants.go", "layout.go", "links.go", "mutation.go", "output_projection.go", "previous.go", "validation.go", "lifecycle_dispatch.go")
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "writer", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{
			{"name": "main", "driver": "sqlite", "owner_id": "owner-a", "status": "active", "etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"},
		}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{
			{"owner_id": "owner-a", "name": "general", "title": "General", "status": "active", "etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"},
		}},
		datatest.Table{Name: "reports", Rows: []datatest.Row{
			{"id": "r-alpha", "slug": "alpha", "title": "Alpha", "description": "before", "owner_id": "owner-a", "status": "active", "default_connector_name": "main", "component_scope": "reports/alpha", "component_name": "alpha", "current_draft_version": 2, "etag": 2, "created_at": "2026-09-17 10:00:00", "updated_at": "2026-09-17 10:00:00"},
		}},
	); err != nil {
		t.Fatal(err)
	}

	holder := reflect.TypeOf(ReportComponent{})
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("missing report writer component holder")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatal(err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: "writer", PackagePath: holder.PkgPath(), Tag: metadata, InputType: "Input", OutputType: "Output"}).Resolve(reflect.TypeOf(Input{}), reflect.TypeOf(Output{}))
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(ReportDatlyResourceNamespace, ReportDatlyResources); err != nil {
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
	invoke := func(body string) (*Output, error) {
		request := httptest.NewRequest("PATCH", "/v1/studio/reports", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", jwt.Bearer(t, "owner-a"))
		scope, scopeErr := requestprovider.New(request)
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		actual, invokeErr := runtime.ExecuteRoute(ctx, "PATCH", "/v1/studio/reports", scope)
		if invokeErr != nil {
			return nil, invokeErr
		}
		return actual.(*Output), nil
	}

	t.Run("sparse update preserves omitted fields", func(t *testing.T) {
		output, err := invoke(`{"Data":[{"id":"r-alpha","etag":2,"description":"changed"}]}`)
		if err != nil {
			t.Fatal(err)
		}
		if output.Status.Status != "ok" || len(output.Data) != 1 || output.Data[0].Description == nil || *output.Data[0].Description != "changed" {
			t.Fatalf("output=%+v", output)
		}
		datatest.AssertRows(t, ctx, db, "SELECT id,slug,title,description,status,current_draft_version,etag FROM reports WHERE id='r-alpha'", nil,
			datatest.Row{"id": "r-alpha", "slug": "alpha", "title": "Alpha", "description": "changed", "status": "active", "current_draft_version": 2, "etag": 2})
	})

	t.Run("stale concurrency token rolls back", func(t *testing.T) {
		_, err := invoke(`{"Data":[{"id":"r-alpha","etag":1,"description":"stale"}]}`)
		if err == nil || response.ErrorStatusCode(err, 500) != 409 && response.ErrorStatusCode(err, 500) != 422 {
			t.Fatalf("stale etag error=%v", err)
		}
		datatest.AssertRows(t, ctx, db, "SELECT description,etag FROM reports WHERE id='r-alpha'", nil,
			datatest.Row{"description": "changed", "etag": 2})
	})

	t.Run("explicit null differs from omission", func(t *testing.T) {
		if _, err := invoke(`{"Data":[{"id":"r-alpha","etag":2,"description":null}]}`); err != nil {
			t.Fatal(err)
		}
		datatest.AssertRows(t, ctx, db, "SELECT description,title,status FROM reports WHERE id='r-alpha'", nil,
			datatest.Row{"description": nil, "title": "Alpha", "status": "active"})
	})

	t.Run("insert", func(t *testing.T) {
		output, err := invoke(`{"Data":[{"id":"r-beta","namespace":"general","slug":"beta","title":"Beta","ownerId":"owner-a","status":"draft","defaultConnectorName":"main","componentScope":"reports/beta","componentName":"beta","etag":1,"createdAt":"2026-09-17T11:00:00Z","updatedAt":"2026-09-17T11:00:00Z"}]}`)
		if err != nil {
			t.Fatal(err)
		}
		if len(output.Data) != 1 || output.Data[0].Id == nil || *output.Data[0].Id != "r-beta" {
			t.Fatalf("insert output=%+v", output)
		}
		datatest.AssertRows(t, ctx, db, "SELECT id,slug,owner_id,status,default_connector_name,etag FROM reports WHERE id='r-beta'", nil,
			datatest.Row{"id": "r-beta", "slug": "beta", "owner_id": "owner-a", "status": "draft", "default_connector_name": "main", "etag": 1})
	})

	t.Run("generated report validation", func(t *testing.T) {
		for _, test := range []struct {
			name string
			body string
		}{
			{"id required", `{"Data":[{"namespace":"general","slug":"missing-id","title":"Missing","ownerId":"owner-c","status":"draft","defaultConnectorName":"main","componentScope":"reports/missing","componentName":"missing","etag":1,"createdAt":"2026-09-17T11:00:00Z","updatedAt":"2026-09-17T11:00:00Z"}]}`},
			{"status supported", `{"Data":[{"id":"r-invalid","namespace":"general","slug":"invalid","title":"Invalid","ownerId":"owner-c","status":"deleted","defaultConnectorName":"main","componentScope":"reports/invalid","componentName":"invalid","etag":1,"createdAt":"2026-09-17T11:00:00Z","updatedAt":"2026-09-17T11:00:00Z"}]}`},
			{"connector exists", `{"Data":[{"id":"r-missing-connector","namespace":"general","slug":"missing-connector","title":"Missing connector","ownerId":"owner-c","status":"draft","defaultConnectorName":"missing","componentScope":"reports/missing-connector","componentName":"missing-connector","etag":1,"createdAt":"2026-09-17T11:00:00Z","updatedAt":"2026-09-17T11:00:00Z"}]}`},
		} {
			t.Run(test.name, func(t *testing.T) {
				if _, err := invoke(test.body); err == nil {
					t.Fatal("invalid report was accepted")
				}
			})
		}
		datatest.AssertRows(t, ctx, db, "SELECT id FROM reports WHERE id IN ('r-invalid','r-missing-connector') ORDER BY id", nil)
	})

	t.Run("false delete marker preserves row", func(t *testing.T) {
		if _, err := invoke(`{"Data":[{"id":"r-beta","etag":1,"shouldDelete":false}]}`); err != nil {
			t.Fatal(err)
		}
		datatest.AssertRows(t, ctx, db, "SELECT id FROM reports WHERE id='r-beta'", nil, datatest.Row{"id": "r-beta"})
	})

	t.Run("explicit delete", func(t *testing.T) {
		if _, err := invoke(`{"Data":[{"id":"r-beta","etag":1,"shouldDelete":true}]}`); err != nil {
			t.Fatal(err)
		}
		datatest.AssertRows(t, ctx, db, "SELECT id FROM reports WHERE id='r-beta'", nil)
	})
}
