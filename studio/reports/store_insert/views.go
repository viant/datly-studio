package store_insert

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
	Etag                 int64            `sqlx:"etag"`
	CreatedAt            time.Time        `sqlx:"created_at"`
	UpdatedAt            time.Time        `sqlx:"updated_at"`
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
	Etag                 bool
	CreatedAt            bool
	UpdatedAt            bool
}
