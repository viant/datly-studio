package store_config

import (
	time "time"
)

// StoredReport is generated canonical view metadata for report.
type StoredReport struct {
	Id                   string           `sqlx:"id,primaryKey"`
	Namespace            string           `sqlx:"namespace"`
	Slug                 string           `sqlx:"slug"`
	Title                string           `sqlx:"title"`
	Description          *string          `sqlx:"description"`
	OwnerId              string           `sqlx:"owner_id"`
	Status               string           `sqlx:"status"`
	DefaultConnectorName string           `sqlx:"default_connector_name"`
	ComponentScope       string           `sqlx:"component_scope"`
	ComponentName        string           `sqlx:"component_name"`
	CurrentDraftVersion  *int             `sqlx:"current_draft_version"`
	Etag                 *int64           `writer:"concurrency" sqlx:"etag"`
	UpdatedAt            *time.Time       `sqlx:"updated_at"`
	DeletedAt            *time.Time       `sqlx:"deleted_at"`
	Has                  *StoredReportHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredReportHas"`
}

type StoredReportHas struct {
	Id                   bool
	Namespace            bool
	Slug                 bool
	Title                bool
	Description          bool
	OwnerId              bool
	Status               bool
	DefaultConnectorName bool
	ComponentScope       bool
	ComponentName        bool
	CurrentDraftVersion  bool
	Etag                 bool
	UpdatedAt            bool
	DeletedAt            bool
}

// CurrentReportView is generated canonical view metadata for report.
type CurrentReportView struct {
	Id                   string     `sqlx:"id,primaryKey"`
	Namespace            string     `sqlx:"namespace"`
	Slug                 string     `sqlx:"slug"`
	Title                string     `sqlx:"title"`
	Description          *string    `sqlx:"description"`
	OwnerId              string     `sqlx:"owner_id"`
	Status               string     `sqlx:"status"`
	DefaultConnectorName string     `sqlx:"default_connector_name"`
	ComponentScope       string     `sqlx:"component_scope"`
	ComponentName        string     `sqlx:"component_name"`
	CurrentDraftVersion  *int       `sqlx:"current_draft_version"`
	Etag                 *int64     `sqlx:"etag"`
	UpdatedAt            *time.Time `sqlx:"updated_at"`
	DeletedAt            *time.Time `sqlx:"deleted_at"`
}

type ReportKeysRow struct {
	Id string `sqlx:"id"`
}
