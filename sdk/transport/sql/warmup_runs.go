package sqltransport

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/viant/datly-studio/sdk"
)

const warmupRunTimeout = 5 * time.Minute

type warmupRunGetRequest struct {
	ReportID string `json:"reportId"`
	RunID    string `json:"runId"`
}

type warmupRunListRequest struct {
	ReportID  string                  `json:"reportId"`
	VersionNo int                     `json:"versionNo"`
	Input     sdk.ListWarmupRunsInput `json:"input"`
}

func (t *Transport) startWarmupRun(ctx context.Context, in versionIdentityRequest, output any) error {
	version, err := t.getVersionValue(ctx, in.ReportID, in.VersionNo)
	if err != nil {
		return err
	}
	requestedBy := "system"
	if principal, ok := sdk.PrincipalFromContext(ctx); ok && strings.TrimSpace(principal.Subject) != "" {
		requestedBy = principal.Subject
	}
	planKey := warmupPlanKey(version)
	activeKey := fmt.Sprintf("%s:%d:%s", in.ReportID, in.VersionNo, planKey)
	now := t.now()
	t.warmupMu.Lock()
	defer t.warmupMu.Unlock()
	if err = t.recoverExpiredWarmupRuns(ctx, now); err != nil {
		return err
	}
	var existingID string
	err = t.DB.QueryRowContext(ctx, `SELECT run_id FROM report_warmup_runs WHERE active_key=?`, activeKey).Scan(&existingID)
	if err == nil {
		run, readErr := t.readWarmupRun(ctx, `WHERE report_id=? AND run_id=?`, in.ReportID, existingID)
		if readErr != nil {
			return readErr
		}
		return assign(output, run)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return internal(err)
	}
	runID, err := generatedWarmupRunID()
	if err != nil {
		return internal(err)
	}
	if _, err = t.DB.ExecContext(ctx, `INSERT INTO report_warmup_runs(run_id,report_id,version_no,source_revision,spec_hash,plan_key,active_key,status,requested_by,target_json,requested_at) VALUES(?,?,?,?,?,?,?,'accepted',?,'{}',?)`, runID, in.ReportID, in.VersionNo, version.SourceRevision, version.SpecHash, planKey, activeKey, requestedBy, now); err != nil {
		return classify(err, "warmup run", runID)
	}
	run, err := t.readWarmupRun(ctx, `WHERE report_id=? AND run_id=?`, in.ReportID, runID)
	if err != nil {
		return err
	}
	jobCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), warmupRunTimeout)
	go func() {
		defer cancel()
		t.executeWarmupRun(jobCtx, runID, in.ReportID, in.VersionNo)
	}()
	return assign(output, run)
}

func (t *Transport) executeWarmupRun(ctx context.Context, runID, reportID string, versionNo int) {
	started := t.now()
	_, _ = t.DB.ExecContext(context.WithoutCancel(ctx), `UPDATE report_warmup_runs SET status='running',started_at=? WHERE run_id=? AND status='accepted'`, started, runID)
	result, runErr := t.Warmup.Warmup(ctx, reportID, versionNo)
	completed := t.now()
	status := "completed"
	var diagnostics []sdk.Diagnostic
	if runErr != nil {
		status = "failed"
		if result != nil && (result.CompletedCases > 0 || result.Entries > 0) {
			status = "partial"
		}
		diagnostics = []sdk.Diagnostic{warmupDiagnostic(runErr)}
	}
	if result == nil {
		result = &sdk.WarmupResult{}
	}
	targetJSON, _ := json.Marshal(result.Target)
	diagnosticsJSON, _ := json.Marshal(diagnostics)
	_, _ = t.DB.ExecContext(context.WithoutCancel(ctx), `UPDATE report_warmup_runs SET status=?,active_key=NULL,cache_name=?,cache_provider=?,connector_name=?,index_column=?,planned_cases=?,completed_cases=?,max_cases=?,row_limit=?,entries=?,duration_ns=?,target_json=?,diagnostics_json=?,completed_at=? WHERE run_id=?`, status, nullable(result.Target.CacheName), nullable(result.Target.CacheProvider), nullable(result.Target.ConnectorName), nullable(result.Target.IndexColumn), result.PlannedCases, result.CompletedCases, result.MaxCases, result.RowLimit, result.Entries, int64(result.Duration), string(targetJSON), nullableJSON(diagnosticsJSON), completed, runID)
}

func warmupDiagnostic(err error) sdk.Diagnostic {
	code, message := "warmup_failed", "cache warmup failed"
	var sdkErr *sdk.Error
	if errors.As(err, &sdkErr) {
		if sdkErr.Code != "" {
			code = string(sdkErr.Code)
		}
		if sdkErr.Message != "" {
			message = sdkErr.Message
		}
	} else if errors.Is(err, context.DeadlineExceeded) {
		code, message = "warmup_timeout", "cache warmup exceeded the server-owned timeout"
	}
	return sdk.Diagnostic{Severity: "error", Code: code, Message: message}
}

func (t *Transport) getWarmupRun(ctx context.Context, input, output any) error {
	var in warmupRunGetRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if strings.TrimSpace(in.ReportID) == "" || strings.TrimSpace(in.RunID) == "" {
		return invalid(errors.New("reportId and runId are required"))
	}
	run, err := t.readWarmupRun(ctx, `WHERE report_id=? AND run_id=?`, in.ReportID, in.RunID)
	if err != nil {
		return err
	}
	return assign(output, run)
}

func (t *Transport) listWarmupRuns(ctx context.Context, input, output any) error {
	var in warmupRunListRequest
	if err := decode(input, &in); err != nil {
		return invalid(err)
	}
	if strings.TrimSpace(in.ReportID) == "" || in.VersionNo <= 0 || in.Input.Offset < 0 {
		return invalid(errors.New("reportId, versionNo, and non-negative offset are required"))
	}
	if err := t.recoverExpiredWarmupRuns(ctx, t.now()); err != nil {
		return err
	}
	limit := in.Input.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := t.DB.QueryContext(ctx, warmupRunSelect+` WHERE report_id=? AND version_no=? ORDER BY requested_at DESC,run_id DESC LIMIT ? OFFSET ?`, in.ReportID, in.VersionNo, limit, in.Input.Offset)
	if err != nil {
		return internal(err)
	}
	defer rows.Close()
	page := &sdk.WarmupRunPage{Limit: limit, Offset: in.Input.Offset}
	for rows.Next() {
		run, scanErr := scanWarmupRun(rows)
		if scanErr != nil {
			return internal(scanErr)
		}
		page.Items = append(page.Items, run)
	}
	if err = rows.Err(); err != nil {
		return internal(err)
	}
	return assign(output, page)
}

func (t *Transport) recoverExpiredWarmupRuns(ctx context.Context, now time.Time) error {
	diagnostics, _ := json.Marshal([]sdk.Diagnostic{{Severity: "error", Code: "warmup_expired", Message: "cache warmup did not reach a terminal state before the server-owned timeout"}})
	_, err := t.DB.ExecContext(ctx, `UPDATE report_warmup_runs SET status='failed',active_key=NULL,diagnostics_json=?,completed_at=? WHERE status IN ('accepted','running') AND requested_at<?`, string(diagnostics), now, now.Add(-warmupRunTimeout))
	if err != nil {
		return internal(err)
	}
	return nil
}

const warmupRunSelect = `SELECT run_id,report_id,version_no,source_revision,spec_hash,plan_key,status,requested_by,requested_at,started_at,completed_at,planned_cases,completed_cases,max_cases,row_limit,entries,duration_ns,target_json,diagnostics_json FROM report_warmup_runs`

func (t *Transport) readWarmupRun(ctx context.Context, where string, args ...any) (*sdk.WarmupRun, error) {
	run, err := scanWarmupRun(t.DB.QueryRowContext(ctx, warmupRunSelect+` `+where, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, &sdk.Error{Code: sdk.ErrorNotFound, Message: "warmup run not found"}
	}
	if err != nil {
		return nil, internal(err)
	}
	return run, nil
}

func scanWarmupRun(scanner interface{ Scan(...any) error }) (*sdk.WarmupRun, error) {
	var run sdk.WarmupRun
	var started, completed sql.NullTime
	var targetJSON, diagnosticsJSON sql.NullString
	var duration int64
	var maxCases, rowLimit sql.NullInt64
	if err := scanner.Scan(&run.RunID, &run.ReportID, &run.VersionNo, &run.SourceRevision, &run.SpecHash, &run.PlanKey, &run.Status, &run.RequestedBy, &run.RequestedAt, &started, &completed, &run.PlannedCases, &run.CompletedCases, &maxCases, &rowLimit, &run.Entries, &duration, &targetJSON, &diagnosticsJSON); err != nil {
		return nil, err
	}
	run.Duration = time.Duration(duration)
	if maxCases.Valid {
		value := int(maxCases.Int64)
		run.MaxCases = &value
	}
	if rowLimit.Valid {
		value := int(rowLimit.Int64)
		run.RowLimit = &value
	}
	if started.Valid {
		value := started.Time
		run.StartedAt = &value
	}
	if completed.Valid {
		value := completed.Time
		run.CompletedAt = &value
	}
	if targetJSON.Valid {
		_ = json.Unmarshal([]byte(targetJSON.String), &run.Target)
	}
	if diagnosticsJSON.Valid {
		_ = json.Unmarshal([]byte(diagnosticsJSON.String), &run.Diagnostics)
	}
	return &run, nil
}

func generatedWarmupRunID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return "w" + hex.EncodeToString(value), nil
}

func warmupPlanKey(version *sdk.ReportVersion) string {
	value := fmt.Sprintf("%s:%d:%d:%s", version.ReportID, version.VersionNo, version.SourceRevision, version.SpecHash)
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func nullableJSON(value []byte) any {
	if len(value) == 0 || string(value) == "null" || string(value) == "[]" {
		return nil
	}
	return string(value)
}
