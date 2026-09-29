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
