package runtimevalidation

import (
	"errors"
	"strings"

	"github.com/viant/datly-studio/sdk"
)

// contractDiagnostic recognizes native operation boundaries, not arbitrary
// driver text. Native errors can contain evaluated SQL, bound values, URLs and
// credentials. Nothing from the matched error is copied into the diagnostic.
// Unknown errors deliberately retain the safe contract-level fallback.
func contractDiagnostic(err error) (sdk.Diagnostic, bool) {
	for cause := err; cause != nil; cause = errors.Unwrap(cause) {
		message := cause.Error()
		var code, stage, field, summary, hint string
		switch {
		case strings.HasPrefix(message, "materialize input marker: "):
			code, stage, field = "runtime_input_marker", "contract.input", "authoredDql"
			summary = "The input presence marker could not be materialized."
			hint = "Check input field names and their generated presence-marker fields for conflicts."
		case strings.HasPrefix(message, "materialize input contract: "):
			code, stage, field = "runtime_input_type", "contract.input", "authoredDql"
			summary = "The input runtime type could not be materialized."
			hint = "Check declared input types, linked type availability, and generated field names."
		case strings.HasPrefix(message, "materialize output contract: "):
			code, stage, field = "runtime_output_type", "contract.output", "authoredDql"
			summary = "The output runtime type could not be materialized."
			hint = "Check output and view types, linked type availability, and generated field names."
		case strings.HasPrefix(message, "evaluate schema-discovery source: ") || strings.HasPrefix(message, "re-evaluate named identity source: "):
			code, stage, field = "runtime_discovery_template", "contract.discovery", "authoredDql"
			summary = "The source template could not be evaluated for column discovery."
			hint = "Check template input declarations and operations against their compile-time types; column discovery has no request values."
		case strings.HasPrefix(message, "bind discovery SQL: "):
			code, stage, field = "runtime_discovery_bind", "contract.discovery", "authoredDql"
			summary = "The column-discovery query could not bind its parameters."
			hint = "Check source parameter declarations and placeholders, including parameters used by independent views."
		case strings.HasPrefix(message, "build schema-discovery query: "):
			code, stage, field = "runtime_discovery_query_shape", "contract.discovery", "authoredDql"
			summary = "The source could not be converted into a column-discovery query."
			hint = "Check the source projection and SQL structure using the configured connector dialect."
		case strings.HasPrefix(message, "SQLX discovery failed for "):
			code, stage, field = "runtime_discovery_query", "contract.discovery", "authoredDql"
			summary = "The configured source rejected the column-discovery query."
			hint = "Verify connector access, source existence, SQL dialect, and expression types with a bounded source-schema probe. Runtime discovery queries may differ from the authored SQL."
		case strings.HasPrefix(message, "resolve SQL dialect: "):
			code, stage, field = "runtime_discovery_dialect", "contract.discovery", "defaultConnectorName"
			summary = "The connector SQL dialect could not be resolved."
			hint = "Check that the connector driver and native source metadata are available."
		default:
			continue
		}
		return sdk.Diagnostic{Severity: "error", Code: code, Stage: stage, Field: field, Message: summary, Hint: hint}, true
	}
	return sdk.Diagnostic{}, false
}
