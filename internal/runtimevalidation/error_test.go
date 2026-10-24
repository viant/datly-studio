package runtimevalidation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/viant/datly-studio/internal/readerinspection"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/transcribe"
	"strings"
	"testing"
)

func TestSafeRuntimeDiagnosticsPreserveCauseAndAuthority(t *testing.T) {
	secret := errors.New("private DSN https://secret:user@private.invalid/source.sql")
	for _, stage := range []string{"definition", "connector", "contract", "artifact", "reader", "resources", "untrusted stage with secret"} {
		err := Wrap(stage, secret)
		if !errors.Is(err, secret) {
			t.Fatal("lost cause")
		}
		var typed *Error
		if !errors.As(err, &typed) {
			t.Fatal("lost typed boundary")
		}
		for _, allowed := range []bool{true, false} {
			data, _ := json.Marshal(readerinspection.RedactDiagnostics(Diagnostics(err), allowed))
			if strings.Contains(string(data), "private") || strings.Contains(string(data), "secret") || strings.Contains(string(data), "untrusted") {
				t.Fatalf("unsafe diagnostic: %s", data)
			}
		}
	}
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded, &sdk.Error{Code: sdk.ErrorForbidden, Message: secret.Error()}} {
		err := Wrap("definition", cause)
		if !errors.Is(err, cause) {
			t.Fatal("lost cancellation/authorization cause")
		}
		data, _ := json.Marshal(Diagnostics(err))
		if strings.Contains(string(data), "private") {
			t.Fatalf("leak %s", data)
		}
	}
	d := Diagnostics(&Error{Stage: "contract", Code: "runtime_package_authority", Cause: secret})[0]
	if d.Code != "runtime_package_authority" || d.Field != "componentScope" || d.Stage != "contract" || d.Hint == "" {
		t.Fatalf("diagnostic %+v", d)
	}
}
func TestCompileSpansAndACLProjectionRemainIntact(t *testing.T) {
	cause := errors.New("parser cause")
	err := Wrap("contract", fmt.Errorf("wrapped: %w", &transcribe.CompileError{Cause: cause, Diagnostics: []*transcribe.Diagnostic{{Code: "DQL-TYPE", Severity: transcribe.SeverityError, Message: "authored source", Hint: "repair source", Span: transcribe.Span{Start: transcribe.Position{Line: 12, Char: 7}}}}}))
	if !errors.Is(err, cause) {
		t.Fatal("lost parser cause")
	}
	ds := Diagnostics(err)
	d := ds[0]
	if d.Code != "DQL-TYPE" || d.Line != 12 || d.Column != 7 || d.Message != "authored source" {
		t.Fatalf("span %+v", d)
	}
	d = readerinspection.RedactDiagnostics(ds, false)[0]
	if d.Message != "" || d.Hint != "" || d.Code != "DQL-TYPE" || d.Line != 12 {
		t.Fatalf("ACL projection %+v", d)
	}
}
