package runtimevalidation

import (
	"errors"
	"fmt"
	"strings"

	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/transcribe/column"
)

// contractDiagnostic recognizes native operation boundaries, not arbitrary
// driver text. Native errors can contain evaluated SQL, bound values, URLs and
// credentials. Nothing from the matched error is copied into the diagnostic.
// Unknown errors deliberately retain the safe contract-level fallback.
func contractDiagnostic(err error) (sdk.Diagnostic, bool) {
	var native *column.DiscoveryError
	if errors.As(err, &native) {
		if d, ok := typedDiscoveryDiagnostic(native.Diagnostic()); ok {
			return d, true
		}
	}
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

func typedDiscoveryDiagnostic(native column.DiscoveryDiagnostic) (sdk.Diagnostic, bool) {
	if native.Operation != "query" {
		return sdk.Diagnostic{}, false
	}
	d := sdk.Diagnostic{Severity: "error", Code: "runtime_discovery_query", Stage: "contract.discovery", Field: "authoredDql",
		Message: "Column discovery could not query the configured source.",
		Hint:    "Verify connector access, source existence, SQL dialect, and expression types with a bounded source-schema probe. Runtime discovery queries may differ from the authored SQL."}
	if validDiagnosticView(native.View) {
		d.Message = fmt.Sprintf("Column discovery could not query view %q.", native.View)
	}
	switch native.Reason {
	case "query_invalid":
		d.Code, d.Hint = "runtime_discovery_query_invalid", "The source reports an invalid query. Check the SQL dialect and expression types in this view using a bounded source-schema probe."
	case "source_not_found":
		d.Code, d.Hint = "runtime_discovery_source_not_found", "The source reports a missing relation. Verify the view's source and connector catalog/schema selection."
	case "column_not_found":
		d.Code, d.Hint = "runtime_discovery_column_not_found", "The source reports a missing column. Compare the view's projection and references with the native source schema."
	case "access_denied":
		d.Code, d.Hint = "runtime_discovery_access_denied", "The source denied access. Verify the configured connector's existing credentials and source permissions."
	case "source_unavailable":
		d.Code, d.Hint = "runtime_discovery_source_unavailable", "The source reports a backend failure. Check source readiness before validating again."
	case "rate_limited":
		d.Code, d.Hint = "runtime_discovery_rate_limited", "The source reports rate limiting. Validate again after the source permits another request."
	case "quota_exceeded":
		d.Code, d.Hint = "runtime_discovery_quota_exceeded", "The source reports an exhausted quota. Check the configured source quota before validating again."
	}
	switch native.DriverCode {
	case "mysql:1045", "mysql:1054", "mysql:1064", "mysql:1146", "googleapi:400", "googleapi:401", "googleapi:403", "googleapi:404", "googleapi:408", "googleapi:409", "googleapi:429", "googleapi:500", "googleapi:502", "googleapi:503", "googleapi:504":
		d.Hint += " Native source code: " + native.DriverCode + "."
	}
	return d, true
}

func validDiagnosticView(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for i, c := range []byte(value) {
		if c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && c >= '0' && c <= '9' {
			continue
		}
		return false
	}
	return true
}
