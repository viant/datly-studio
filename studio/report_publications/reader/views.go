package reader

import (
	json "encoding/json"
	time "time"
)

// ReportPublication is generated canonical view metadata for publication.
type ReportPublication struct {
	FailureJson json.RawMessage `sqlx:"failure_json,enc=JSON"`
	ReportId *string `sqlx:"report_id"`
	ActiveVersionNo *int `sqlx:"active_version_no"`
	DesiredVersionNo *int `sqlx:"desired_version_no"`
	DesiredGeneration *int `sqlx:"desired_generation"`
	ActiveGeneration *int `sqlx:"active_generation"`
	PublicationStatus *string `sqlx:"publication_status"`
	RuntimeRevision *string `sqlx:"runtime_revision"`
	SpecHash *string `sqlx:"spec_hash"`
	PublishedBy *string `sqlx:"published_by"`
	PublishedAt *time.Time `sqlx:"published_at"`
	ActivatedAt *time.Time `sqlx:"activated_at"`
}
