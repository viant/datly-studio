package store_recover

import (
	time "time"
)

// StoredPublication is generated canonical view metadata for publication.
type StoredPublication struct {
	ReportId          string                `sqlx:"report_id,primaryKey"`
	ActiveVersionNo   int                   `sqlx:"active_version_no"`
	DesiredVersionNo  *int                  `sqlx:"desired_version_no"`
	DesiredGeneration *int64                `sqlx:"desired_generation"`
	ActiveGeneration  *int64                `sqlx:"active_generation"`
	PublicationStatus string                `writer:"concurrency" sqlx:"publication_status"`
	RuntimeRevision   *string               `sqlx:"runtime_revision"`
	SpecHash          string                `sqlx:"spec_hash"`
	PublishedBy       string                `sqlx:"published_by"`
	PublishedAt       *time.Time            `sqlx:"published_at"`
	ActivatedAt       *time.Time            `sqlx:"activated_at"`
	FailureJson       *string               `sqlx:"failure_json,enc=RAW"`
	Has               *StoredPublicationHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredPublicationHas"`
}

type StoredPublicationHas struct {
	ReportId          bool
	ActiveVersionNo   bool
	DesiredVersionNo  bool
	DesiredGeneration bool
	ActiveGeneration  bool
	PublicationStatus bool
	RuntimeRevision   bool
	SpecHash          bool
	PublishedBy       bool
	PublishedAt       bool
	ActivatedAt       bool
	FailureJson       bool
}

// CurrentPublicationView is generated canonical view metadata for publication.
type CurrentPublicationView struct {
	ReportId          string     `sqlx:"report_id,primaryKey"`
	ActiveVersionNo   int        `sqlx:"active_version_no"`
	DesiredVersionNo  *int       `sqlx:"desired_version_no"`
	DesiredGeneration *int64     `sqlx:"desired_generation"`
	ActiveGeneration  *int64     `sqlx:"active_generation"`
	PublicationStatus string     `sqlx:"publication_status"`
	RuntimeRevision   *string    `sqlx:"runtime_revision"`
	SpecHash          string     `sqlx:"spec_hash"`
	PublishedBy       string     `sqlx:"published_by"`
	PublishedAt       *time.Time `sqlx:"published_at"`
	ActivatedAt       *time.Time `sqlx:"activated_at"`
	FailureJson       *string    `sqlx:"failure_json,enc=RAW"`
}

type PublicationKeysRow struct {
	ReportId string `sqlx:"report_id"`
}
