package readerinspection

import "github.com/viant/datly-studio/sdk"

// SetPreviewInput attaches only metadata compiled for this inspection snapshot.
// Callers authorize with their existing exact-version preview path first.
func SetPreviewInput(inspection *sdk.ReaderInspection, input *sdk.PreviewInputContract) {
	v := inspection.Version
	if input != nil && v != nil && input.ReportID == v.ReportID && input.VersionNo == v.VersionNo && input.SourceRevision == v.SourceRevision {
		inspection.PreviewInput = input
		return
	}
	// This static diagnostic is safe independently of DQL visibility. Never put
	// compiler errors, SQL, source spans, or connector details into this message.
	inspection.Diagnostics = append(inspection.Diagnostics, sdk.Diagnostic{Severity: "warning", Code: "preview_input_unavailable", Message: "Preview parameter metadata is unavailable. Refresh the reader; if the problem persists, validate its source and connector configuration."})
}
