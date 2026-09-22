package writer

import (
	json "encoding/json"
	time "time"
)

// ReportPublication is generated canonical view metadata for publication.
type ReportPublication struct {
	FailureJson json.RawMessage `sqlx:"failure_json,enc=JSON"`
	ReportId *string `sqlx:"report_id,primaryKey" validate:"required"`
	ActiveVersionNo *int `validate:"required" sqlx:"active_version_no"`
	DesiredGeneration *int `validate:"required" sqlx:"desired_generation"`
	PublicationStatus *string `validate:"required,choice(pending,active,failed,unpublishing)" sqlx:"publication_status"`
	SpecHash *string `validate:"required" sqlx:"spec_hash"`
	PublishedBy *string `validate:"required" sqlx:"published_by"`
	PublishedAt *time.Time `validate:"required" sqlx:"published_at"`
	ActiveGeneration *int `sqlx:"active_generation"`
	RuntimeRevision *string `sqlx:"runtime_revision"`
	ActivatedAt *time.Time `sqlx:"activated_at"`
	DesiredVersionNo *int `sqlx:"desired_version_no"`
	Has *ReportPublicationHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"ReportPublicationHas"`
}

type ReportPublicationHas struct {
	FailureJson bool
	ReportId bool
	ActiveVersionNo bool
	DesiredGeneration bool
	PublicationStatus bool
	SpecHash bool
	PublishedBy bool
	PublishedAt bool
	ActiveGeneration bool
	RuntimeRevision bool
	ActivatedAt bool
	DesiredVersionNo bool
}

// CurrentPublicationView is generated canonical view metadata for publication.
type CurrentPublicationView struct {
	FailureJson json.RawMessage `sqlx:"failure_json,enc=JSON"`
	ReportId *string `sqlx:"report_id,primaryKey" validate:"required"`
	ActiveVersionNo *int `validate:"required" sqlx:"active_version_no"`
	DesiredGeneration *int `validate:"required" sqlx:"desired_generation"`
	PublicationStatus *string `validate:"required,choice(pending,active,failed,unpublishing)" sqlx:"publication_status"`
	SpecHash *string `validate:"required" sqlx:"spec_hash"`
	PublishedBy *string `validate:"required" sqlx:"published_by"`
	PublishedAt *time.Time `validate:"required" sqlx:"published_at"`
	ActiveGeneration *int `sqlx:"active_generation"`
	RuntimeRevision *string `sqlx:"runtime_revision"`
	ActivatedAt *time.Time `sqlx:"activated_at"`
	DesiredVersionNo *int `sqlx:"desired_version_no"`
}

type PublicationKeysRow struct {
	ReportId *string `sqlx:"report_id"`
}
