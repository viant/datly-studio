package store_stage

import (
	json "encoding/json"
	time "time"
)

// StoredPublication is generated canonical view metadata for publication.
type StoredPublication struct {
	ReportId          string                `sqlx:"report_id,primaryKey"`
	DesiredVersionNo  *int                  `sqlx:"desired_version_no"`
	DesiredGeneration *int64                `writer:"concurrency" sqlx:"desired_generation"`
	PublicationStatus string                `sqlx:"publication_status"`
	RuntimeRevision   *string               `sqlx:"runtime_revision"`
	SpecHash          string                `sqlx:"spec_hash"`
	PublishedBy       string                `sqlx:"published_by"`
	PublishedAt       *time.Time            `sqlx:"published_at"`
	FailureJson       json.RawMessage       `sqlx:"failure_json,enc=JSON"`
	Has               *StoredPublicationHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredPublicationHas"`
}

type StoredPublicationHas struct {
	ReportId          bool
	DesiredVersionNo  bool
	DesiredGeneration bool
	PublicationStatus bool
	RuntimeRevision   bool
	SpecHash          bool
	PublishedBy       bool
	PublishedAt       bool
	FailureJson       bool
}

// CurrentPublicationView is generated canonical view metadata for publication.
type CurrentPublicationView struct {
	ReportId          string          `sqlx:"report_id,primaryKey"`
	DesiredVersionNo  *int            `sqlx:"desired_version_no"`
	DesiredGeneration *int64          `sqlx:"desired_generation"`
	PublicationStatus string          `sqlx:"publication_status"`
	RuntimeRevision   *string         `sqlx:"runtime_revision"`
	SpecHash          string          `sqlx:"spec_hash"`
	PublishedBy       string          `sqlx:"published_by"`
	PublishedAt       *time.Time      `sqlx:"published_at"`
	FailureJson       json.RawMessage `sqlx:"failure_json,enc=JSON"`
}

type PublicationKeysRow struct {
	ReportId string `sqlx:"report_id"`
}
