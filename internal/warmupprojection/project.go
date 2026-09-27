package warmupprojection

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/viant/datly-studio/sdk"
	stored "github.com/viant/datly-studio/studio/report_warmup_runs/store_read"
)

func PlanKey(version *sdk.ReportVersion) string {
	if version == nil {
		return ""
	}
	value := fmt.Sprintf("%s:%d:%d:%s", version.ReportID, version.VersionNo, version.SourceRevision, version.SpecHash)
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

// Runs is the shared SDK projection for generic and native warmup readers.
func Runs(reportID, runID string, versionNo int, rows []*stored.StoredWarmupRun) ([]*sdk.WarmupRun, error) {
	result := make([]*sdk.WarmupRun, 0, len(rows))
	for _, row := range rows {
		if row == nil || reportID != "" && row.ReportId != reportID || runID != "" && row.RunId != runID || versionNo != 0 && row.VersionNo != versionNo {
			return nil, fmt.Errorf("warmup run reader returned a mismatched row")
		}
		run := &sdk.WarmupRun{RunID: row.RunId, ReportID: row.ReportId, VersionNo: row.VersionNo,
			SourceRevision: row.SourceRevision, SpecHash: row.SpecHash, PlanKey: row.PlanKey,
			Status: row.Status, RequestedBy: row.RequestedBy, RequestedAt: row.RequestedAt,
			CreatedAt: row.CreatedAt, CreatedBy: row.CreatedBy, UpdatedAt: row.UpdatedAt, UpdatedBy: row.UpdatedBy,
			StartedAt: row.StartedAt, CompletedAt: row.CompletedAt, PlannedCases: row.PlannedCases,
			CompletedCases: row.CompletedCases, MaxCases: row.MaxCases, RowLimit: row.RowLimit,
			Entries: row.Entries, Duration: time.Duration(row.DurationNs)}
		if len(row.TargetJson) != 0 {
			if err := json.Unmarshal(row.TargetJson, &run.Target); err != nil {
				return nil, fmt.Errorf("decode warmup target: %w", err)
			}
		}
		if len(row.DiagnosticsJson) != 0 {
			if err := json.Unmarshal(row.DiagnosticsJson, &run.Diagnostics); err != nil {
				return nil, fmt.Errorf("decode warmup diagnostics: %w", err)
			}
		}
		result = append(result, run)
	}
	return result, nil
}
