package get

import (
	time "time"
)

// Publication is generated canonical view metadata for publication.
type Publication struct {
	ReportId          string     `sqlx:"report_id"`
	ActiveVersionNo   int        `sqlx:"active_version_no"`
	DesiredVersionNo  *int       `sqlx:"desired_version_no"`
	DesiredGeneration int64      `sqlx:"desired_generation"`
	ActiveGeneration  *int64     `sqlx:"active_generation"`
	PublicationStatus string     `sqlx:"publication_status"`
	RuntimeRevision   *string    `sqlx:"runtime_revision"`
	SpecHash          *string    `sqlx:"spec_hash"`
	PublishedAt       *time.Time `sqlx:"published_at"`
}
