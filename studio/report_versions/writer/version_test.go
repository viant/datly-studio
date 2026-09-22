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
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/sql/dml"
	viewprovider "github.com/viant/datly/sql/reader/provider"
	dtag "github.com/viant/datly/tag"
	"github.com/viant/xdatly/response"
)

func TestReportVersionWriterMinimumContract(t *testing.T) {
	datatest.AssertFilesAbsent(t, ".", "actions.go", "entities.go", "frames.go", "invariants.go", "layout.go", "links.go", "mutation.go", "output_projection.go", "previous.go", "validation.go", "lifecycle_dispatch.go")
	if _, ok := reflect.TypeOf(ReportVersion{}).FieldByName("ShouldDelete"); ok {
		t.Fatal("immutable report version unexpectedly exposes physical deletion")
	}
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "writer", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := (datatest.HydrationPhase{JSON: []byte(reportVersionWriterDataset)}).Apply(ctx, db); err != nil {
		t.Fatal(err)
	}

	holder := reflect.TypeOf(VersionComponent{})
	field, _ := holder.FieldByName("Contract")
	metadata, _, err := dtag.ParseComponent(field.Tag)
	if err != nil {
		t.Fatal(err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: "writer", PackagePath: holder.PkgPath(), Tag: metadata, InputType: "Input", OutputType: "Output"}).Resolve(reflect.TypeOf(Input{}), reflect.TypeOf(Output{}))
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(VersionDatlyResourceNamespace, VersionDatlyResources); err != nil {
		t.Fatal(err)
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeOf(Input{}), OutputType: reflect.TypeOf(Output{}), Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory})
	if err != nil {
		t.Fatal(err)
	}
	sqlComponent := &dsql.SQLComponent{DB: db}
	if err := sqlComponent.RegisterConnector("studio", db); err != nil {
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
	registered := &registry.RegisteredComponent{Component: artifact.Component, Input: artifact.Input, Output: artifact.Output, OutputType: reflect.TypeOf(Output{}), Handler: handler, Providers: []locator.Provider{views}, DataSource: dml.Source{DB: db}}
	registered.Capabilities.Connector = sqlComponent
	runtime, err := druntime.NewRuntime([]*druntime.RegisteredComponent{registered}, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	invoke := func(body string, authorization ...string) (*Output, error) {
		request := httptest.NewRequest("PATCH", "/v1/studio/report-versions", strings.NewReader(body))
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
		actual, invokeErr := runtime.ExecuteRoute(ctx, "PATCH", "/v1/studio/report-versions", scope)
		if invokeErr != nil {
			return nil, invokeErr
		}
		return actual.(*Output), nil
	}

	t.Run("sparse update", func(t *testing.T) {
		output, err := invoke(`{"data":[{"reportId":"r-alpha","versionNo":1,"sourceRevision":2,"notes":"changed"}]}`)
		if err != nil {
			t.Fatal(err)
		}
		if output.Status.Status != "ok" || len(output.Data) != 1 || output.Data[0].Notes == nil || *output.Data[0].Notes != "changed" {
			t.Fatalf("output=%+v", output)
		}
		datatest.AssertRows(t, ctx, db, "SELECT report_id,version_no,state,authoring_mode,authored_dql,notes,source_revision FROM report_versions WHERE report_id='r-alpha' AND version_no=1", nil,
			datatest.Row{"report_id": "r-alpha", "version_no": 1, "state": "draft", "authoring_mode": "dql", "authored_dql": "SELECT 1", "notes": "changed", "source_revision": 2})
	})

	t.Run("stale concurrency", func(t *testing.T) {
		_, err := invoke(`{"data":[{"reportId":"r-alpha","versionNo":1,"sourceRevision":1,"notes":"stale"}]}`)
		if err == nil || response.ErrorStatusCode(err, 500) != 409 && response.ErrorStatusCode(err, 500) != 422 {
			t.Fatalf("stale source revision error=%v", err)
		}
		datatest.AssertRows(t, ctx, db, "SELECT notes FROM report_versions WHERE report_id='r-alpha' AND version_no=1", nil, datatest.Row{"notes": "changed"})
	})

	t.Run("explicit null", func(t *testing.T) {
		if _, err := invoke(`{"data":[{"reportId":"r-alpha","versionNo":1,"sourceRevision":2,"notes":null}]}`); err != nil {
			t.Fatal(err)
		}
		datatest.AssertRows(t, ctx, db, "SELECT notes,state FROM report_versions WHERE report_id='r-alpha' AND version_no=1", nil, datatest.Row{"notes": nil, "state": "draft"})
	})

	t.Run("insert", func(t *testing.T) {
		output, err := invoke(`{"data":[{"reportId":"r-alpha","versionNo":2,"state":"draft","authoringMode":"structured","componentSpecJson":{},"specFormatVersion":"1","specHash":"hash-2","typeManifestJson":{},"compileStatus":"pending","datlyVersion":"v1","compilerVersion":"v1","sourceRevision":1,"createdBy":"bob","createdAt":"2026-09-17T11:00:00Z"}]}`)
		if err != nil {
			t.Fatal(err)
		}
		if len(output.Data) != 1 || output.Data[0].VersionNo == nil || *output.Data[0].VersionNo != 2 {
			t.Fatalf("insert output=%+v", output)
		}
		datatest.AssertRows(t, ctx, db, "SELECT report_id,version_no,state,authoring_mode,compile_status,source_revision,created_by FROM report_versions WHERE report_id='r-alpha' AND version_no=2", nil,
			datatest.Row{"report_id": "r-alpha", "version_no": 2, "state": "draft", "authoring_mode": "structured", "compile_status": "pending", "source_revision": 1, "created_by": "bob"})
	})

	t.Run("unauthorized child insert is forbidden", func(t *testing.T) {
		_, err := invoke(`{"data":[{"reportId":"r-alpha","versionNo":99,"state":"draft","authoringMode":"structured","componentSpecJson":{},"specFormatVersion":"1","specHash":"forbidden","typeManifestJson":{},"compileStatus":"pending","datlyVersion":"v1","compilerVersion":"v1","sourceRevision":1,"createdBy":"attacker","createdAt":"2026-09-17T11:00:00Z"}]}`, jwt.Bearer(t, "attacker"))
		if err == nil || response.ErrorStatusCode(err, 500) != 403 {
			t.Fatalf("unauthorized insert error=%v, want 403", err)
		}
		datatest.AssertRows(t, ctx, db, "SELECT version_no FROM report_versions WHERE report_id='r-alpha' AND version_no=99", nil)
	})

	t.Run("full insert validation", func(t *testing.T) {
		for _, body := range []string{
			`{"data":[{"reportId":"r-alpha","versionNo":3,"state":"unknown","authoringMode":"sql","componentSpecJson":{},"specFormatVersion":"1","specHash":"hash-3","typeManifestJson":{},"compileStatus":"pending","datlyVersion":"v1","compilerVersion":"v1","sourceRevision":1,"createdBy":"bob","createdAt":"2026-09-17T12:00:00Z"}]}`,
			`{"data":[{"reportId":"missing","versionNo":1,"state":"draft","authoringMode":"sql","componentSpecJson":{},"specFormatVersion":"1","specHash":"missing-hash","typeManifestJson":{},"compileStatus":"pending","datlyVersion":"v1","compilerVersion":"v1","sourceRevision":1,"createdBy":"bob","createdAt":"2026-09-17T12:00:00Z"}]}`,
			`{"data":[{"reportId":"r-alpha","versionNo":4,"state":"draft","authoringMode":"sql","specFormatVersion":"1","specHash":"hash-4","typeManifestJson":{},"compileStatus":"pending","datlyVersion":"v1","compilerVersion":"v1","sourceRevision":1,"createdBy":"bob","createdAt":"2026-09-17T12:00:00Z"}]}`,
		} {
			if _, err := invoke(body); err == nil {
				t.Fatalf("invalid report version accepted: %s", body)
			}
		}
	})

	t.Run("setters mark sparse fields", func(t *testing.T) {
		value := &ReportVersion{}
		notes := "setter"
		value.SetNotes(&notes)
		if value.Has == nil || !value.Has.Notes || value.Has.State {
			t.Fatalf("setter presence=%+v", value.Has)
		}
	})
}

const reportVersionWriterDataset = `{"tables":[
  {"name":"connectors","rows":[
    {"name":"main","driver":"sqlite","owner_id":"owner-a","status":"active","created_at":"2026-09-17 09:00:00","updated_at":"2026-09-17 09:00:00"}
  ]},
  {"name":"namespaces","rows":[
    {"owner_id":"owner-a","name":"general","title":"General","status":"active","created_at":"2026-09-17 09:00:00","updated_at":"2026-09-17 09:00:00"}
  ]},
  {"name":"reports","rows":[
    {"id":"r-alpha","slug":"alpha","title":"Alpha","owner_id":"owner-a","status":"active","default_connector_name":"main","component_scope":"reports/alpha","component_name":"alpha","created_at":"2026-09-17 09:00:00","updated_at":"2026-09-17 09:00:00"}
  ]},
  {"name":"report_versions","rows":[
    {"report_id":"r-alpha","version_no":1,"state":"draft","authoring_mode":"dql","authored_dql":"SELECT 1","component_spec_json":{},"spec_format_version":"1","spec_hash":"hash-1","type_manifest_json":{},"compile_status":"pending","datly_version":"v1","compiler_version":"v1","source_revision":2,"notes":"before","created_by":"alice","created_at":"2026-09-17 10:00:00"}
  ]}
]}`
