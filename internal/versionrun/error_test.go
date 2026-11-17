package versionrun

import (
	"context"
	"errors"
	"fmt"
	"github.com/viant/datly-studio/sdk"
	xresponse "github.com/viant/xdatly/response"
	"strings"
	"testing"
)

func TestVersionExecutionDenialDoesNotMisidentifyComponentRunGuard(t *testing.T) {
	err := PublicError(&sdk.Error{Code: sdk.ErrorForbidden, Message: "internal policy/connection details", Cause: errors.New("private diagnostic")}, "reader preview")
	var response *xresponse.Error
	if !errors.As(err, &response) || response.Code != 403 {
		t.Fatalf("wrong public category: %v", err)
	}
	message := response.Payload.(map[string]string)["message"]
	if message != "reader preview is not permitted for this version" {
		t.Fatalf("wrong execution-stage message: %v", message)
	}
	if strings.Contains(err.Error(), "private diagnostic") || strings.Contains(err.Error(), "internal policy") {
		t.Fatal("internal cause exposed")
	}
}

func TestPublicErrorPreservesSafeAuthoringDiagnostics(t *testing.T) {
	original := &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "private source", Diagnostics: []sdk.Diagnostic{{Code: "cube_source_not_eligible", Field: "authoredDql", Message: "Use a grouped read-only source reader."}}}
	public := PublicError(original, "reader preview")
	var response *xresponse.Error
	if !errors.As(public, &response) || response.Code != 400 || !errors.Is(public, original) {
		t.Fatalf("lost error category or identity: %v", public)
	}
	payload := response.Payload.(map[string]any)
	diagnostics := payload["diagnostics"].([]sdk.Diagnostic)
	if len(diagnostics) != 1 || diagnostics[0].Code != "cube_source_not_eligible" || strings.Contains(public.Error(), "private source") {
		t.Fatalf("unsafe or missing diagnostic: %v", public)
	}
}

func TestPublicErrorPreservesWrappedNativeCategory(t *testing.T) {
	for _, code := range []int{400, 401, 403, 404, 409, 503} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			native := &xresponse.Error{Code: code, Payload: map[string]string{"message": "PRIVATE_CANARY"}}
			wrapped := &sdk.Error{Code: sdk.ErrorInternal, Message: "INTERNAL_CANARY", Cause: fmt.Errorf("runtime: %w", native)}
			public := ExecutionError(context.Background(), wrapped, "reader preview", "synthetic", 1)
			var result *xresponse.Error
			if !errors.As(public, &result) || result.Code != code || !errors.Is(public, native) {
				t.Fatalf("lost native category or cause: %v", public)
			}
			if strings.Contains(public.Error(), "CANARY") {
				t.Fatal("private payload exposed")
			}
		})
	}
	native := &xresponse.Error{Code: 400, Cause: context.Canceled}
	public := PublicError(&sdk.Error{Code: sdk.ErrorInternal, Cause: native}, "reader preview")
	var result *xresponse.Error
	if !errors.As(public, &result) || result.Code != 502 || !errors.Is(public, context.Canceled) {
		t.Fatal("cancellation misclassified")
	}
}
