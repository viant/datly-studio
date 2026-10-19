package sqltransport

import (
	"context"
	"strings"
	"testing"

	"github.com/viant/datly/authoring/readerbuilder"
)

func TestOutputFieldEditsPreserveInputAndOptions(t *testing.T) {
	source := "#package('example.com/read')\n#define($_ = $Records<string>(query/records))\n#define($_ = $Records<[]*Record>(output/view).WithTag('json:\"records\"'))\nSELECT record.*, type(record,'Record') FROM (SELECT 1 AS id) record"
	op := readerbuilder.Operation{Type: "updateOutputField", Field: &readerbuilder.Field{ExistingName: "Records", Name: "Items", Type: "[]*Record", SourceKind: "output", SourceName: "view"}}
	updated, err := editOutputField(source, op)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"$Records<string>(query/records)", "$Items<[]*Record>(output/view).WithTag('json:\"records\"')"} {
		if !strings.Contains(updated, want) {
			t.Fatalf("missing %s in %s", want, updated)
		}
	}
	removed, err := editOutputField(updated, readerbuilder.Operation{Type: "removeOutputField", Field: &readerbuilder.Field{ExistingName: "Items"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(removed, "$Items") || !strings.Contains(removed, "$Records<string>") {
		t.Fatal(removed)
	}
}

func TestOutputFieldCompilesCandidateBeforeAccepting(t *testing.T) {
	service := readerbuilder.New(readerbuilder.Config{Scope: "example.com/read", Name: "reader"})
	source := "#package('example.com/read')\n#setting($_ = $connector('main'))\n#setting($_ = $route('/items','GET'))\n#define($_ = $Records<[]*Record>(output/view))\nSELECT record.*, type(record,'Record') FROM (SELECT 1 AS id) record"
	op := readerbuilder.Operation{Type: "updateOutputField", Field: &readerbuilder.Field{ExistingName: "Records", Name: "Items", Type: "[]*Record", SourceKind: "output", SourceName: "view"}}
	result := applyOutputField(context.Background(), service, source, op)
	if !result.Applied {
		t.Fatalf("rename rejected: %+v", result.Diagnostics)
	}
	if !strings.Contains(result.DQL, "$Items<[]*Record>(output/view)") {
		t.Fatal(result.DQL)
	}
	op.Field.Type = "[]*"
	rejected := applyOutputField(context.Background(), service, source, op)
	if rejected.Applied || rejected.DQL != source || len(rejected.Diagnostics) == 0 {
		t.Fatalf("invalid output accepted: %+v", rejected)
	}
}

func TestOutputFieldRejectsWrongDirectionAndInjectedHead(t *testing.T) {
	source := "#define($_ = $ID<int>(query/id))\n#define($_ = $Records<[]*Record>(output/view))\nSELECT 1"
	for _, field := range []*readerbuilder.Field{
		{ExistingName: "ID", Name: "Changed", Type: "int", SourceKind: "output", SourceName: "view"},
		{ExistingName: "Records", Name: "Changed", Type: "int", SourceKind: "query", SourceName: "view"},
		{ExistingName: "Records", Name: "Changed", Type: "int", SourceKind: "output", SourceName: "view).Required("},
	} {
		if _, err := editOutputField(source, readerbuilder.Operation{Type: "updateOutputField", Field: field}); err == nil {
			t.Fatalf("accepted %+v", field)
		}
	}
}

func TestOutputFieldEditAcceptsQualifiedMetricsType(t *testing.T) {
	service := readerbuilder.New(readerbuilder.Config{Scope: "example.com/read", Name: "reader"})
	source := "#package('example.com/read')\n#setting($_ = $connector('main'))\n#setting($_ = $route('/items','GET'))\n#define($_ = $Records<[]*Record>(output/view))\n#define($_ = $Metrics<github.com/viant/xdatly/response.Metrics>(output/metrics))\nSELECT record.*, type(record,'Record') FROM (SELECT 1 AS id) record"
	operation := readerbuilder.Operation{Type: "updateOutputField", Field: &readerbuilder.Field{ExistingName: "Metrics", Name: "ExecutionMetrics", Type: "github.com/viant/xdatly/response.Metrics", SourceKind: "output", SourceName: "metrics"}}
	result := applyOutputField(context.Background(), service, source, operation)
	if !result.Applied || !strings.Contains(result.DQL, "$ExecutionMetrics<github.com/viant/xdatly/response.Metrics>(output/metrics)") {
		t.Fatalf("qualified metrics edit rejected: %+v", result)
	}
}
