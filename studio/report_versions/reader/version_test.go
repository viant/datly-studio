package reader

import (
	"context"
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
)

func TestReportVersionReaderMinimumContract(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "reader", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := (datatest.HydrationPhase{JSON: []byte(reportVersionReaderDataset)}).Apply(ctx, db); err != nil {
		t.Fatal(err)
	}

	holder := reflect.TypeOf(VersionComponent{})
	field, _ := holder.FieldByName("Contract1")
	metadata, _, err := dtag.ParseComponent(field.Tag)
	if err != nil {
		t.Fatal(err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: "reader", PackagePath: holder.PkgPath(), Tag: metadata, InputType: "Input", OutputType: "Output"}).Resolve(reflect.TypeOf(Input{}), reflect.TypeOf(Output{}))
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
	execution, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: &dsql.SQLComponent{DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := artifact.Registration(registry.RegisteredComponent{Reader: execution})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{entry}, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })

	invoke := func(path string, subjects ...string) (*Output, error) {
		request := httptest.NewRequest("GET", path, nil)
		subject := "owner-a"
		if len(subjects) > 0 {
			subject = subjects[0]
		}
		request.Header.Set("Authorization", jwt.Bearer(t, subject))
		routePath := "/v1/studio/reports/{reportId}/versions"
		parts := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
		params := map[string]string{"reportId": parts[3]}
		if len(parts) == 6 {
			routePath += "/{versionNo}"
			params["versionNo"] = parts[5]
		}
		scope, scopeErr := requestprovider.New(request, requestprovider.WithPathParams(params))
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
	versions := func(output *Output) []int {
		result := make([]int, 0, len(output.Versions))
		for _, item := range output.Versions {
			if item != nil && item.VersionNo != nil {
				result = append(result, *item.VersionNo)
			}
		}
		return result
	}
	for _, test := range []struct {
		name, path string
		want       []int
	}{
		{"report scope", "/v1/studio/reports/r-alpha/versions?orderBy=version_no", []int{1, 2, 3}},
		{"version route", "/v1/studio/reports/r-alpha/versions/2", []int{2}},
		{"state", "/v1/studio/reports/r-alpha/versions?state=draft", []int{2}},
		{"authoring mode", "/v1/studio/reports/r-alpha/versions?authoringMode=structured", []int{3}},
		{"compile status", "/v1/studio/reports/r-alpha/versions?compileStatus=valid&orderBy=version_no", []int{1, 3}},
		{"created by", "/v1/studio/reports/r-alpha/versions?createdBy=alice&orderBy=version_no", []int{1, 3}},
		{"explicit empty", "/v1/studio/reports/r-alpha/versions?state=", []int{}},
		{"pagination", "/v1/studio/reports/r-alpha/versions?orderBy=version_no&limit=1&offset=1", []int{2}},
		{"descending", "/v1/studio/reports/r-alpha/versions?orderBy=" + url.QueryEscape("version_no DESC"), []int{3, 2, 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			output, invokeErr := invoke(test.path)
			if invokeErr != nil {
				t.Fatal(invokeErr)
			}
			if actual := versions(output); !reflect.DeepEqual(actual, test.want) {
				t.Fatalf("%s versions=%v want=%v", test.path, actual, test.want)
			}
		})
	}
	t.Run("field projection", func(t *testing.T) {
		output, invokeErr := invoke("/v1/studio/reports/r-alpha/versions?fields=version_no&fields=state&orderBy=version_no&limit=1")
		if invokeErr != nil {
			t.Fatal(invokeErr)
		}
		if len(output.Versions) != 1 || output.Versions[0].VersionNo == nil || output.Versions[0].State == nil || output.Versions[0].AuthoringMode != nil {
			t.Fatalf("projected report version=%+v", output.Versions)
		}
	})
	if _, err = invoke("/v1/studio/reports/r-alpha/versions?orderBy=component_spec_json"); err == nil {
		t.Fatal("disallowed order accepted")
	}
	t.Run("authorization predicate scopes owner and ACL access", func(t *testing.T) {
		denied, invokeErr := invoke("/v1/studio/reports/r-alpha/versions?orderBy=version_no", "owner-b")
		if invokeErr != nil {
			t.Fatal(invokeErr)
		}
		if actual := versions(denied); len(actual) != 0 {
			t.Fatalf("unrelated subject versions=%v", actual)
		}
		allowed, invokeErr := invoke("/v1/studio/reports/r-alpha/versions?orderBy=version_no", "viewer")
		if invokeErr != nil {
			t.Fatal(invokeErr)
		}
		if actual := versions(allowed); !reflect.DeepEqual(actual, []int{1, 2, 3}) {
			t.Fatalf("ACL subject versions=%v", actual)
		}
	})
	input := Input{}
	input.SetState("")
	if input.Has == nil || !input.Has.State || input.Has.CompileStatus || input.Has.VersionNo {
		t.Fatalf("predicate presence=%+v", input.Has)
	}
}

const reportVersionReaderDataset = `{"tables":[
  {"name":"connectors","rows":[
    {"name":"main","driver":"sqlite","owner_id":"owner-a","status":"active","created_at":"2026-09-17 09:00:00","updated_at":"2026-09-17 09:00:00"}
  ]},
  {"name":"namespaces","rows":[
    {"owner_id":"owner-a","name":"general","title":"General","status":"active","created_at":"2026-09-17 09:00:00","updated_at":"2026-09-17 09:00:00"},
    {"owner_id":"owner-b","name":"general","title":"General","status":"active","created_at":"2026-09-17 09:00:00","updated_at":"2026-09-17 09:00:00"}
  ]},
  {"name":"reports","rows":[
    {"id":"r-alpha","slug":"alpha","title":"Alpha","owner_id":"owner-a","status":"active","default_connector_name":"main","component_scope":"reports/alpha","component_name":"alpha","created_at":"2026-09-17 09:00:00","updated_at":"2026-09-17 09:00:00"},
    {"id":"r-beta","slug":"beta","title":"Beta","owner_id":"owner-b","status":"draft","default_connector_name":"main","component_scope":"reports/beta","component_name":"beta","created_at":"2026-09-17 09:00:00","updated_at":"2026-09-17 09:00:00"}
  ]},
  {"name":"report_acl","rows":[
    {"report_id":"r-alpha","subject_type":"user","subject_id":"viewer","can_view":true}
  ]},
  {"name":"report_versions","rows":[
    {"report_id":"r-alpha","version_no":1,"state":"published","authoring_mode":"sql","component_spec_json":{},"spec_format_version":"1","spec_hash":"alpha-1","type_manifest_json":{},"compile_status":"valid","datly_version":"v1","compiler_version":"v1","source_revision":1,"created_by":"alice","created_at":"2026-09-17 10:00:00"},
    {"report_id":"r-alpha","version_no":2,"state":"draft","authoring_mode":"dql","component_spec_json":{},"spec_format_version":"1","spec_hash":"alpha-2","type_manifest_json":{},"compile_status":"pending","datly_version":"v1","compiler_version":"v1","source_revision":2,"created_by":"bob","created_at":"2026-09-17 11:00:00"},
    {"report_id":"r-alpha","version_no":3,"state":"validated","authoring_mode":"structured","component_spec_json":{},"spec_format_version":"1","spec_hash":"alpha-3","type_manifest_json":{},"compile_status":"valid","datly_version":"v1","compiler_version":"v1","source_revision":3,"created_by":"alice","created_at":"2026-09-17 12:00:00"},
    {"report_id":"r-beta","version_no":1,"state":"draft","authoring_mode":"sql","component_spec_json":{},"spec_format_version":"1","spec_hash":"beta-1","type_manifest_json":{},"compile_status":"invalid","datly_version":"v1","compiler_version":"v1","source_revision":1,"created_by":"carol","created_at":"2026-09-17 13:00:00"}
  ]}
]}`
