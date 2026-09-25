package store_insert

import (
	context "context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	"strings"
)

// VersionInsertRules customizes role Input.Versions.
type VersionInsertRules struct{}

func VersionInsertRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*VersionInsertRules)(nil)).Elem()
}

var (
	VersionInsertRulesHooks = new(VersionInsertRules)
	VersionInsertRulesDatly = VersionInsertRulesDatlyType()
)

func (hooks *VersionInsertRules) Init(_ context.Context, entity *StoredVersion, _ xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	if entity == nil || strings.TrimSpace(entity.ReportId) == "" || entity.VersionNo <= 0 ||
		entity.State != "draft" || entity.CompileStatus != "pending" || entity.SourceRevision != 1 ||
		strings.TrimSpace(entity.CreatedBy) == "" || entity.CreatedAt.IsZero() ||
		entity.SpecFormatVersion != "studio.v1" || entity.DatlyVersion != "v1" ||
		entity.CompilerVersion != "studio.v1" {
		return fmt.Errorf("new report version requires draft identity, actor and initial metadata")
	}
	switch entity.AuthoringMode {
	case "sql", "dql", "structured":
	default:
		return fmt.Errorf("unsupported version authoring mode %q", entity.AuthoringMode)
	}
	if !json.Valid(entity.ComponentSpecJson) || !json.Valid(entity.TypeManifestJson) {
		return fmt.Errorf("new report version requires valid JSON manifests")
	}
	if len(entity.SpecHash) != 64 {
		return fmt.Errorf("new report version requires a SHA-256 spec hash")
	}
	if _, err := hex.DecodeString(entity.SpecHash); err != nil {
		return fmt.Errorf("new report version spec hash: %w", err)
	}
	return nil
}
func (hooks *VersionInsertRules) Validate(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *VersionInsertRules) AfterSequence(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *VersionInsertRules) AfterQueue(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *VersionInsertRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
