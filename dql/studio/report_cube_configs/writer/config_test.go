package writer_test

import (
	"context"
	"os"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
)

func TestReportCubeConfigWriterCompiles(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "cube_writer_compile", "studio")
	_ = db
	payload, err := os.ReadFile("config.dql")
	if err != nil {
		t.Fatal(err)
	}
	resources, err := resource.New().WithDefault(os.DirFS("."))
	if err != nil {
		t.Fatal(err)
	}
	_, err = transcribe.NewCompiler().Compile(ctx, &transcribe.Source{Scope: "github.com/viant/datly-studio/dql/studio/report_cube_configs/writer", Name: "config", Path: "config.dql", Text: string(payload), Connector: "studio", Resources: resources, ColumnRefiner: column.New(column.Connections{"studio": db}), Types: datatest.StudioAuthorizationTypes(t)})
	if err != nil {
		t.Fatal(err)
	}
}
