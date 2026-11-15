package readerinspection

import "github.com/viant/datly-studio/sdk"

// SetEffectiveWarmup accepts only metadata for the inspected source snapshot.
// Callers must first authorize through the existing warmup execution boundary.
func SetEffectiveWarmup(inspection *sdk.ReaderInspection, metadata *sdk.EffectiveWarmupInspection) {
	v := inspection.Version
	if metadata != nil && v != nil && metadata.Status == "resolved" && metadata.Targets != nil && metadata.ReportID == v.ReportID && metadata.VersionNo == v.VersionNo && metadata.SourceRevision == v.SourceRevision {
		inspection.EffectiveWarmup = metadata
		return
	}
	if v != nil {
		inspection.EffectiveWarmup = &sdk.EffectiveWarmupInspection{Status: "unavailable", ReportID: v.ReportID, VersionNo: v.VersionNo, SourceRevision: v.SourceRevision}
	}
	// Safe even without source access. Never forward compilation or cache errors.
	inspection.Diagnostics = append(inspection.Diagnostics, sdk.Diagnostic{Severity: "warning", Code: "warmup_inspection_unavailable", Message: "Effective warmup plans are unavailable. Check the reader, cache, and host policy configuration."})
}
