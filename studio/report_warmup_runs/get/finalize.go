package get

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/viant/datly-studio/sdk"
	xresponse "github.com/viant/xdatly/response"
)

func (output *WarmupGetOutput) Finalize(ctx context.Context) error {
	input, ok := ctx.Value(reflect.TypeFor[*WarmupGetInput]()).(*WarmupGetInput)
	if !ok || input == nil {
		return fmt.Errorf("warmup get bound input is unavailable")
	}
	if output == nil || output.Item == nil {
		return &xresponse.Error{Code: 404, Cause: errors.New("warmup run not found")}
	}
	row := output.Item
	if row.ReportId != input.ReportId || row.RunId != input.RunId || row.VersionNo <= 0 {
		return fmt.Errorf("warmup reader returned a mismatched row")
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
			return fmt.Errorf("decode warmup target: %w", err)
		}
	}
	if len(row.DiagnosticsJson) != 0 {
		if err := json.Unmarshal(row.DiagnosticsJson, &run.Diagnostics); err != nil {
			return fmt.Errorf("decode warmup diagnostics: %w", err)
		}
	}
	output.Response = run
	return nil
}
