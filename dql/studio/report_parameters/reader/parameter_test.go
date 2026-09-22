package reader_test

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

func TestReportParameterReaderCompilesNestedPredicates(t *testing.T) {
	ctx := context.Background()
	payload, err := os.ReadFile("parameter.dql")
	if err != nil {
		t.Fatal(err)
	}
	resources, err := resource.New().WithDefault(os.DirFS("."))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := transcribe.NewCompiler().Compile(ctx, &transcribe.Source{
		Scope: "github.com/viant/datly-studio/dql/studio/report_parameters/reader", Name: "parameter", Path: "parameter.dql",
		Text: string(payload), Connector: "studio", Resources: resources, Types: datatest.StudioAuthorizationTypes(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = (routes.Compiler{Component: compiled.Component}).Compile(); err != nil {
		t.Fatal(err)
	}
	if len(compiled.Component.RootView.Relations) != 1 || !strings.EqualFold(compiled.Component.RootView.Relations[0].View.CanonicalName(), "Predicates") {
		t.Fatalf("parameter relations=%+v", compiled.Component.RootView.Relations)
	}
}
