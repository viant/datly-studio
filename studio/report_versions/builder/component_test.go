package builder

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/viant/datly-studio/sdk"
	xresponse "github.com/viant/xdatly/response"
)

func TestPublicBuilderConflictPreservesConstraint(t *testing.T) {
	for _, tc := range []struct{ field, message string }{
		{"sourceRevision", "version source revision does not match"},
		{"mcp.name", `MCP tool name "owner.records" is already used by another reader`},
		{"state", "published version cannot be mutated"},
		{"etag", "report etag does not match"},
		{"", "reader version changed during save"},
		{"", ""},
	} {
		t.Run(tc.field+tc.message, func(t *testing.T) {
			input := &sdk.Error{Code: sdk.ErrorConflict, Field: tc.field, Message: tc.message, Cause: errors.New("PRIVATE_CAUSE_CANARY")}
			var public *xresponse.Error
			if !errors.As(publicBuilderError(fmt.Errorf("PRIVATE_WRAPPER_CANARY: %w", input)), &public) || public.Code != 409 {
				t.Fatalf("public conflict=%+v", public)
			}
			raw, err := json.Marshal(public.Payload)
			if err != nil {
				t.Fatal(err)
			}
			var payload struct{ Code, Field, Message string }
			if err := json.Unmarshal(raw, &payload); err != nil {
				t.Fatal(err)
			}
			message := tc.message
			if message == "" {
				message = "reader builder conflict"
			}
			if payload.Code != "conflict" || payload.Field != tc.field || payload.Message != message || public.Cause.Error() != message {
				t.Fatalf("public payload=%s cause=%v", raw, public.Cause)
			}
			if strings.Contains(string(raw), "PRIVATE_") || strings.Contains(public.Error(), "PRIVATE_") {
				t.Fatal("private cause leaked")
			}
		})
	}
	for _, input := range []error{errors.New("PRIVATE_CAUSE_CANARY"), &sdk.Error{Code: sdk.ErrorInternal, Message: "PRIVATE_MESSAGE_CANARY", Cause: errors.New("PRIVATE_CAUSE_CANARY")}} {
		var public *xresponse.Error
		if !errors.As(publicBuilderError(input), &public) || public.Code != 502 || public.Cause.Error() != "reader builder failed" {
			t.Fatalf("internal error projection=%+v", public)
		}
	}
}
