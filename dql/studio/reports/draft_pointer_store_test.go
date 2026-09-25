package reports_test

import (
	"context"
	"strings"
	"testing"

	"github.com/viant/datly-studio/internal/datatest"
	bootstraproutes "github.com/viant/datly/bootstrap/routes"
	"github.com/viant/datly/transcribe/column"
)

// The draft pointer writer projects only the columns the import moves, keys
// on the report id and uses etag as its concurrency token.
func TestDraftPointerStoreWriterContract(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "draft_pointer_store", "studio")
	compiled := compileReport(t, ctx, "store_draft_pointer", column.New(column.Connections{"studio": db}))
	if err := column.ApplyWriterMetadata(compiled.Component); err != nil {
		t.Fatal(err)
	}
	if err := (bootstraproutes.Compiler{Component: compiled.Component}).Compile(); err != nil {
		t.Fatal(err)
	}
	routes := map[string]bool{}
	for _, route := range compiled.Component.Routes {
		routes[route.Method+" "+route.Path] = true
		if len(route.MCP) != 0 {
			t.Fatalf("server-only draft pointer writer must not be an MCP exposure: %+v", route.MCP)
		}
	}
	if len(routes) != 1 || !routes["PATCH /_studio/report-store/draft-pointer"] {
		t.Fatalf("draft pointer routes = %#v", routes)
	}
	view := compiled.Component.RootView
	if view.EntityHooks != "DraftPointerRules" {
		t.Fatalf("lifecycle hooks=%q", view.EntityHooks)
	}
	columns := map[string]bool{}
	keys, concurrency := map[string]bool{}, false
	for _, field := range view.Columns {
		if field == nil {
			continue
		}
		columns[strings.ToLower(field.Name)] = true
		if field.PrimaryKey {
			keys[strings.ToLower(field.Name)] = true
		}
		concurrency = concurrency || strings.EqualFold(field.Name, "etag") && field.ConcurrencyToken
	}
	if len(keys) != 1 || !keys["id"] || !concurrency {
		t.Fatalf("draft pointer mutation metadata: keys=%v concurrency=%v", keys, concurrency)
	}
	if len(columns) != 4 || !columns["id"] || !columns["current_draft_version"] || !columns["etag"] || !columns["updated_at"] {
		t.Fatalf("draft pointer must stay a sparse projection, got %v", columns)
	}
	if len(view.Relations) != 0 {
		t.Fatalf("draft pointer writer must not own related rows: %+v", view.Relations)
	}
}
