package sqltransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/viant/datly-studio/internal/runtimeadmin"
	"github.com/viant/datly-studio/internal/runtimevalidation"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/transcribe"
)

func TestPublicationRuntimeErrorsAreActionableWithoutPrivateCauses(t *testing.T) {
	private := errors.New("SELECT secret_column FROM private_table /* DEFINITION_CANARY */ password=private-password")
	for _, tc := range []struct {
		name, stage, wantCode, wantField string
		cause                            error
	}{
		{"unknown reload", "runtime_reload", "publication_runtime_reload", "runtime", private},
		{"activation", "runtime_activation", "publication_runtime_activation", "runtime", fmt.Errorf("activation: %w", private)},
		{"namespace", "runtime_reload", "publication_namespace_unavailable", "namespaceId", runtimeadmin.ErrNamespaceUnavailable},
		{"canceled", "runtime_reload", "publication_canceled", "runtime", fmt.Errorf("private SQL context: %w", context.Canceled)},
		{"deadline", "runtime_reload", "publication_deadline", "runtime", context.DeadlineExceeded},
		{"contract", "runtime_reload", "publication_runtime_contract", "authoredDql", runtimevalidation.Wrap("contract", private)},
		{"host package", "runtime_reload", "runtime_package_authority", "componentScope", &runtimevalidation.Error{Stage: "contract", Code: "runtime_package_authority", Cause: private}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := publicationRuntimeError(tc.stage, tc.cause)
			if got.Code != sdk.ErrorUnavailable || len(got.Diagnostics) != 1 || got.Diagnostics[0].Code != tc.wantCode || got.Field != tc.wantField {
				t.Fatalf("public classification=%+v", got)
			}
			d := got.Diagnostics[0]
			if d.Stage != tc.stage || d.Field != tc.wantField || d.Hint == "" || d.Message == "" || d.Severity != "error" {
				t.Fatalf("diagnostic=%+v", d)
			}
			if !errors.Is(got, tc.cause) {
				t.Fatal("internal cause chain lost")
			}
			wire, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			for _, canary := range []string{"secret_column", "private_table", "DEFINITION_CANARY", "private-password", "private SQL context"} {
				if strings.Contains(string(wire), canary) || strings.Contains(got.Error(), canary) {
					t.Fatal("private cause leaked in public error")
				}
			}
		})
	}
}

func TestPublicationKnownDiagnosticsRetainFieldSpanAndRepair(t *testing.T) {
	private := errors.New("SELECT private_column FROM private_table password=private-password")
	projected := sdk.Diagnostic{Severity: "error", Stage: "contract", Code: "DQL-TYPE", Field: "componentSpec.views[0]", Message: "Unsupported scalar type.", Hint: "Use a supported scalar type.", Line: 12, Column: 7}
	original := &sdk.Error{Code: sdk.ErrorUnavailable, Field: projected.Field, Message: private.Error(), Diagnostics: []sdk.Diagnostic{projected}, Cause: private}
	got := publicationRuntimeError("runtime_reload", original)
	if len(got.Diagnostics) != 1 || got.Diagnostics[0] != projected || got.Field != projected.Field || !errors.Is(got, private) {
		t.Fatalf("projected feedback changed: %+v", got)
	}
	got.Diagnostics[0].Hint = "caller mutation"
	if original.Diagnostics[0].Hint != projected.Hint {
		t.Fatal("diagnostic slice aliased caller state")
	}
	compiled := &transcribe.CompileError{Cause: private, Diagnostics: []*transcribe.Diagnostic{{Code: "DQL-FIELD", Severity: transcribe.SeverityError, Message: private.Error(), Hint: private.Error(), Span: transcribe.Span{Start: transcribe.Position{Line: 18, Char: 4}}}}}
	got = publicationRuntimeError("runtime_reload", runtimevalidation.Wrap("contract", compiled))
	var preserved *transcribe.CompileError
	if !errors.As(got, &preserved) || preserved != compiled || got.Diagnostics[0].Code != "DQL-FIELD" || got.Diagnostics[0].Line != 18 || got.Diagnostics[0].Column != 4 || got.Diagnostics[0].Hint == "" {
		t.Fatalf("typed compilation location lost: %+v", got)
	}
	raw, _ := json.Marshal(got)
	if strings.Contains(string(raw), "private_column") || strings.Contains(string(raw), "private-password") {
		t.Fatal("raw unprojected compiler text leaked")
	}
}

func TestPublicationProjectedDiagnosticReceiptIsBounded(t *testing.T) {
	diagnostics := make([]sdk.Diagnostic, 20)
	for i := range diagnostics {
		diagnostics[i] = sdk.Diagnostic{Severity: "warning", Code: "KNOWN", Field: "componentSpec", Message: strings.Repeat("x", 1500), Hint: "Check the reader field.", Line: i + 1}
	}
	original := &sdk.Error{Code: sdk.ErrorUnavailable, Diagnostics: diagnostics}
	got := publicationRuntimeError("runtime_reload", original)
	again := PublicationReceiptDiagnostics(got.Diagnostics)
	if len(again) != len(got.Diagnostics) || again[len(again)-1] != got.Diagnostics[len(got.Diagnostics)-1] {
		t.Fatal("reprojection changed the original omission receipt")
	}
	if len(got.Diagnostics) != 17 || got.Diagnostics[16].Code != "publication_diagnostics_omitted" || !strings.Contains(got.Diagnostics[16].Message, "4 additional") {
		t.Fatal("bounded diagnostics lack an omission receipt")
	}
	if len([]rune(got.Diagnostics[0].Message)) > publicationEventTextLimit || got.Diagnostics[0].Hint != diagnostics[0].Hint || got.Diagnostics[0].Line != 1 || len(diagnostics[0].Message) != 1500 {
		t.Fatal("diagnostic bounds lost repair/location or changed source")
	}
}
