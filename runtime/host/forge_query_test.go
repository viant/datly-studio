package host

import (
	"encoding/json"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestPublishedStructuredInputsSurviveURLAndJSONBinding(t *testing.T) {
	query, err := publishedQuery(map[string]any{"ids": []any{json.Number("9007199254740993"), "42"}, "request": map[string]any{"limit": json.Number("50"), "offset": json.Number("0"), "selection": map[string]any{"entityId": json.Number("9007199254740993")}}, "optional": nil})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.ParseQuery(query.Encode())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parsed["ids"], []string{"9007199254740993", "42"}) {
		t.Fatalf("entity IDs changed across URL binding: %v", parsed["ids"])
	}
	decoder := json.NewDecoder(strings.NewReader(parsed.Get("request")))
	decoder.UseNumber()
	var request map[string]any
	if err = decoder.Decode(&request); err != nil {
		t.Fatalf("published JSON binding rejected structured request: %v", err)
	}
	if request["limit"] != json.Number("50") || request["selection"].(map[string]any)["entityId"] != json.Number("9007199254740993") {
		t.Fatalf("request was changed: %#v", request)
	}
	if _, exists := parsed["optional"]; exists {
		t.Fatal("missing optional input was made present")
	}
}
