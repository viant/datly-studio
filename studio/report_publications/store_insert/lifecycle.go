package store_insert

import (
	context "context"
	"encoding/hex"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	"strings"
)

// PublicationInsertRules customizes role Input.Publications.
type PublicationInsertRules struct{}

func PublicationInsertRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*PublicationInsertRules)(nil)).Elem()
}

var (
	PublicationInsertRulesHooks = new(PublicationInsertRules)
	PublicationInsertRulesDatly = PublicationInsertRulesDatlyType()
)

func (hooks *PublicationInsertRules) Init(_ context.Context, entity *StoredPublication, _ xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	if entity == nil || strings.TrimSpace(entity.ReportId) == "" || entity.ActiveVersionNo <= 0 ||
		entity.DesiredVersionNo == nil || *entity.DesiredVersionNo != entity.ActiveVersionNo ||
		entity.DesiredGeneration <= 0 || entity.ActiveGeneration != nil ||
		entity.PublicationStatus != "pending" || entity.RuntimeRevision == nil ||
		strings.TrimSpace(*entity.RuntimeRevision) == "" || strings.TrimSpace(entity.PublishedBy) == "" ||
		entity.PublishedAt.IsZero() || entity.ActivatedAt != nil || entity.FailureJson != nil {
		return fmt.Errorf("new publication requires a pending exact-version stage without active generation")
	}
	if len(entity.SpecHash) != 64 {
		return fmt.Errorf("new publication requires SHA-256 spec hash")
	}
	if _, err := hex.DecodeString(entity.SpecHash); err != nil {
		return fmt.Errorf("publication spec hash: %w", err)
	}
	return nil
}
func (hooks *PublicationInsertRules) Validate(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationInsertRules) AfterSequence(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationInsertRules) AfterQueue(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationInsertRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
