package sqltransport

import (
	"context"
	"errors"
	"fmt"

	"github.com/viant/datly-studio/internal/runtimeadmin"
	"github.com/viant/datly-studio/internal/runtimevalidation"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/transcribe"
)

// Publication failures retain their trusted cause internally. Public messages
// and diagnostics use only fixed text and typed source locations, never SQL,
// definitions, connector values or an arbitrary underlying Error string.
func publicationRuntimeError(stage string, cause error) *sdk.Error {
	diagnostic := sdk.Diagnostic{Severity: "error", Stage: stage, Field: "runtime", Code: "publication_runtime_reload", Message: "The selected namespace runtime could not rebuild the validated component.", Hint: "Check the selected namespace runtime and component validation; correct its dependencies before retrying Release."}
	message := "dynamic runtime reload failed"
	if stage == "runtime_activation" {
		message = "runtime activation persistence failed"
		diagnostic.Code = "publication_runtime_activation"
		diagnostic.Message = "Runtime activation could not be recorded."
		diagnostic.Hint = "Inspect the current publication state before retrying Release; do not assume the candidate became active."
	}
	var phase *runtimevalidation.Error
	var compiled *transcribe.CompileError
	var projected *sdk.Error
	if errors.As(cause, &projected) && len(projected.Diagnostics) > 0 {
		// These diagnostics have already crossed the SDK's public projection
		// boundary. Preserve their field, span and repair instead of rebuilding
		// them from the error string or discarding useful compiler feedback.
		diagnostics := PublicationReceiptDiagnostics(projected.Diagnostics)
		field := projected.Field
		if field == "" {
			field = diagnostics[0].Field
		}
		return &sdk.Error{Code: sdk.ErrorUnavailable, Field: field, Message: message + ": " + diagnostics[0].Message, Diagnostics: diagnostics, Cause: cause}
	}
	switch {
	case errors.Is(cause, context.Canceled):
		diagnostic.Code = "publication_canceled"
		diagnostic.Message = "The publication transition was canceled."
		diagnostic.Hint = "Inspect the current publication state before retrying the same component version."
	case errors.Is(cause, context.DeadlineExceeded):
		diagnostic.Code = "publication_deadline"
		diagnostic.Message = "The publication transition exceeded its deadline."
		diagnostic.Hint = "Check the namespace runtime and component dependencies, then inspect publication state before retrying."
	case errors.Is(cause, runtimeadmin.ErrNamespaceUnavailable):
		diagnostic.Code = "publication_namespace_unavailable"
		diagnostic.Field = "namespaceId"
		diagnostic.Message = "The selected namespace has no available MCP runtime."
		diagnostic.Hint = "Open Namespaces, enable MCP for the selected namespace, and save. If already enabled, check its runtime status before retrying Release."
	case errors.As(cause, &compiled):
		diagnostic.Code = "publication_runtime_compile"
		diagnostic.Field = "componentSpec"
		diagnostic.Message = "The reader runtime contract could not compile."
		diagnostic.Hint = "Validate this component version and correct the reader source or linked types before retrying Release."
		// Raw compiler text is not an already ACL-projected SDK diagnostic.
		// Retain its stable code/location without exposing authored source.
		for _, item := range compiled.Diagnostics {
			if item != nil {
				if item.Code != "" {
					diagnostic.Code = boundedPublicationText(item.Code)
				}
				diagnostic.Line = item.Span.Start.Line
				diagnostic.Column = item.Span.Start.Char
				break
			}
		}
	case errors.As(cause, &phase):
		if phase.Code == "runtime_package_authority" {
			diagnostic.Code = "runtime_package_authority"
			diagnostic.Field = "componentScope"
			diagnostic.Message = "Reader package does not belong to the host authoring module."
			diagnostic.Hint = "Restore the host-owned reader package, validate this version, then retry Release."
		} else if field, ok := map[string]string{"definition": "reportId", "connector": "defaultConnectorName", "contract": "authoredDql", "artifact": "componentSpec", "reader": "componentSpec", "resources": "resources"}[phase.Stage]; ok {
			diagnostic.Code = "publication_runtime_" + phase.Stage
			diagnostic.Field = field
			diagnostic.Message = "A reader runtime dependency or contract could not be initialized."
			diagnostic.Hint = "Check the indicated reader field and its configured dependencies, validate this version, then retry Release."
		}
	}
	return &sdk.Error{Code: sdk.ErrorUnavailable, Field: diagnostic.Field, Message: message + ": " + diagnostic.Message, Diagnostics: []sdk.Diagnostic{diagnostic}, Cause: cause}
}

// PublicationReceiptDiagnostics bounds already-public SDK feedback for the
// publication transport and its typed mutation endpoint. It never projects raw
// error strings or changes the source diagnostic slice.
func PublicationReceiptDiagnostics(source []sdk.Diagnostic) []sdk.Diagnostic {
	const limit = 16
	count := len(source)
	alreadyBounded := count == limit+1 && source[limit].Code == "publication_diagnostics_omitted"
	if count > limit && !alreadyBounded {
		count = limit
	}
	result := append([]sdk.Diagnostic(nil), source[:count]...)
	for i := range result {
		d := &result[i]
		d.Code, d.Stage, d.Field = boundedPublicationText(d.Code), boundedPublicationText(d.Stage), boundedPublicationText(d.Field)
		d.Message, d.Hint = boundedPublicationText(d.Message), boundedPublicationText(d.Hint)
	}
	if len(source) > count {
		result = append(result, sdk.Diagnostic{Severity: "info", Code: "publication_diagnostics_omitted", Message: fmt.Sprintf("%d additional diagnostics are omitted from this receipt.", len(source)-count), Hint: "Open the component validation details for the complete diagnostics."})
	}
	return result
}
