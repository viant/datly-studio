package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/viant/xdatly/response"
	"testing"
)

func TestToolNamespaceValidationHasActionablePublicBody(t *testing.T) {
	_, _, err := toolRuntime(context.Background(), nil, nil, nil, "analytics")
	if err == nil {
		t.Fatal("invalid namespace accepted")
	}
	var typed *response.Error
	if !errors.As(err, &typed) || typed.StatusCode() != 400 || errors.Unwrap(typed) == nil {
		t.Fatalf("lost typed error: %v", err)
	}
	body, ok := response.ErrorBody(err)
	if !ok {
		t.Fatal("missing public body")
	}
	raw, e := json.Marshal(body)
	if e != nil {
		t.Fatal(e)
	}
	var decoded map[string]any
	if e = json.Unmarshal(raw, &decoded); e != nil {
		t.Fatal(e)
	}
	if decoded["code"] != "invalid_namespace_id" || decoded["field"] != "namespaceId" || decoded["hint"] == "" || decoded["message"] == "" {
		t.Fatalf("unactionable body: %s", raw)
	}
}
