package store_validation

import (
	context "context"
	"encoding/json"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	"strings"
)

// VersionValidationRules customizes role Input.Versions.
type VersionValidationRules struct{}

func VersionValidationRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*VersionValidationRules)(nil)).Elem()
}

var (
	VersionValidationRulesHooks = new(VersionValidationRules)
	VersionValidationRulesDatly = VersionValidationRulesDatlyType()
)

func (hooks *VersionValidationRules) Init(_ context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	if entity == nil || strings.TrimSpace(entity.ReportId) == "" || entity.VersionNo <= 0 ||
		entity.SourceRevision == nil || entity.ValidatedAt == nil || entity.ValidatedAt.IsZero() ||
		entity.Has == nil || !entity.Has.ReportId || !entity.Has.VersionNo ||
		!entity.Has.CompileStatus || !entity.Has.CompileDiagnosticsJson ||
		!entity.Has.ValidatedAt || !entity.Has.SourceRevision {
		return fmt.Errorf("version validation requires identity, timestamp and expected source revision")
	}
	previous := state.Previous
	if previous == nil || previous.SourceRevision == nil || *previous.SourceRevision != *entity.SourceRevision {
		return &xhandler.Conflict{Entity: "report_version", Field: "source_revision", Reason: "validated source revision changed"}
	}
	if entity.CompileStatus != "valid" && entity.CompileStatus != "invalid" {
		return fmt.Errorf("unsupported version validation status %q", entity.CompileStatus)
	}
	if !json.Valid(entity.CompileDiagnosticsJson) {
		return fmt.Errorf("version validation diagnostics must be valid JSON")
	}
	// Validation records evidence for this source revision; it does not change
	// the source revision itself. Datly still matches the token in the UPDATE.
	return nil
}
func (hooks *VersionValidationRules) Validate(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *VersionValidationRules) AfterSequence(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *VersionValidationRules) AfterQueue(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *VersionValidationRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
