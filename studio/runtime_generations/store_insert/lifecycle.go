package store_insert

import (
	context "context"
	"encoding/json"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	"strings"
)

// GenerationInsertRules customizes role Input.Generations.
type GenerationInsertRules struct{}

func GenerationInsertRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*GenerationInsertRules)(nil)).Elem()
}

var (
	GenerationInsertRulesHooks = new(GenerationInsertRules)
	GenerationInsertRulesDatly = GenerationInsertRulesDatlyType()
)

func (hooks *GenerationInsertRules) Init(_ context.Context, entity *StoredGeneration, _ xhandler.LifecycleContext[StoredGeneration, xhandler.NoParent, Output]) error {
	if entity == nil || entity.GenerationNo <= 0 || strings.TrimSpace(entity.SourceRevision) == "" ||
		entity.Status != "building" || entity.ReportCount != 0 ||
		strings.TrimSpace(entity.RequestedBy) == "" || entity.RequestedAt.IsZero() ||
		!json.Valid(entity.BuildManifestJson) {
		return fmt.Errorf("new runtime generation requires building identity, actor, timestamp and manifest")
	}
	return nil
}
func (hooks *GenerationInsertRules) Validate(ctx context.Context, entity *StoredGeneration, state xhandler.LifecycleContext[StoredGeneration, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *GenerationInsertRules) AfterSequence(ctx context.Context, entity *StoredGeneration, state xhandler.LifecycleContext[StoredGeneration, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *GenerationInsertRules) AfterQueue(ctx context.Context, entity *StoredGeneration, state xhandler.LifecycleContext[StoredGeneration, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *GenerationInsertRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
