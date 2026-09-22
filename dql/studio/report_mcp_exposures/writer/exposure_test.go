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

func TestMCPExposureWriterCompiles(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "mcp_writer_compile", "studio")
	payload, err := os.ReadFile("exposure.dql")
	if err != nil {
		t.Fatal(err)
	}
	resources, err := resource.New().WithDefault(os.DirFS("."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = transcribe.NewCompiler().Compile(ctx, &transcribe.Source{Scope: "github.com/viant/datly-studio/dql/studio/report_mcp_exposures/writer", Name: "exposure", Path: "exposure.dql", Text: string(payload), Connector: "studio", Resources: resources, ColumnRefiner: column.New(column.Connections{"studio": db}), Types: datatest.StudioAuthorizationTypes(t)}); err != nil {
		t.Fatal(err)
	}
}
