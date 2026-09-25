package sqltransport

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/sdk"
	storedreader "github.com/viant/datly-studio/studio/report_warmup_runs/store_read"
	storedwriter "github.com/viant/datly-studio/studio/report_warmup_runs/store_write"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	xhandler "github.com/viant/xdatly/handler"
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

func warmupOptionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func warmupOptionalJSON(value []byte) *json.RawMessage {
	if len(value) == 0 || string(value) == "null" || string(value) == "[]" {
		return nil
	}
	message := json.RawMessage(value)
	return &message
}

func warmupActor(ctx context.Context) sdk.Principal {
	if principal, ok := sdk.PrincipalFromContext(ctx); ok {
		return principal
	}
	return sdk.SystemPrincipal()
}

func (t *Transport) startWarmupRun(ctx context.Context, in versionIdentityRequest, output any) error {
	version, err := t.getVersionValue(ctx, in.ReportID, in.VersionNo)
	if err != nil {
		return err
	}
	requestedBy := warmupActor(ctx).Subject
	planKey := warmupPlanKey(version)
	activeKey := fmt.Sprintf("%s:%d:%s", in.ReportID, in.VersionNo, planKey)
	now := t.now()
	t.warmupMu.Lock()
	defer t.warmupMu.Unlock()
	if err = t.recoverExpiredWarmupRuns(ctx, now); err != nil {
		return err
	}
	active, err := t.readWarmupRows(ctx, &storedreader.Input{ReportId: in.ReportID, ActiveKey: activeKey, PageLimit: 2})
	if err != nil {
		return internal(err)
	}
	if len(active) > 1 {
		return internal(errors.New("warmup active-key reader returned multiple rows"))
	}
	if len(active) == 1 {
		if active[0].ReportID != in.ReportID {
			return internal(errors.New("warmup active-key reader returned a mismatched report"))
		}
		return assign(output, active[0])
	}
	runID, err := generatedWarmupRunID()
	if err != nil {
		return internal(err)
	}
	if err = t.writeWarmupRun(ctx, &storedwriter.StoredWarmupRun{RunId: runID, ReportId: in.ReportID,
		VersionNo: in.VersionNo, SourceRevision: version.SourceRevision, SpecHash: version.SpecHash,
		PlanKey: planKey, ActiveKey: &activeKey, Status: "accepted", RequestedBy: requestedBy,
		TargetJson: json.RawMessage(`{}`), RequestedAt: now, CreatedAt: &now, CreatedBy: &requestedBy,
		UpdatedAt: &now, UpdatedBy: &requestedBy,
		Has: &storedwriter.StoredWarmupRunHas{RunId: true, ReportId: true, VersionNo: true,
			SourceRevision: true, SpecHash: true, PlanKey: true, ActiveKey: true,
			Status: true, RequestedBy: true, TargetJson: true, RequestedAt: true,
			CreatedAt: true, CreatedBy: true, UpdatedAt: true, UpdatedBy: true}}); err != nil {
		return classify(err, "warmup run", runID)
	}
	run, err := t.readWarmupRun(ctx, in.ReportID, runID)
	if err != nil {
		return err
	}
	if run.UpdatedAt == nil {
		return internal(errors.New("warmup run has no updated_at token"))
	}
	jobCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), warmupRunTimeout)
	go func() {
		defer cancel()
		t.executeWarmupRun(jobCtx, runID, in.ReportID, in.VersionNo, *run.UpdatedAt)
	}()
	return assign(output, run)
}

func (t *Transport) executeWarmupRun(ctx context.Context, runID, reportID string, versionNo int, expectedUpdatedAt time.Time) {
	started := t.now()
	actor := warmupActor(ctx).Subject
	if err := t.writeWarmupRun(context.WithoutCancel(ctx), &storedwriter.StoredWarmupRun{
		RunId: runID, Status: "running", UpdatedAt: &expectedUpdatedAt, UpdatedBy: &actor, StartedAt: &started,
		Has: &storedwriter.StoredWarmupRunHas{RunId: true, Status: true, UpdatedAt: true,
			UpdatedBy: true, StartedAt: true}}); err != nil {
		return
	}
	startedRun, err := t.readWarmupRun(context.WithoutCancel(ctx), reportID, runID)
	if err != nil || startedRun.UpdatedAt == nil {
		return
	}
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
	_ = t.writeWarmupRun(context.WithoutCancel(ctx), &storedwriter.StoredWarmupRun{
		RunId: runID, Status: status, UpdatedAt: startedRun.UpdatedAt, UpdatedBy: &actor,
		CacheName: warmupOptionalString(result.Target.CacheName), CacheProvider: warmupOptionalString(result.Target.CacheProvider),
		ConnectorName: warmupOptionalString(result.Target.ConnectorName), IndexColumn: warmupOptionalString(result.Target.IndexColumn),
		PlannedCases: result.PlannedCases, CompletedCases: result.CompletedCases, MaxCases: result.MaxCases,
		RowLimit: result.RowLimit, Entries: result.Entries, DurationNs: int64(result.Duration),
		TargetJson: json.RawMessage(targetJSON), DiagnosticsJson: warmupOptionalJSON(diagnosticsJSON), CompletedAt: &completed,
		Has: &storedwriter.StoredWarmupRunHas{RunId: true, Status: true, ActiveKey: true,
			UpdatedAt: true, UpdatedBy: true,
			CacheName: true, CacheProvider: true, ConnectorName: true, IndexColumn: true,
			PlannedCases: true, CompletedCases: true, MaxCases: true, RowLimit: true,
			Entries: true, DurationNs: true, TargetJson: true, DiagnosticsJson: true, CompletedAt: true}})
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
	run, err := t.readWarmupRun(ctx, in.ReportID, in.RunID)
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
	runs, err := t.readWarmupRows(ctx, &storedreader.Input{ReportId: in.ReportID, VersionNo: in.VersionNo, PageLimit: limit, PageOffset: in.Input.Offset})
	if err != nil {
		return internal(err)
	}
	page := &sdk.WarmupRunPage{Items: runs, Limit: limit, Offset: in.Input.Offset}
	return assign(output, page)
}

func (t *Transport) recoverExpiredWarmupRuns(ctx context.Context, now time.Time) error {
	systemCtx, err := sdk.WithSystemIdentity(ctx, t.SystemCredentialProvider)
	if err != nil {
		return internal(err)
	}
	diagnostics, _ := json.Marshal([]sdk.Diagnostic{{Severity: "error", Code: "warmup_expired", Message: "cache warmup did not reach a terminal state before the server-owned timeout"}})
	diagnosticsPayload := json.RawMessage(diagnostics)
	actor := sdk.SystemPrincipal().Subject
	for {
		rows, err := t.expiredWarmupRuns(systemCtx, now.Add(-warmupRunTimeout), 100)
		if err != nil {
			return internal(err)
		}
		if len(rows) == 0 {
			return nil
		}
		changed := 0
		for _, row := range rows {
			err := t.writeWarmupRun(systemCtx, &storedwriter.StoredWarmupRun{RunId: row.RunId, Status: "failed",
				UpdatedAt: row.UpdatedAt, UpdatedBy: &actor, DiagnosticsJson: &diagnosticsPayload,
				CompletedAt: &now,
				Has: &storedwriter.StoredWarmupRunHas{RunId: true, Status: true, ActiveKey: true,
					UpdatedAt: true, UpdatedBy: true, DiagnosticsJson: true, CompletedAt: true}})
			if err == nil {
				changed++
				continue
			}
			var conflict *xhandler.Conflict
			if errors.As(err, &conflict) {
				continue // another worker changed the row after the expired read
			}
			return internal(err)
		}
		if changed == 0 {
			return nil // a concurrent worker owns the remaining transitions
		}
	}
}

func (t *Transport) readWarmupRun(ctx context.Context, reportID, runID string) (*sdk.WarmupRun, error) {
	runs, err := t.readWarmupRows(ctx, &storedreader.Input{ReportId: reportID, RunId: runID, PageLimit: 2})
	if err != nil {
		return nil, internal(err)
	}
	if len(runs) == 0 {
		return nil, &sdk.Error{Code: sdk.ErrorNotFound, Message: "warmup run not found"}
	}
	if len(runs) != 1 || runs[0].ReportID != reportID || runs[0].RunID != runID {
		return nil, internal(errors.New("warmup run reader returned an ambiguous or mismatched row"))
	}
	return runs[0], nil
}

func (t *Transport) readWarmupRows(ctx context.Context, input *storedreader.Input) (runs []*sdk.WarmupRun, err error) {
	resources := resource.New()
	if err := resources.Register(storedreader.WarmupRunDatlyResourceNamespace, storedreader.WarmupRunDatlyResources); err != nil {
		return nil, err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return nil, err
	}
	registration, target, err := readercomponent.Compile(reflect.TypeOf(storedreader.WarmupRunComponent{}), "store_read",
		reflect.TypeOf(storedreader.Input{}), reflect.TypeOf(storedreader.Output{}), resources, connector)
	if err != nil {
		return nil, err
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, runtime.Shutdown(context.Background())) }()
	input.Has = &storedreader.InputHas{ReportId: true, RunId: true, ActiveKey: true, VersionNo: true, PageLimit: true, PageOffset: true}
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*storedreader.Output)
	if !ok {
		return nil, fmt.Errorf("warmup run reader returned %T", value)
	}
	if len(output.WarmupRuns) > input.PageLimit {
		return nil, fmt.Errorf("warmup run reader exceeded page limit")
	}
	runs = make([]*sdk.WarmupRun, 0, len(output.WarmupRuns))
	for _, row := range output.WarmupRuns {
		if row == nil || input.ReportId != "" && row.ReportId != input.ReportId || input.RunId != "" && row.RunId != input.RunId || input.VersionNo != 0 && row.VersionNo != input.VersionNo {
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
			_ = json.Unmarshal(row.TargetJson, &run.Target)
		}
		if len(row.DiagnosticsJson) != 0 {
			_ = json.Unmarshal(row.DiagnosticsJson, &run.Diagnostics)
		}
		runs = append(runs, run)
	}
	return runs, nil
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
