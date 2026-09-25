package store_insert

import (
	json "encoding/json"
	time "time"
)

// StoredPublication is generated canonical view metadata for publication.
type StoredPublication struct {
	ReportId          string                `sqlx:"report_id,primaryKey"`
	ActiveVersionNo   int                   `sqlx:"active_version_no"`
	DesiredVersionNo  *int                  `sqlx:"desired_version_no"`
	DesiredGeneration int64                 `sqlx:"desired_generation"`
	ActiveGeneration  *int64                `sqlx:"active_generation"`
	PublicationStatus string                `sqlx:"publication_status"`
	RuntimeRevision   *string               `sqlx:"runtime_revision"`
	SpecHash          string                `sqlx:"spec_hash"`
	PublishedBy       string                `sqlx:"published_by"`
	PublishedAt       time.Time             `sqlx:"published_at"`
	ActivatedAt       *time.Time            `sqlx:"activated_at"`
	FailureJson       json.RawMessage       `sqlx:"failure_json,enc=JSON"`
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
