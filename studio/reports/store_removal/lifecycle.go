package store_removal

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
	xhandler "github.com/viant/xdatly/handler"
)

type RemovalRules struct {
	Input *Input `bind:"kind=input,required"`
}
type VersionRules struct {
	Input *Input `bind:"kind=input,required"`
}
type WarmupRules struct {
	Input *Input `bind:"kind=input,required"`
}
type ReceiptRules struct {
	Input *Input `bind:"kind=input,required"`
}

func dependency(message string) error {
	return &xhandler.Conflict{Entity: "component", Field: "dependency", Reason: message}
}
func (h *RemovalRules) Init(_ context.Context, row *RemovalReport, state xhandler.LifecycleContext[RemovalReport, xhandler.NoParent, Output]) error {
	in := h.Input
	previous := state.Previous
	if in == nil || row == nil || previous == nil || row.Id != in.ReportId || len(in.Reports) != 1 || row.Etag == nil || previous.Etag == nil {
		return dependency("Component changed or is no longer available; reload before deleting")
	}
	if in.Actor == "" || previous.OwnerId != in.Actor || row.OwnerId != previous.OwnerId || previous.NamespaceId != in.NamespaceId || row.NamespaceId != previous.NamespaceId {
		return fmt.Errorf("removal requires the verified component owner and namespace")
	}
	if *row.Etag != *previous.Etag {
		return dependency("Component configuration changed; reload before deleting")
	}
	if row.ShouldDelete != (in.TargetVersion == 0) || in.TargetVersion < 0 {
		return fmt.Errorf("removal target is inconsistent")
	}
	for _, p := range in.Publications {
		if p == nil {
			continue
		}
		matches := in.TargetVersion == 0 || (p.ActiveVersionNo != nil && *p.ActiveVersionNo == in.TargetVersion) || (p.DesiredVersionNo != nil && *p.DesiredVersionNo == in.TargetVersion)
		if matches {
			return dependency("The deletion target is referenced by a publication; unpublish or resolve its transition before deleting")
		}
	}
	for _, run := range in.WarmupSnapshot {
		if run != nil && run.Status != nil && (*run.Status == "accepted" || *run.Status == "running") {
			return dependency("The deletion target has accepted or running warmup work; wait for it to finish before deleting")
		}
	}
	for _, stored := range in.BundleBindings {
		if stored == nil {
			continue
		}
		var bindings []releasevalue.ComponentBinding
		if err := json.Unmarshal(stored.BindingsJson, &bindings); err != nil {
			return dependency("Release reference metadata is unavailable; repair it before deleting")
		}
		for _, b := range bindings {
			if b.ComponentKind != "dynamic" || b.ComponentID != row.Id {
				continue
			}
			version, err := strconv.Atoi(b.ComponentRevision)
			if err != nil || version < 1 {
				return dependency("Release reference metadata is unavailable; repair it before deleting")
			}
			if in.TargetVersion == 0 || version == in.TargetVersion {
				return dependency("The deletion target is referenced by a retained release bundle")
			}
		}
	}
	if !row.ShouldDelete {
		if len(row.Version) != 1 || row.Version[0] == nil || row.Version[0].VersionNo != in.TargetVersion || len(row.Deletion) != 0 {
			return fmt.Errorf("version removal must target one exact version")
		}
		if previous.CurrentDraftVersion != nil && *previous.CurrentDraftVersion == in.TargetVersion {
			if row.CurrentDraftVersion != nil {
				return fmt.Errorf("removed draft pointer must be cleared")
			}
		} else if !equalInt(previous.CurrentDraftVersion, row.CurrentDraftVersion) {
			return fmt.Errorf("unrelated draft pointer cannot change during removal")
		}
		next := *row.Etag + 1
		row.SetEtag(&next)
	} else if len(row.Deletion) != 1 {
		return fmt.Errorf("component removal requires one immutable ownership receipt")
	}
	if row.UpdatedAt == nil || row.UpdatedAt.IsZero() {
		return fmt.Errorf("removal timestamp is required")
	}
	return nil
}
func equalInt(a, b *int) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func (h *VersionRules) Init(_ context.Context, row *RemovalVersion, state xhandler.LifecycleContext[RemovalVersion, RemovalReport, Output]) error {
	if h.Input == nil || row == nil || state.Previous == nil || !row.ShouldDelete || row.SourceRevision == nil || row.ReportId != h.Input.ReportId || row.NamespaceId != h.Input.NamespaceId || h.Input.TargetVersion > 0 && row.VersionNo != h.Input.TargetVersion {
		return dependency("Version changed or is no longer available; reload before deleting")
	}
	return nil
}
func (h *WarmupRules) Init(_ context.Context, row *RemovalWarmup, state xhandler.LifecycleContext[RemovalWarmup, RemovalVersion, Output]) error {
	if h.Input == nil || row == nil || state.Previous == nil || row.ReportId != h.Input.ReportId || !row.ShouldDelete {
		return dependency("Warmup evidence changed before deletion")
	}
	switch state.Previous.Status {
	case "completed", "partial", "failed", "canceled":
		return nil
	}
	return dependency("Accepted or running warmup work cannot be removed")
}
func (h *ReceiptRules) Init(_ context.Context, row *RemovalReceipt, state xhandler.LifecycleContext[RemovalReceipt, RemovalReport, Output]) error {
	if h.Input == nil || h.Input.TargetVersion != 0 || row == nil || state.Previous != nil || row.ComponentId != h.Input.ReportId || row.NamespaceId != h.Input.NamespaceId || row.OwnerId != h.Input.Actor || row.DeletedBy != h.Input.Actor || row.DeletedAt.IsZero() {
		return fmt.Errorf("component deletion receipt must preserve exact final ownership")
	}
	return nil
}

func RemovalRulesDatlyType() reflect.Type { return reflect.TypeFor[RemovalRules]() }

var RemovalRulesDatly = RemovalRulesDatlyType()

func (h *RemovalRules) Validate(context.Context, *RemovalReport, xhandler.LifecycleContext[RemovalReport, xhandler.NoParent, Output]) error {
	return nil
}
func (h *RemovalRules) AfterSequence(context.Context, *RemovalReport, xhandler.LifecycleContext[RemovalReport, xhandler.NoParent, Output]) error {
	return nil
}
func (h *RemovalRules) AfterQueue(context.Context, *RemovalReport, xhandler.LifecycleContext[RemovalReport, xhandler.NoParent, Output]) error {
	return nil
}
func (h *RemovalRules) Finalize(context.Context, *Input, *Output, xhandler.Outcome) error { return nil }

func VersionRulesDatlyType() reflect.Type { return reflect.TypeFor[VersionRules]() }

var VersionRulesDatly = VersionRulesDatlyType()

func (h *VersionRules) Validate(context.Context, *RemovalVersion, xhandler.LifecycleContext[RemovalVersion, RemovalReport, Output]) error {
	return nil
}
func (h *VersionRules) AfterSequence(context.Context, *RemovalVersion, xhandler.LifecycleContext[RemovalVersion, RemovalReport, Output]) error {
	return nil
}
func (h *VersionRules) AfterQueue(context.Context, *RemovalVersion, xhandler.LifecycleContext[RemovalVersion, RemovalReport, Output]) error {
	return nil
}
func (h *VersionRules) Finalize(context.Context, *Input, *Output, xhandler.Outcome) error { return nil }

func WarmupRulesDatlyType() reflect.Type { return reflect.TypeFor[WarmupRules]() }

var WarmupRulesDatly = WarmupRulesDatlyType()

func (h *WarmupRules) Validate(context.Context, *RemovalWarmup, xhandler.LifecycleContext[RemovalWarmup, RemovalVersion, Output]) error {
	return nil
}
func (h *WarmupRules) AfterSequence(context.Context, *RemovalWarmup, xhandler.LifecycleContext[RemovalWarmup, RemovalVersion, Output]) error {
	return nil
}
func (h *WarmupRules) AfterQueue(context.Context, *RemovalWarmup, xhandler.LifecycleContext[RemovalWarmup, RemovalVersion, Output]) error {
	return nil
}
func (h *WarmupRules) Finalize(context.Context, *Input, *Output, xhandler.Outcome) error { return nil }

func ReceiptRulesDatlyType() reflect.Type { return reflect.TypeFor[ReceiptRules]() }

var ReceiptRulesDatly = ReceiptRulesDatlyType()

func (h *ReceiptRules) Validate(context.Context, *RemovalReceipt, xhandler.LifecycleContext[RemovalReceipt, RemovalReport, Output]) error {
	return nil
}
func (h *ReceiptRules) AfterSequence(context.Context, *RemovalReceipt, xhandler.LifecycleContext[RemovalReceipt, RemovalReport, Output]) error {
	return nil
}
func (h *ReceiptRules) AfterQueue(context.Context, *RemovalReceipt, xhandler.LifecycleContext[RemovalReceipt, RemovalReport, Output]) error {
	return nil
}
func (h *ReceiptRules) Finalize(context.Context, *Input, *Output, xhandler.Outcome) error { return nil }

// ScheduleRules retains optimistic concurrency for a version-owned schedule.
type ScheduleRules struct{}

func ScheduleRulesDatlyType() reflect.Type { return reflect.TypeFor[ScheduleRules]() }

var ScheduleRulesDatly = ScheduleRulesDatlyType()

func (h *ScheduleRules) Init(ctx context.Context, row *RemovalSchedule, state xhandler.LifecycleContext[RemovalSchedule, RemovalVersion, Output]) error {
	if row == nil || state.Previous == nil || !row.ShouldDelete || row.Revision == nil {
		return dependency("Warmup schedule changed before deletion")
	}
	return nil
}

func (h *ScheduleRules) Validate(ctx context.Context, row *RemovalSchedule, state xhandler.LifecycleContext[RemovalSchedule, RemovalVersion, Output]) error {
	return nil
}

func (h *ScheduleRules) AfterSequence(ctx context.Context, row *RemovalSchedule, state xhandler.LifecycleContext[RemovalSchedule, RemovalVersion, Output]) error {
	return nil
}

func (h *ScheduleRules) AfterQueue(ctx context.Context, row *RemovalSchedule, state xhandler.LifecycleContext[RemovalSchedule, RemovalVersion, Output]) error {
	return nil
}

func (h *ScheduleRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
