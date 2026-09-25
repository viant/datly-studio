package store_recover

import (
	context "context"
	"encoding/json"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	"strings"
)

// PublicationRecoverRules customizes role Input.Publications.
type PublicationRecoverRules struct {
	Input *Input `bind:"kind=input,required"`
}

func PublicationRecoverRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*PublicationRecoverRules)(nil)).Elem()
}

var (
	PublicationRecoverRulesHooks = new(PublicationRecoverRules)
	PublicationRecoverRulesDatly = PublicationRecoverRulesDatlyType()
)

func (hooks *PublicationRecoverRules) Init(_ context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	if hooks.Input == nil || entity == nil || strings.TrimSpace(entity.ReportId) == "" ||
		entity.DesiredGeneration == nil || entity.FailureJson == nil ||
		!json.Valid([]byte(*entity.FailureJson)) || entity.Has == nil ||
		!entity.Has.ReportId || !entity.Has.DesiredGeneration ||
		!entity.Has.PublicationStatus || !entity.Has.FailureJson {
		return fmt.Errorf("publication recovery requires exact staged identity and JSON failure evidence")
	}
	previous := state.Previous
	if previous == nil || previous.ReportId != entity.ReportId || previous.DesiredGeneration == nil ||
		previous.PublicationStatus != entity.PublicationStatus ||
		(previous.PublicationStatus != "pending" && previous.PublicationStatus != "unpublishing") {
		return &xhandler.Conflict{Entity: "report_publication", Field: "publication_status", Reason: "staged publication changed before recovery"}
	}
	compensation := hooks.Input.Operation == "compensate_restore" || hooks.Input.Operation == "compensate_initial"
	if compensation {
		if hooks.Input.StagedGeneration <= 0 || *previous.DesiredGeneration != hooks.Input.StagedGeneration {
			return &xhandler.Conflict{Entity: "report_publication", Field: "desired_generation", Reason: "staged generation changed before compensation"}
		}
	} else if *previous.DesiredGeneration != *entity.DesiredGeneration {
		return &xhandler.Conflict{Entity: "report_publication", Field: "desired_generation", Reason: "staged generation changed before recovery"}
	}
	switch hooks.Input.Operation {
	case "restore_active":
		if previous.ActiveGeneration == nil || previous.ActiveVersionNo <= 0 ||
			entity.RuntimeRevision == nil || strings.TrimSpace(*entity.RuntimeRevision) == "" ||
			strings.TrimSpace(entity.SpecHash) == "" || !entity.Has.RuntimeRevision || !entity.Has.SpecHash {
			return fmt.Errorf("active publication recovery requires the previous active generation, version, revision and spec hash")
		}
		version := previous.ActiveVersionNo
		entity.SetDesiredVersionNo(&version)
		entity.SetDesiredGeneration(previous.ActiveGeneration)
		entity.SetRuntimeRevision(entity.RuntimeRevision)
		entity.SetSpecHash(entity.SpecHash)
		entity.SetPublicationStatus("active")
	case "fail":
		if previous.ActiveGeneration != nil || entity.Has.DesiredVersionNo ||
			entity.Has.RuntimeRevision || entity.Has.SpecHash {
			return fmt.Errorf("failed publication recovery requires no prior active generation")
		}
		entity.SetPublicationStatus("failed")
	case "compensate_restore":
		if hooks.Input.RestoreStatus != "active" && hooks.Input.RestoreStatus != "failed" ||
			entity.ActiveVersionNo <= 0 || entity.DesiredGeneration == nil || *entity.DesiredGeneration <= 0 ||
			strings.TrimSpace(entity.SpecHash) == "" || strings.TrimSpace(entity.PublishedBy) == "" ||
			entity.PublishedAt == nil || entity.PublishedAt.IsZero() ||
			!entity.Has.ActiveVersionNo || !entity.Has.DesiredVersionNo || !entity.Has.DesiredGeneration ||
			!entity.Has.ActiveGeneration || !entity.Has.RuntimeRevision || !entity.Has.SpecHash ||
			!entity.Has.PublishedBy || !entity.Has.PublishedAt || !entity.Has.ActivatedAt {
			return fmt.Errorf("publication compensation requires a complete previous snapshot")
		}
		entity.SetPublicationStatus(hooks.Input.RestoreStatus)
	case "compensate_initial":
		if previous.ActiveGeneration != nil || entity.ActiveGeneration != nil || entity.Has.ActiveVersionNo ||
			entity.Has.DesiredVersionNo || entity.Has.RuntimeRevision || entity.Has.SpecHash ||
			entity.Has.PublishedBy || entity.Has.PublishedAt || entity.Has.ActivatedAt {
			return fmt.Errorf("initial publication compensation requires no previous active publication")
		}
		entity.SetActiveGeneration(nil)
		entity.SetPublicationStatus("failed")
	default:
		return fmt.Errorf("unsupported publication recovery operation %q", hooks.Input.Operation)
	}
	entity.SetFailureJson(entity.FailureJson)
	return nil
}
func (hooks *PublicationRecoverRules) Validate(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationRecoverRules) AfterSequence(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationRecoverRules) AfterQueue(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationRecoverRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
