package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/viant/datly-studio/sdk"
	publicationstore "github.com/viant/datly-studio/sdk/transport/sql/internal/publications"
	publicationrepoint "github.com/viant/datly-studio/studio/report_publications/store_repoint"
	reportconfig "github.com/viant/datly-studio/studio/reports/store_config"
	generationstate "github.com/viant/datly-studio/studio/runtime_generations/store_state"
	xhandler "github.com/viant/xdatly/handler"
)

func (t *Transport) repointActivePublications(ctx context.Context, tx *sql.Tx, excludeReportID string, generation int64) (int, error) {
	others, err := publicationstore.ReadOtherActive(ctx, t.DB, tx, excludeReportID)
	if err != nil {
		return 0, internal(err)
	}
	if len(others) == 0 {
		return 0, nil
	}
	rows := make([]*publicationrepoint.StoredPublication, 0, len(others))
	for _, other := range others {
		rows = append(rows, &publicationrepoint.StoredPublication{ReportId: other.ReportId,
			ActiveGeneration: other.ActiveGeneration,
			Has:              &publicationrepoint.StoredPublicationHas{ReportId: true, ActiveGeneration: true}})
	}
	if err = publicationstore.WriteRepoint(ctx, t.DB, tx, excludeReportID, generation, rows); err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return 0, &sdk.Error{Code: sdk.ErrorConflict, Message: "other active publication changed before activation"}
		}
		return 0, internal(err)
	}
	return len(others), nil
}

func (t *Transport) activateGenerationState(ctx context.Context, tx *sql.Tx, generation int64, reportCount int, now time.Time) error {
	err := t.writeGenerationState(ctx, tx, "activate", generation, []*generationstate.StoredGeneration{{
		GenerationNo: generation, Status: "building", ReportCount: &reportCount, ActivatedAt: &now,
		Has: &generationstate.StoredGenerationHas{GenerationNo: true, Status: true,
			ReportCount: true, ActivatedAt: true},
	}})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "building generation changed before activation"}
		}
		return internal(err)
	}
	retired, err := t.readOtherActiveGenerations(ctx, tx, generation)
	if err != nil {
		return internal(err)
	}
	if len(retired) == 0 {
		return nil
	}
	rows := make([]*generationstate.StoredGeneration, 0, len(retired))
	for _, other := range retired {
		rows = append(rows, &generationstate.StoredGeneration{GenerationNo: other.GenerationNo,
			Status: "active", RetiredAt: &now,
			Has: &generationstate.StoredGenerationHas{GenerationNo: true, Status: true, RetiredAt: true}})
	}
	if err = t.writeGenerationState(ctx, tx, "retire", generation, rows); err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "other active generation changed before retirement"}
		}
		return internal(err)
	}
	return nil
}

func (t *Transport) updateReportStatusTx(ctx context.Context, tx *sql.Tx, reportID, status string, now time.Time) error {
	reports, err := t.readReportCatalogTx(ctx, tx, reportCatalogRequest{ID: reportID, Limit: 2, Unscoped: true})
	if err != nil {
		return internal(err)
	}
	if len(reports) != 1 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "report is absent or deleted during activation"}
	}
	report := reports[0]
	etag := report.ETag
	err = t.writeReportConfig(ctx, tx, &reportconfig.StoredReport{
		Id: report.ID, Namespace: report.Namespace, Slug: report.Slug, Title: report.Title,
		Description: namespaceOptionalDescription(report.Description), OwnerId: report.OwnerID,
		Status: status, DefaultConnectorName: report.DefaultConnectorName,
		ComponentScope: report.ComponentScope, ComponentName: report.ComponentName,
		CurrentDraftVersion: report.CurrentDraftVersion, Etag: &etag, UpdatedAt: &now,
		Has: &reportconfig.StoredReportHas{Id: true, Namespace: true, Slug: true,
			Title: true, Description: true, OwnerId: true, Status: true,
			DefaultConnectorName: true, ComponentScope: true, ComponentName: true,
			CurrentDraftVersion: true, Etag: true, UpdatedAt: true},
	})
	if err != nil {
		var conflict *xhandler.Conflict
		if errors.As(err, &conflict) {
			return &sdk.Error{Code: sdk.ErrorConflict, Message: "report changed before activation"}
		}
		return internal(err)
	}
	return nil
}
