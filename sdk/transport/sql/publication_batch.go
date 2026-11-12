package sqltransport

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/viant/datly-studio/sdk"
	generationstate "github.com/viant/datly-studio/studio/runtime_generations/store_state"
)

// A publicationMember is request-owned. Stage and activation share one database
// transaction per phase and one runtime generation across every member.
type publicationMember struct {
	request     publishRequest
	version     *sdk.ReportVersion
	event       publicationEventRecord
	previous    publicationState
	hadPrevious bool
}

func (t *Transport) publish(ctx context.Context, input, output any, operation string) (returnErr error) {
	t.publicationMu.Lock()
	defer t.publicationMu.Unlock()
	var primary publishRequest
	if err := decode(input, &primary); err != nil {
		return invalid(err)
	}
	if !validPublicationOperation(operation) || operation == "unpublish" {
		return invalid(errors.New("invalid publication operation"))
	}
	if operation != "publish" && len(primary.Input.AdditionalVersions) > 0 {
		return invalid(errors.New("additional versions are supported only for publication"))
	}
	requests := []publishRequest{primary}
	for _, version := range primary.Input.AdditionalVersions {
		requests = append(requests, publishRequest{ReportID: version.ReportID, VersionNo: version.VersionNo, Input: sdk.PublishInput{ExpectedSourceRevision: version.ExpectedSourceRevision, RequestedBy: primary.Input.RequestedBy, Reason: primary.Input.Reason}})
	}
	seen := make(map[string]bool, len(requests))
	for _, request := range requests {
		if strings.TrimSpace(request.ReportID) == "" || request.VersionNo <= 0 || request.Input.ExpectedSourceRevision <= 0 {
			return invalid(errors.New("reportId, versionNo and positive expectedSourceRevision are required for every publication member"))
		}
		if seen[request.ReportID] {
			return invalid(errors.New("a component may occur only once in a publication"))
		}
		seen[request.ReportID] = true
	}
	var members []*publicationMember
	defer func() {
		if returnErr != nil {
			for _, member := range members {
				t.recordPublicationFailure(ctx, member.event, returnErr)
			}
		}
	}()
	var owner, namespace string
	for index, request := range requests {
		// Invoke already authorized the primary member. Additional members use
		// precisely the same component-scoped authorization, before staging.
		if index > 0 {
			if err := t.authorize(ctx, sdk.OperationPublicationPublish, request); err != nil {
				return err
			}
		}
		if principal, ok := sdk.PrincipalFromContext(ctx); ok {
			if request.Input.RequestedBy != "" && request.Input.RequestedBy != principal.Subject {
				return &sdk.Error{Code: sdk.ErrorForbidden, Message: "publisher must match the Studio principal"}
			}
			request.Input.RequestedBy = principal.Subject
		}
		if strings.TrimSpace(request.Input.RequestedBy) == "" {
			return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Studio principal is required"}
		}
		component, err := t.getReportValue(ctx, request.ReportID)
		if err != nil {
			return err
		}
		if index == 0 {
			owner, namespace = component.OwnerID, component.Namespace
		} else if owner != component.OwnerID || namespace != component.Namespace {
			return invalid(errors.New("publication members must belong to the same namespace owner and namespace"))
		}
		version, err := t.getVersionValue(ctx, request.ReportID, request.VersionNo)
		if err != nil {
			return err
		}
		member := &publicationMember{request: request, version: version, event: publicationEventRecord{ReportID: request.ReportID, OwnerID: component.OwnerID, Operation: operation, VersionNo: &request.VersionNo, RequestedBy: request.Input.RequestedBy, Reason: boundedPublicationText(request.Input.Reason)}}
		members = append(members, member)
		if request.Input.ExpectedSourceRevision != version.SourceRevision {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "version source revision does not match", ExpectedSourceRevision: request.Input.ExpectedSourceRevision, CurrentSourceRevision: version.SourceRevision}
		}
		if version.CompileStatus != "valid" {
			return &sdk.Error{Code: sdk.ErrorInvalidArgument, Message: "version must validate before publication"}
		}
		if err = t.requireNamespaceAuthorizationPredicate(ctx, request.ReportID, version); err != nil {
			return err
		}
	}
	if t.Activator == nil {
		return &sdk.Error{Code: sdk.ErrorUnavailable, Message: "dynamic runtime activator is not configured"}
	}
	generation, now, err := t.stagePublications(ctx, members)
	if err != nil {
		return err
	}
	for _, member := range members {
		member.event.GenerationNo = &generation
	}
	if err = t.Activator.Reload(ctx, generation); err != nil {
		if restoreErr := t.restoreFailedPublications(context.WithoutCancel(ctx), members, generation, err); restoreErr != nil {
			err = errors.Join(err, fmt.Errorf("restore failed publication: %w", restoreErr))
		}
		return publicationRuntimeError("runtime_reload", err)
	}
	for _, member := range members {
		member.event.Status, member.event.OccurredAt = "succeeded", now
	}
	if err = t.activatePublications(ctx, members, generation, now); err != nil {
		rollbackGeneration := int64(0)
		for _, member := range members {
			if member.hadPrevious && member.previous.activeGeneration.Valid {
				rollbackGeneration = member.previous.activeGeneration.Int64
				break
			}
		}
		if rollbackErr := t.Activator.Reload(context.WithoutCancel(ctx), rollbackGeneration); rollbackErr != nil {
			err = errors.Join(err, fmt.Errorf("runtime compensation failed: %w", rollbackErr))
		}
		if restoreErr := t.restoreFailedPublications(context.WithoutCancel(ctx), members, generation, err); restoreErr != nil {
			err = errors.Join(err, fmt.Errorf("restore failed publication: %w", restoreErr))
		}
		return publicationRuntimeError("runtime_activation", err)
	}
	// Keep the established response shape. The primary receipt identifies the
	// shared generation; each member is also readable through publications.get.
	return t.getPublication(ctx, primary.ReportID, output)
}

func (t *Transport) stagePublications(ctx context.Context, members []*publicationMember) (int64, time.Time, error) {
	tx, err := t.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, time.Time{}, internal(err)
	}
	defer tx.Rollback()
	now := t.now()
	for _, member := range members {
		r := member.request
		if err = t.requirePublicationVersion(ctx, tx, r.ReportID, r.VersionNo, r.Input.ExpectedSourceRevision); err != nil {
			return 0, time.Time{}, err
		}
	}
	if err = t.ensureNoStagedGeneration(ctx, tx, now); err != nil {
		return 0, time.Time{}, err
	}
	first := members[0].request
	runtimeRevision := fmt.Sprintf("%s:%d:%s", first.ReportID, first.VersionNo, rand.Text())
	generation, err := t.createBuildingGeneration(ctx, tx, runtimeRevision, first.Input.RequestedBy, now)
	if err != nil {
		return 0, time.Time{}, classify(err, "runtime generation", fmt.Sprint(generation))
	}
	for _, member := range members {
		member.previous, member.hadPrevious, err = t.stagePublicationRow(ctx, tx, member.request, member.version, generation, runtimeRevision, now)
		if err != nil {
			return 0, time.Time{}, err
		}
	}
	if err = commitWithServerGuard(ctx, tx); err != nil {
		return 0, time.Time{}, internal(err)
	}
	return generation, now, nil
}

func (t *Transport) activatePublications(ctx context.Context, members []*publicationMember, generation int64, now time.Time) error {
	tx, err := t.DB.BeginTx(ctx, nil)
	if err != nil {
		return internal(err)
	}
	defer tx.Rollback()
	for _, member := range members {
		r := member.request
		if err = t.requirePublicationVersion(ctx, tx, r.ReportID, r.VersionNo, r.Input.ExpectedSourceRevision); err != nil {
			return err
		}
	}
	for _, member := range members {
		r := member.request
		if err = t.activatePublicationRow(ctx, tx, r.ReportID, r.VersionNo, generation, now, member.event); err != nil {
			return err
		}
	}
	// One namespace-wide repoint, not one scan per member. Already activated
	// batch members are counted but need no second write.
	others, err := t.repointActivePublications(ctx, tx, members[0].request.ReportID, generation)
	if err != nil {
		return err
	}
	if err = t.activateGenerationState(ctx, tx, generation, others+1, now); err != nil {
		return err
	}
	if err = commitWithServerGuard(ctx, tx); err != nil {
		return internal(err)
	}
	return nil
}

func (t *Transport) restoreFailedPublications(ctx context.Context, members []*publicationMember, generation int64, cause error) error {
	tx, err := t.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	failure, _ := json.Marshal([]sdk.Diagnostic{{Severity: "error", Code: "runtime_reload", Message: cause.Error()}})
	diagnostics := string(failure)
	if err = t.writeGenerationState(ctx, tx, "fail", generation, []*generationstate.StoredGeneration{{GenerationNo: generation, Status: "building", DiagnosticsJson: &diagnostics, Has: &generationstate.StoredGenerationHas{GenerationNo: true, Status: true, DiagnosticsJson: true}}}); err != nil {
		return fmt.Errorf("fail staged generation: %w", err)
	}
	for _, member := range members {
		if err = t.restorePublicationRow(ctx, tx, member.request.ReportID, generation, member.previous, member.hadPrevious, diagnostics); err != nil {
			return err
		}
	}
	return commitWithServerGuard(ctx, tx)
}
