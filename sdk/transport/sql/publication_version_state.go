package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/viant/datly-studio/sdk"
	stored "github.com/viant/datly-studio/studio/report_versions/store_state"
	xhandler "github.com/viant/xdatly/handler"
)

func (t *Transport) activateVersionState(ctx context.Context, tx *sql.Tx, reportID string, targetVersionNo int, now time.Time) error {
	target, err := t.readVersionCatalogTx(ctx, tx, versionCatalogRequest{ReportID: reportID, VersionNo: targetVersionNo, Limit: 2})
	if err != nil {
		return internal(err)
	}
	if len(target) != 1 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "target publication version changed before activation"}
	}
	const pageSize = 500
	var older []*sdk.ReportVersion
	for offset := 0; ; {
		page, readErr := t.readVersionCatalogTx(ctx, tx, versionCatalogRequest{ReportID: reportID, State: "published", Limit: pageSize, Offset: offset})
		if readErr != nil {
			return internal(readErr)
		}
		for _, version := range page {
			if version.VersionNo != targetVersionNo {
				older = append(older, version)
			}
		}
		if len(page) < pageSize {
			break
		}
		offset += len(page)
	}
	if len(older) > 0 {
		rows := make([]*stored.StoredVersion, 0, len(older))
		for _, version := range older {
			rows = append(rows, &stored.StoredVersion{ReportId: reportID, VersionNo: version.VersionNo, State: "published",
				Has: &stored.StoredVersionHas{ReportId: true, VersionNo: true, State: true}})
		}
		if err := t.writeVersionState(ctx, tx, "supersede", targetVersionNo, rows); err != nil {
			return versionStateWriteError(err)
		}
	}
	current := target[0]
	if current.VersionNo != targetVersionNo || current.ReportID != reportID {
		return internal(fmt.Errorf("version catalog returned mismatched target"))
	}
	row := &stored.StoredVersion{ReportId: reportID, VersionNo: targetVersionNo, State: current.State, PublishedAt: &now,
		Has: &stored.StoredVersionHas{ReportId: true, VersionNo: true, State: true, PublishedAt: true}}
	if err := t.writeVersionState(ctx, tx, "publish", targetVersionNo, []*stored.StoredVersion{row}); err != nil {
		return versionStateWriteError(err)
	}
	return nil
}

func (t *Transport) unpublishVersionState(ctx context.Context, tx *sql.Tx, reportID string, activeVersionNo int) error {
	if activeVersionNo <= 0 {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "active version is missing during unpublish"}
	}
	const pageSize = 500
	var published []*sdk.ReportVersion
	for offset := 0; ; {
		page, err := t.readVersionCatalogTx(ctx, tx, versionCatalogRequest{ReportID: reportID, State: "published", Limit: pageSize, Offset: offset})
		if err != nil {
			return internal(err)
		}
		published = append(published, page...)
		if len(page) < pageSize {
			break
		}
		offset += len(page)
	}
	if len(published) == 0 {
		return nil
	}
	rows := make([]*stored.StoredVersion, 0, len(published))
	for _, version := range published {
		rows = append(rows, &stored.StoredVersion{ReportId: reportID, VersionNo: version.VersionNo, State: "published",
			Has: &stored.StoredVersionHas{ReportId: true, VersionNo: true, State: true}})
	}
	if err := t.writeVersionState(ctx, tx, "unpublish", activeVersionNo, rows); err != nil {
		return versionStateWriteError(err)
	}
	return nil
}

func versionStateWriteError(err error) error {
	var conflict *xhandler.Conflict
	if errors.As(err, &conflict) {
		return &sdk.Error{Code: sdk.ErrorConflict, Message: "version state changed before publication"}
	}
	return internal(err)
}
