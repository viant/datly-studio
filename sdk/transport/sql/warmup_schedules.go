package sqltransport

import (
	"context"
	"errors"
	"fmt"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/internal/warmupschedule"
	"github.com/viant/datly-studio/sdk"
	runwrite "github.com/viant/datly-studio/studio/report_warmup_runs/store_write"
	admit "github.com/viant/datly-studio/studio/report_warmup_schedules/store_admit"
	read "github.com/viant/datly-studio/studio/report_warmup_schedules/store_read"
	write "github.com/viant/datly-studio/studio/report_warmup_schedules/store_write"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"reflect"
	"time"
)

// scheduleGraph invokes a generated native graph, sharing the enclosing native
// transaction when the caller already owns one. No SQL is executed here.
func (t *Transport) scheduleGraph(ctx context.Context, kind string, input any) (any, error) {
	var holder, in, out reflect.Type
	res := resource.New()
	switch kind {
	case "store_read":
		holder = reflect.TypeFor[read.ScheduleComponent]()
		in = reflect.TypeFor[read.Input]()
		out = reflect.TypeFor[read.Output]()
		if e := res.Register(read.ScheduleDatlyResourceNamespace, read.ScheduleDatlyResources); e != nil {
			return nil, e
		}
	case "store_write":
		holder = reflect.TypeFor[write.ScheduleComponent]()
		in = reflect.TypeFor[write.Input]()
		out = reflect.TypeFor[write.Output]()
		if e := res.Register(write.ScheduleDatlyResourceNamespace, write.ScheduleDatlyResources); e != nil {
			return nil, e
		}
	case "store_admit":
		holder = reflect.TypeFor[admit.ScheduleComponent]()
		in = reflect.TypeFor[admit.Input]()
		out = reflect.TypeFor[admit.Output]()
		if e := res.Register(admit.ScheduleDatlyResourceNamespace, admit.ScheduleDatlyResources); e != nil {
			return nil, e
		}
	default:
		return nil, fmt.Errorf("unknown schedule graph")
	}
	method := "PATCH"
	if kind == "store_read" {
		method = "GET"
	}
	target := dexec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: holder.PkgPath(), Name: "schedule"}, Route: spec.RouteRef{Method: method, Path: "/_studio/warmup-schedule/" + kind}}
	if t.ComponentInvoker != nil {
		return t.ComponentInvoker.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	}
	conn := &dsql.SQLComponent{DB: t.DB}
	if e := conn.RegisterConnector("studio", t.DB); e != nil {
		return nil, e
	}
	var entry *registry.RegisteredComponent
	var e error
	if kind == "store_read" {
		entry, target, e = readercomponent.Compile(holder, kind, in, out, res, conn)
	} else {
		entry, target, e = readercomponent.CompileWriter(holder, kind, in, out, res, conn)
	}
	if e != nil {
		return nil, e
	}
	rt, e := druntime.NewRuntime([]*registry.RegisteredComponent{entry}, druntime.WithResources(res))
	if e != nil {
		return nil, e
	}
	defer rt.Shutdown(context.WithoutCancel(ctx))
	return rt.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
}
func (t *Transport) readWarmupSchedules(ctx context.Context, report string, version int, due *time.Time) ([]*sdk.WarmupSchedule, error) {
	in := &read.Input{}
	in.SetScheduleId("")
	in.SetReportId(report)
	in.SetVersionNo(version)
	in.SetDueBefore(due)
	in.SetPageLimit(100)
	in.SetAfterDue(nil)
	in.SetAfterId("")
	result := make([]*sdk.WarmupSchedule, 0)
	for {
		value, e := t.scheduleGraph(ctx, "store_read", in)
		if e != nil {
			return nil, e
		}
		out, ok := value.(*read.Output)
		if !ok || out == nil {
			return nil, fmt.Errorf("invalid schedule reader output")
		}
		for _, s := range out.Schedules {
			if s.Revision == nil {
				return nil, fmt.Errorf("schedule revision is unavailable")
			}
			result = append(result, &sdk.WarmupSchedule{ScheduleID: s.ScheduleId, NamespaceID: s.NamespaceId, ReportID: s.ReportId, VersionNo: s.VersionNo, IntervalSeconds: s.IntervalSeconds, FirstRunAt: s.FirstRunAt, NextDueAt: s.NextDueAt, Enabled: s.Enabled, Revision: *s.Revision, LastRunID: s.LastRunId, CreatedAt: s.CreatedAt, CreatedBy: s.CreatedBy, UpdatedAt: s.UpdatedAt, UpdatedBy: s.UpdatedBy})
		}
		if len(out.Schedules) < 100 {
			break
		}
		last := out.Schedules[len(out.Schedules)-1]
		in.SetAfterDue(&last.NextDueAt)
		in.SetAfterId(last.ScheduleId)
	}
	return result, nil
}

// ManageWarmupSchedule is shared by the authorized native handler and SDK
// dispatch. Its caller owns the existing publish authorization check.
func (t *Transport) ManageWarmupSchedule(ctx context.Context, operation string, input, output any) error {
	var in struct {
		ReportID  string                      `json:"reportId"`
		VersionNo int                         `json:"versionNo"`
		Input     sdk.SaveWarmupScheduleInput `json:"input"`
	}
	if e := decode(input, &in); e != nil {
		var parseError *time.ParseError
		if errors.As(e, &parseError) {
			return &sdk.Error{Code: sdk.ErrorInvalidArgument, Field: "firstRunAt", Message: "firstRunAt must be a complete RFC3339 timestamp including a timezone", Cause: e}
		}
		return invalid(e)
	}
	if in.ReportID == "" || in.VersionNo <= 0 {
		return &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "reportId and positive versionNo are required"}
	}
	if _, e := t.getVersionValue(ctx, in.ReportID, in.VersionNo); e != nil {
		return e
	}
	ns, e := t.resourceWorkspaceID(ctx, nil, in.ReportID)
	if e != nil {
		return e
	}
	rows, e := t.readWarmupSchedules(ctx, in.ReportID, in.VersionNo, nil)
	if e != nil {
		return internal(e)
	}
	if len(rows) > 1 {
		return internal(fmt.Errorf("multiple schedules for one version"))
	}
	var current *sdk.WarmupSchedule
	if len(rows) == 1 {
		current = rows[0]
		if current.NamespaceID != ns {
			return &sdk.Error{Code: sdk.ErrorForbidden, Message: "schedule namespace does not match component"}
		}
	}
	if operation == sdk.OperationWarmupScheduleGet {
		return assign(output, sdk.WarmupScheduleResult{Schedule: current})
	}
	if in.Input.ExpectedRevision < 0 || current == nil && in.Input.ExpectedRevision != 0 || current != nil && in.Input.ExpectedRevision != current.Revision {
		return &sdk.Error{Code: sdk.ErrorConflict, Field: "revision", Message: "schedule changed; reload before saving"}
	}
	if operation == sdk.OperationWarmupScheduleDelete && current == nil {
		return &sdk.Error{Code: sdk.ErrorNotFound, Message: "schedule not found"}
	}
	actor, ok := sdk.PrincipalFromContext(ctx)
	if !ok {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "verified principal is required"}
	}
	now := t.now().UTC().Truncate(time.Microsecond)
	var s sdk.WarmupSchedule
	if current != nil {
		s = *current
	} else {
		id, e := generatedWarmupRunID()
		if e != nil {
			return internal(e)
		}
		s = sdk.WarmupSchedule{ScheduleID: "s" + id, NamespaceID: ns, ReportID: in.ReportID, VersionNo: in.VersionNo, CreatedAt: now, CreatedBy: actor.Subject}
	}
	remove := operation == sdk.OperationWarmupScheduleDelete
	if !remove {
		if e := warmupschedule.Validate(in.Input.IntervalSeconds, in.Input.FirstRunAt); e != nil {
			return e
		}
		first := in.Input.FirstRunAt.UTC().Truncate(time.Microsecond)
		if current == nil || s.IntervalSeconds != in.Input.IntervalSeconds || !s.FirstRunAt.Equal(first) {
			s.NextDueAt = first
		}
		s.IntervalSeconds = in.Input.IntervalSeconds
		s.FirstRunAt = first
		s.Enabled = in.Input.Enabled
	}
	s.UpdatedAt = now
	s.UpdatedBy = actor.Subject
	s.Revision = in.Input.ExpectedRevision
	row := &write.StoredSchedule{ScheduleId: s.ScheduleID, NamespaceId: s.NamespaceID, ReportId: s.ReportID, VersionNo: s.VersionNo, IntervalSeconds: s.IntervalSeconds, FirstRunAt: s.FirstRunAt, NextDueAt: s.NextDueAt, Enabled: s.Enabled, Revision: &s.Revision, LastRunId: s.LastRunID, CreatedAt: s.CreatedAt, CreatedBy: s.CreatedBy, UpdatedAt: now, UpdatedBy: s.UpdatedBy, ShouldDelete: remove,
		Has: &write.StoredScheduleHas{ScheduleId: true, NamespaceId: true, ReportId: true, VersionNo: true, IntervalSeconds: true, FirstRunAt: true, NextDueAt: true, Enabled: true, Revision: true, LastRunId: true, CreatedAt: true, CreatedBy: true, UpdatedAt: true, UpdatedBy: true, ShouldDelete: true}}
	request := &write.Input{}
	request.SetSchedules([]*write.StoredSchedule{row})
	if _, e = t.scheduleGraph(ctx, "store_write", request); e != nil {
		return classify(e, "warmup schedule", s.ScheduleID)
	}
	if remove {
		return assign(output, sdk.WarmupScheduleResult{})
	}
	s.Revision = *row.Revision
	return assign(output, sdk.WarmupScheduleResult{Schedule: &s})
}

func (t *Transport) consumeWarmupSchedule(ctx context.Context, s *sdk.WarmupSchedule, run *runwrite.StoredWarmupRun, runID string, now time.Time) error {
	next, e := warmupschedule.NextDue(s.NextDueAt, now, s.IntervalSeconds)
	if e != nil {
		return e
	}
	revision := s.Revision
	row := &admit.StoredSchedule{ScheduleId: s.ScheduleID, NamespaceId: s.NamespaceID, ReportId: s.ReportID, VersionNo: s.VersionNo, IntervalSeconds: s.IntervalSeconds, FirstRunAt: s.FirstRunAt, NextDueAt: next, Enabled: s.Enabled, Revision: &revision, LastRunId: &runID, CreatedAt: s.CreatedAt, CreatedBy: s.CreatedBy, UpdatedAt: now, UpdatedBy: warmupActor(ctx).Subject,
		Has: &admit.StoredScheduleHas{ScheduleId: true, NamespaceId: true, ReportId: true, VersionNo: true, IntervalSeconds: true, FirstRunAt: true, NextDueAt: true, Enabled: true, Revision: true, LastRunId: true, CreatedAt: true, CreatedBy: true, UpdatedAt: true, UpdatedBy: true, Run: true}}
	if run != nil {
		row.Run = []*admit.AcceptedRun{{NamespaceId: run.NamespaceId, RunId: run.RunId, ReportId: run.ReportId, VersionNo: run.VersionNo, SourceRevision: run.SourceRevision, SpecHash: run.SpecHash, PlanKey: run.PlanKey, ActiveKey: run.ActiveKey, Status: run.Status, RequestedBy: run.RequestedBy, TargetJson: run.TargetJson, RequestedAt: run.RequestedAt, DeadlineAt: run.DeadlineAt, CreatedAt: run.CreatedAt, CreatedBy: run.CreatedBy, UpdatedAt: run.UpdatedAt, UpdatedBy: run.UpdatedBy, Has: &admit.AcceptedRunHas{NamespaceId: true, RunId: true, ReportId: true, VersionNo: true, SourceRevision: true, SpecHash: true, PlanKey: true, ActiveKey: true, Status: true, RequestedBy: true, TargetJson: true, RequestedAt: true, DeadlineAt: true, CreatedAt: true, CreatedBy: true, UpdatedAt: true, UpdatedBy: true}}}
	}
	in := &admit.Input{}
	in.SetSchedules([]*admit.StoredSchedule{row})
	_, e = t.scheduleGraph(ctx, "store_admit", in)
	if e != nil {
		return classify(e, "warmup schedule", s.ScheduleID)
	}
	return nil
}

// RunDueWarmupSchedules is host-only. Each admission retains ordinary namespace
// and warmup authorization; scanning due metadata grants no execution authority.
func (t *Transport) RunDueWarmupSchedules(ctx context.Context) error {
	principal, ok := sdk.PrincipalFromContext(ctx)
	credential, verified := sdk.VerifiedCredentialFromContext(ctx)
	if !ok || !verified || credential.Bearer == "" || principal.Subject != sdk.SystemPrincipal().Subject {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "verified service identity is required for scheduled warmup"}
	}
	if t.Warmup == nil {
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "dynamic warmup executor is not configured"}
	}
	now := t.now().UTC()
	rows, e := t.readWarmupSchedules(ctx, "", 0, &now)
	if e != nil {
		return e
	}
	var failures []error
	for _, s := range rows {
		if e := ctx.Err(); e != nil {
			return e
		}
		selected := sdk.WithNamespaceSelection(ctx, s.NamespaceID)
		selected, e = t.resolveNamespace(selected)
		if e == nil {
			e = t.authorize(selected, sdk.OperationVersionWarmup, versionIdentityRequest{ReportID: s.ReportID, VersionNo: s.VersionNo})
		}
		if e == nil {
			var result sdk.WarmupRun
			e = t.startWarmupRun(selected, versionIdentityRequest{ReportID: s.ReportID, VersionNo: s.VersionNo}, &result, s)
		}
		if e != nil {
			failures = append(failures, fmt.Errorf("schedule %s: %w", s.ScheduleID, e))
		}
	}
	return errors.Join(failures...)
}
