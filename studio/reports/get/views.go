package get

import (
	time "time"
)

// Report is generated canonical view metadata for report.
type Report struct {
	Id                   string    `sqlx:"id"`
	Namespace            string    `sqlx:"namespace"`
	Slug                 string    `sqlx:"slug"`
	Title                string    `sqlx:"title"`
	Description          *string   `json:"description,omitempty" sqlx:"description"`
	OwnerId              string    `sqlx:"owner_id"`
	OwnerPackage         string    `sqlx:"owner_package"`
	Status               string    `sqlx:"status"`
	DefaultConnectorName string    `sqlx:"default_connector_name"`
	ComponentScope       string    `sqlx:"component_scope"`
	ComponentName        string    `sqlx:"component_name"`
	CurrentDraftVersion  *int      `json:"currentDraftVersion,omitempty" sqlx:"current_draft_version"`
	Etag                 int64     `sqlx:"etag"`
	CreatedAt            time.Time `sqlx:"created_at"`
	UpdatedAt            time.Time `sqlx:"updated_at"`
}
