package store_draft_pointer

import (
	time "time"
)

// DraftPointer is generated canonical view metadata for report.
type DraftPointer struct {
	Id                  string           `sqlx:"id,primaryKey"`
	CurrentDraftVersion *int             `sqlx:"current_draft_version"`
	Etag                *int64           `writer:"concurrency" sqlx:"etag"`
	UpdatedAt           *time.Time       `sqlx:"updated_at"`
	Has                 *DraftPointerHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"DraftPointerHas"`
}

type DraftPointerHas struct {
	Id                  bool
	CurrentDraftVersion bool
	Etag                bool
	UpdatedAt           bool
}

// CurrentReportView is generated canonical view metadata for report.
type CurrentReportView struct {
	Id                  string     `sqlx:"id,primaryKey"`
	CurrentDraftVersion *int       `sqlx:"current_draft_version"`
	Etag                *int64     `sqlx:"etag"`
	UpdatedAt           *time.Time `sqlx:"updated_at"`
}

type ReportKeysRow struct {
	Id string `sqlx:"id"`
}
