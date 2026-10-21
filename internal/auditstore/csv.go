package auditstore

import (
	"context"
	"github.com/viant/datly/output"
	"reflect"
)

// CSVEvent is the flat metadata projection passed to Datly's native CSV codec.
type CSVEvent struct {
	ID                int64  `csvName:"id"`
	NamespaceID       string `csvName:"namespace_id"`
	ComponentID       string `csvName:"component_id"`
	ToolName          string `csvName:"tool_name"`
	StartedAt         string `csvName:"started_at"`
	DurationMS        int64  `csvName:"duration_ms"`
	Status            string `csvName:"status"`
	Subject           string `csvName:"subject"`
	ClientIP          string `csvName:"client_ip"`
	ResponseTruncated bool   `csvName:"response_truncated"`
}

// EncodeCSV uses the same output plan as Datly's _format=csv / Accept selector.
// No CSV quoting, headers, delimiters or row serialization live in Studio.
func EncodeCSV(ctx context.Context, rows []CSVEvent) ([]byte, error) {
	plan, err := (output.Compiler{}).Compile(output.CompileInput{Type: reflect.TypeFor[[]CSVEvent]()})
	if err != nil {
		return nil, err
	}
	encoded, err := plan.Encode(ctx, "csv", rows)
	return encoded.Data, err
}
