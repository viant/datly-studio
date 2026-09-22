package reportviews_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly/bootstrap/routes"
	"github.com/viant/datly/transcribe"
)

func TestReportViewsReaderCompilesNestedFields(t *testing.T) {
	ctx := context.Background()
	path := "view.dql"
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	resources, err := resource.New().WithDefault(os.DirFS("."))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := transcribe.NewCompiler().Compile(ctx, &transcribe.Source{
		Scope: "github.com/viant/datly-studio/dql/studio/report_views/reader", Name: "view", Path: path,
		Text: string(payload), Connector: "studio", Resources: resources, Types: datatest.StudioAuthorizationTypes(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = (routes.Compiler{Component: compiled.Component}).Compile(); err != nil {
		t.Fatal(err)
	}
	if len(compiled.Component.RootView.Relations) != 1 || compiled.Component.RootView.Relations[0].View == nil {
		t.Fatalf("report-view relations=%+v", compiled.Component.RootView.Relations)
	}
	if !strings.EqualFold(compiled.Component.RootView.Relations[0].View.CanonicalName(), "Fields") {
		t.Fatalf("nested field relation=%s", compiled.Component.RootView.Relations[0].View.CanonicalName())
	}
	if len(compiled.Component.Routes) != 1 || compiled.Component.Routes[0].Path != "/v1/studio/reports/{reportId}/versions/{versionNo}/views" {
		t.Fatalf("report-view routes=%+v", compiled.Component.Routes)
	}
}
