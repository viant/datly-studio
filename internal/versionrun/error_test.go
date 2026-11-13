package versionrun

import (
	"errors"
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
