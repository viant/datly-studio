package store_catalog

import (
	time "time"
)

// StoredReport is generated canonical view metadata for report.
type StoredReport struct {
	Id                   string    `sqlx:"id"`
	Namespace            string    `sqlx:"namespace"`
	Slug                 string    `sqlx:"slug"`
	Title                string    `sqlx:"title"`
	Description          *string   `sqlx:"description"`
	OwnerId              string    `sqlx:"owner_id"`
	Status               string    `sqlx:"status"`
	DefaultConnectorName string    `sqlx:"default_connector_name"`
	ComponentScope       string    `sqlx:"component_scope"`
	ComponentName        string    `sqlx:"component_name"`
	CurrentDraftVersion  *int      `sqlx:"current_draft_version"`
	Etag                 int64     `sqlx:"etag"`
	CreatedAt            time.Time `sqlx:"created_at"`
	UpdatedAt            time.Time `sqlx:"updated_at"`
}
