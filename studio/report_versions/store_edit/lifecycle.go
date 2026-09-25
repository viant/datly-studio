package store_edit

import (
	context "context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	"strings"
)

// VersionEditRules customizes role Input.Versions.
type VersionEditRules struct{}

func VersionEditRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*VersionEditRules)(nil)).Elem()
}

var (
	VersionEditRulesHooks = new(VersionEditRules)
	VersionEditRulesDatly = VersionEditRulesDatlyType()
)

func (hooks *VersionEditRules) Init(_ context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	if entity == nil || strings.TrimSpace(entity.ReportId) == "" || entity.VersionNo <= 0 ||
		entity.SourceRevision == nil || entity.Has == nil || !entity.Has.ReportId ||
		!entity.Has.VersionNo || !entity.Has.AuthoredSql || !entity.Has.AuthoredDql ||
		!entity.Has.ComponentSpecJson || !entity.Has.SpecHash || !entity.Has.GeneratedDql ||
		!entity.Has.CompileStatus || !entity.Has.CompileDiagnosticsJson || !entity.Has.SourceRevision {
		return fmt.Errorf("version edit requires a complete patch and expected source revision")
	}
	previous := state.Previous
	if previous == nil || previous.SourceRevision == nil || *previous.SourceRevision != *entity.SourceRevision {
		return &xhandler.Conflict{Entity: "report_version", Field: "source_revision", Reason: "expected source revision does not match"}
	}
	if entity.CompileStatus != "pending" || string(entity.CompileDiagnosticsJson) != "[]" ||
		!json.Valid(entity.ComponentSpecJson) || len(entity.SpecHash) != 64 {
		return fmt.Errorf("version edit requires pending compilation, cleared diagnostics, valid spec and SHA-256 hash")
	}
	if _, err := hex.DecodeString(entity.SpecHash); err != nil {
		return fmt.Errorf("version edit spec hash: %w", err)
	}
	next := *entity.SourceRevision + 1
	if *entity.SourceRevision == 0 {
		next = 2
	}
	entity.SetSourceRevision(&next)
	return nil
}
func (hooks *VersionEditRules) Validate(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *VersionEditRules) AfterSequence(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *VersionEditRules) AfterQueue(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *VersionEditRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
