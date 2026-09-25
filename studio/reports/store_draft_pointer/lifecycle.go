package store_draft_pointer

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	xhandler "github.com/viant/xdatly/handler"
)

// DraftPointerRules moves a report's draft pointer to a newly allocated
// version. The caller supplies the report's expected etag, which Datly's
// universal writer captured as the concurrency token before Init runs. Init
// advances the committed etag by exactly one and the matched UPDATE rejects a
// concurrent report change, so the caller-owned transaction rolls back the
// version rows written alongside the pointer.
type DraftPointerRules struct{}

func DraftPointerRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*DraftPointerRules)(nil)).Elem()
}

var (
	DraftPointerRulesHooks = new(DraftPointerRules)
	DraftPointerRulesDatly = DraftPointerRulesDatlyType()
)

const draftPointerEntity = "report"

func conflict(reason string) error {
	return &xhandler.Conflict{Entity: draftPointerEntity, Field: "etag", Reason: reason}
}

// Init turns the caller's expected etag into the next committed etag. A
// missing or deleted report has no Previous row and is denied here.
func (hooks *DraftPointerRules) Init(_ context.Context, report *DraftPointer, state xhandler.LifecycleContext[DraftPointer, xhandler.NoParent, Output]) error {
	if report == nil || strings.TrimSpace(report.Id) == "" {
		return fmt.Errorf("report id is required")
	}
	if state.Previous == nil {
		return fmt.Errorf("report %s does not exist or is deleted", report.Id)
	}
	if report.Etag == nil {
		return conflict("expected report etag is required")
	}
	if report.CurrentDraftVersion == nil || *report.CurrentDraftVersion <= 0 {
		return fmt.Errorf("report %s draft version is required", report.Id)
	}
	if report.UpdatedAt == nil || report.UpdatedAt.IsZero() {
		return fmt.Errorf("report %s updated_at is required", report.Id)
	}
	next := *report.Etag + 1
	report.SetEtag(&next)
	return nil
}

// Validate denies a stale expectation and a pointer that does not advance.
func (hooks *DraftPointerRules) Validate(_ context.Context, report *DraftPointer, state xhandler.LifecycleContext[DraftPointer, xhandler.NoParent, Output]) error {
	previous := state.Previous
	if report == nil || previous == nil || report.Etag == nil || previous.Etag == nil || report.CurrentDraftVersion == nil {
		return fmt.Errorf("draft pointer state is unavailable")
	}
	if *report.Etag != *previous.Etag+1 {
		return conflict("report etag is stale")
	}
	if previous.CurrentDraftVersion != nil && *report.CurrentDraftVersion <= *previous.CurrentDraftVersion {
		return conflict("draft pointer must advance to a newer version")
	}
	return nil
}

func (hooks *DraftPointerRules) AfterSequence(context.Context, *DraftPointer, xhandler.LifecycleContext[DraftPointer, xhandler.NoParent, Output]) error {
	return nil
}

func (hooks *DraftPointerRules) AfterQueue(context.Context, *DraftPointer, xhandler.LifecycleContext[DraftPointer, xhandler.NoParent, Output]) error {
	return nil
}

func (hooks *DraftPointerRules) Finalize(context.Context, *Input, *Output, xhandler.Outcome) error {
	return nil
}
