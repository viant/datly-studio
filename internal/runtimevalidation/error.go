// Package runtimevalidation projects runtime failures without exposing source or secrets.
package runtimevalidation

import (
	"context"
	"errors"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/transcribe"
)

type Error struct {
	Stage string
	Code  string
	Field string
	Cause error `json:"-"`
}

func (e *Error) Error() string             { return "reader runtime validation failed at " + e.Stage }
func (e *Error) Unwrap() error             { return e.Cause }
func Wrap(stage string, cause error) error { return &Error{Stage: stage, Cause: cause} }

// Diagnostics only exports allowlisted static text. The cause remains available
// to trusted callers through errors.Is/As, never through serialized diagnostics.
func Diagnostics(err error) []sdk.Diagnostic {
	var compiled *transcribe.CompileError
	if errors.As(err, &compiled) && len(compiled.Diagnostics) > 0 {
		var result []sdk.Diagnostic
		for _, d := range compiled.Diagnostics {
			if d != nil {
				result = append(result, sdk.Diagnostic{Severity: string(d.Severity), Code: d.Code, Message: d.Message, Hint: d.Hint, Line: d.Span.Start.Line, Column: d.Span.Start.Char})
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	d := sdk.Diagnostic{Severity: "error", Code: "runtime_contract", Message: "reader runtime contract could not be initialized", Hint: "Check the reader configuration and validate again."}
	var phase *Error
	if errors.As(err, &phase) {
		switch phase.Stage {
		case "definition":
			d.Stage, d.Field, d.Hint = "definition", "reportId", "Check that this reader version and its dependencies are available."
		case "connector":
			d.Stage, d.Field, d.Hint = "connector", "defaultConnectorName", "Check the active connector configuration and credentials."
		case "contract":
			d.Stage, d.Field, d.Hint = "contract", "authoredDql", "Check the reader package, linked types, and source contract."
		case "artifact":
			d.Stage, d.Field, d.Hint = "artifact", "componentSpec", "Check the reader contract and linked runtime capabilities."
		case "reader":
			d.Stage, d.Field, d.Hint = "reader", "componentSpec", "Check reader execution wiring and configured caches."
		case "resources":
			d.Stage, d.Field, d.Hint = "resources", "resources", "Check the reader's MCP resources and skills."
		}
		if d.Stage != "" {
			d.Code = "runtime_" + d.Stage
		}

		if phase.Code == "runtime_package_authority" {
			d.Code = phase.Code
			d.Stage = "contract"
			d.Field = "componentScope"
			d.Message = "Reader package does not belong to the host authoring module."
			d.Hint = "Use the reader package operation to restore the host-owned package, then validate again."
		} else if phase.Stage == "contract" {
			if native, ok := contractDiagnostic(phase.Cause); ok {
				d = native
			}
		}
	}
	if errors.Is(err, context.Canceled) {
		d.Code = "validation_canceled"
		d.Message = "Reader validation was canceled."
		d.Hint = "Validate again when ready."
	}
	if errors.Is(err, context.DeadlineExceeded) {
		d.Code = "validation_deadline"
		d.Message = "Reader validation exceeded its deadline."
		d.Hint = "Check reader dependencies and validate again."
	}
	return []sdk.Diagnostic{d}
}
