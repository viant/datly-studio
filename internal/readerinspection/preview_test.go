package readerinspection

import (
	"encoding/json"
	"github.com/viant/datly-studio/sdk"
	"testing"
)

func TestPreviewInputRequiresMatchingInspectionSnapshot(t *testing.T) {
	for _, test := range []struct {
		name      string
		report    string
		version   int
		revision  int64
		available bool
	}{
		{"matching", "reader", 1, 7, true}, {"other reader", "other", 1, 7, false}, {"other version", "reader", 2, 7, false}, {"stale revision", "reader", 1, 8, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := &sdk.ReaderInspection{Version: &sdk.ReportVersion{ReportID: "reader", VersionNo: 1, SourceRevision: 7}}
			SetPreviewInput(result, &sdk.PreviewInputContract{ReportID: test.report, VersionNo: test.version, SourceRevision: test.revision, Method: "GET", Path: "/reader", Schema: json.RawMessage(`{"type":"object","properties":{}}`)})
			if (result.PreviewInput != nil) != test.available {
				t.Fatalf("snapshot mismatch accepted: %+v", result)
			}
			if !test.available && (len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "preview_input_unavailable") {
				t.Fatalf("missing diagnostic: %+v", result)
			}
		})
	}
}
