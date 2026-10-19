package sqltransport

import (
	"context"
	"strings"
	"testing"

	"github.com/viant/datly/authoring/readerbuilder"
)

func TestWholeValueInputPreservesEmptyLocationAndOptions(t *testing.T) {
	source := "#package('example.com/read')\n#setting($_ = $connector('main'))\n#setting($_ = $route('/items','GET'))\n#define($_ = $Payload<string>(body/).Optional().WithDescription('old'))\n#define($_ = $Records<[]*Record>(output/view))\nSELECT record.*, type(record,'Record') FROM (SELECT 1 AS id, '$Payload' AS literal) record"
	required := false
	description := "New body"
	field := &readerbuilder.Field{ExistingName: "Payload", Name: "RequestBody", Type: "string", SourceKind: "body", SourceName: "", Required: &required, Description: &description, UpdateDescription: true}
	operation := readerbuilder.Operation{Type: readerbuilder.OperationUpdateField, Field: field}
	if !wholeValueInputEdit(operation) {
		t.Fatal("whole body update not recognized")
	}
	candidate, err := editWholeValueInput(source, field)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"$RequestBody<string>(body/).Optional()", `.WithDescription("New body")`, "'$Payload'"} {
		if !strings.Contains(candidate, want) {
			t.Fatalf("missing %s in %s", want, candidate)
		}
	}
	service := readerbuilder.New(readerbuilder.Config{Scope: "example.com/read", Name: "reader"})
	result := compileFieldEdit(context.Background(), service, source, candidate, nil)
	if !result.Applied {
		t.Fatalf("whole body update rejected: %+v", result.Diagnostics)
	}
}
