package sqltransport

import (
	"encoding/json"
	"github.com/viant/datly/transcribe/dql"
	"testing"
)

func TestInputErrorOptionsRoundTrip(t *testing.T) {
	source := `#set($_ = $ID<int>(query/id).Required().WithPredicate(1,'in','c','id'))
#set($_ = $Rows<?>(output/view))
SELECT 1 AS id`
	for _, step := range []struct {
		raw     string
		code    int
		message string
	}{
		{`{"type":"addField","field":{"name":"ID","errorStatusCode":422,"errorMessage":"bad 'ID' /* detail */"}}`, 422, "bad 'ID' /* detail */"},
		{`{"type":"updateField","field":{"name":"ID","errorStatusCode":401,"errorMessage":"Unauthorized"}}`, 401, "Unauthorized"},
		{`{"type":"updateField","field":{"name":"ID","errorStatusCode":0,"errorMessage":""}}`, 0, ""},
	} {
		var err error
		source, err = editInputErrors(source, json.RawMessage(step.raw))
		if err != nil {
			t.Fatal(err)
		}
		decl, err := dql.DeclarationOccurrences(source)
		if err != nil {
			t.Fatal(err)
		}
		p := decl[0].Parameter
		if p.ErrorStatusCode != step.code || p.ErrorMessage != step.message {
			t.Fatalf("error options lost: %+v", p)
		}
		if len(decl[0].Options) < 2 {
			t.Fatal("required or predicate removed")
		}
	}
	if _, err := editInputErrors(source, json.RawMessage(`{"type":"updateField","field":{"name":"ID","errorStatusCode":200}}`)); err == nil {
		t.Fatal("accepted success status")
	}
}
