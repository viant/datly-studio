package sqltransport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/viant/datly-studio/sdk"
	publicationrecover "github.com/viant/datly-studio/studio/report_publications/store_recover"
	generationstate "github.com/viant/datly-studio/studio/runtime_generations/store_state"
	xhandler "github.com/viant/xdatly/handler"
)

func (t *Transport) ensureNoStagedGeneration(ctx context.Context, tx *sql.Tx, now time.Time) error {
	building, err := t.readGenerationCatalog(ctx, tx, 0, "building")
	if err != nil {
		return internal(err)
	}
	cutoff := now.Add(-stagedGenerationLease)
	for _, generation := range building {
		if !generation.RequestedAt.Before(cutoff) {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "another runtime generation is already building"}
		}
	}
	if len(building) == 0 {
		return nil
	}
	failure, err := json.Marshal([]sdk.Diagnostic{{Severity: "error", Code: "staged_generation_expired", Message: "staged runtime generation expired before activation"}})
	if err != nil {
		return internal(err)
	}
	diagnostics := string(failure)
	for _, generation := range building {
		if err := t.recoverExpiredPublications(ctx, tx, generation.GenerationNo, diagnostics); err != nil {
			return err
		}
		err := t.writeGenerationState(ctx, tx, "fail", generation.GenerationNo, []*generationstate.StoredGeneration{{
			GenerationNo: generation.GenerationNo, Status: "building", DiagnosticsJson: &diagnostics, RetiredAt: &now,
			Has: &generationstate.StoredGenerationHas{GenerationNo: true, Status: true,
				DiagnosticsJson: true, RetiredAt: true},
		}})
		if err != nil {
			var conflict *xhandler.Conflict
			if errors.As(err, &conflict) {
				return &sdk.Error{Code: sdk.ErrorConflict, Message: "expired generation changed before recovery"}
			}
			return internal(err)
		}
	}
	return nil
}

func (t *Transport) recoverExpiredPublications(ctx context.Context, tx *sql.Tx, generation int64, diagnostics string) error {
	staged, err := t.readStagedPublications(ctx, tx, generation)
	if err != nil {
		return internal(err)
	}
	for _, candidate := range staged {
		if candidate.PublicationStatus != "pending" && candidate.PublicationStatus != "unpublishing" {
			continue
		}
		current, err := t.readPublicationRow(ctx, tx, candidate.ReportId)
		if err != nil {
			return internal(err)
		}
		if current.DesiredGeneration != generation || current.PublicationStatus != candidate.PublicationStatus {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "staged publication changed before recovery"}
		}
		expected := generation
		row := &publicationrecover.StoredPublication{ReportId: candidate.ReportId,
			DesiredGeneration: &expected, PublicationStatus: current.PublicationStatus,
			FailureJson: &diagnostics,
			Has: &publicationrecover.StoredPublicationHas{ReportId: true,
				DesiredGeneration: true, PublicationStatus: true, FailureJson: true}}
		operation := "fail"
		if current.ActiveGeneration != nil {
			activeGeneration, err := t.readGenerationCatalog(ctx, tx, *current.ActiveGeneration, "")
			if err != nil {
				return internal(err)
			}
			if len(activeGeneration) != 1 || activeGeneration[0].SourceRevision == "" || current.ActiveVersionNo <= 0 {
				return &sdk.Error{Code: sdk.ErrorConflict, Message: "active generation is missing during expired-stage recovery"}
			}
			versions, err := t.readVersionCatalogTx(ctx, tx, versionCatalogRequest{ReportID: candidate.ReportId,
				VersionNo: current.ActiveVersionNo, Limit: 2})
			if err != nil {
				return internal(err)
			}
			if len(versions) != 1 || versions[0].SpecHash == "" {
				return &sdk.Error{Code: sdk.ErrorConflict, Message: "active version is missing during expired-stage recovery"}
			}
			operation = "restore_active"
			row.RuntimeRevision = &activeGeneration[0].SourceRevision
			row.SpecHash = versions[0].SpecHash
			row.Has.RuntimeRevision = true
			row.Has.SpecHash = true
		}
		if err := t.writePublicationRecovery(ctx, tx, operation, row); err != nil {
			var conflict *xhandler.Conflict
			if errors.As(err, &conflict) {
				return &sdk.Error{Code: sdk.ErrorConflict, Message: "staged publication changed before recovery"}
			}
			return internal(fmt.Errorf("recover expired publication %q: %w", candidate.ReportId, err))
		}
	}
	return nil
}
