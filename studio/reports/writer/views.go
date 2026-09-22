package writer

import (
	time "time"
)

// Report is generated canonical view metadata for report.
type Report struct {
	Namespace string `validate:"required" sqlx:"namespace"`
	Id *string `sqlx:"id,primaryKey" validate:"required"`
	Slug *string `validate:"required" sqlx:"slug"`
	Title *string `validate:"required" sqlx:"title"`
	OwnerId *string `validate:"required" sqlx:"owner_id"`
	Status *string `validate:"required,choice(draft,active,disabled,archived)" sqlx:"status"`
	DefaultConnectorName *string `validate:"required" sqlx:"default_connector_name"`
	ComponentScope *string `validate:"required" invariant:"ComponentIdentity" sqlx:"component_scope"`
	ComponentName *string `validate:"required" invariant:"ComponentIdentity" sqlx:"component_name"`
	ShouldDelete bool `sqlx:"-" writer:"delete"`
	Etag *int `writer:"concurrency" sqlx:"etag"`
	Description *string `sqlx:"description"`
	CurrentDraftVersion *int `sqlx:"current_draft_version"`
	CreatedAt *time.Time `sqlx:"created_at"`
	UpdatedAt *time.Time `sqlx:"updated_at"`
	DeletedAt *time.Time `sqlx:"deleted_at"`
	Has *ReportHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"ReportHas"`
}

type ReportHas struct {
	Namespace bool
	Id bool
	Slug bool
	Title bool
	OwnerId bool
	Status bool
	DefaultConnectorName bool
	ComponentScope bool
	ComponentName bool
	ShouldDelete bool
	Etag bool
	Description bool
	CurrentDraftVersion bool
	CreatedAt bool
	UpdatedAt bool
	DeletedAt bool
}

// CurrentReportView is generated canonical view metadata for report.
type CurrentReportView struct {
	Namespace string `validate:"required" sqlx:"namespace"`
	Id *string `sqlx:"id,primaryKey" validate:"required"`
	Slug *string `validate:"required" sqlx:"slug"`
	Title *string `validate:"required" sqlx:"title"`
	OwnerId *string `validate:"required" sqlx:"owner_id"`
	Status *string `validate:"required,choice(draft,active,disabled,archived)" sqlx:"status"`
	DefaultConnectorName *string `validate:"required" sqlx:"default_connector_name"`
	ComponentScope *string `validate:"required" invariant:"ComponentIdentity" sqlx:"component_scope"`
	ComponentName *string `validate:"required" invariant:"ComponentIdentity" sqlx:"component_name"`
	Etag *int `sqlx:"etag"`
	Description *string `sqlx:"description"`
	CurrentDraftVersion *int `sqlx:"current_draft_version"`
	CreatedAt *time.Time `sqlx:"created_at"`
	UpdatedAt *time.Time `sqlx:"updated_at"`
	DeletedAt *time.Time `sqlx:"deleted_at"`
}

type ReportKeysRow struct {
	Id *string `sqlx:"id"`
}
