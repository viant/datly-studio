package store_status

import (
	time "time"
)

// StoredPublication is generated canonical view metadata for publication.
type StoredPublication struct {
	ReportId          string     `sqlx:"report_id"`
	ActiveVersionNo   int        `sqlx:"active_version_no"`
	DesiredVersionNo  *int       `sqlx:"desired_version_no"`
	DesiredGeneration int64      `sqlx:"desired_generation"`
	ActiveGeneration  *int64     `sqlx:"active_generation"`
	PublicationStatus string     `sqlx:"publication_status"`
	RuntimeRevision   *string    `sqlx:"runtime_revision"`
	SpecHash          string     `sqlx:"spec_hash"`
	PublishedBy       string     `sqlx:"published_by"`
	PublishedAt       *time.Time `sqlx:"published_at"`
	ActivatedAt       *time.Time `sqlx:"activated_at"`
}
