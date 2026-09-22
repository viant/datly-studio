package reader

import (
	json "encoding/json"
	time "time"
)

// RuntimeGeneration is generated canonical view metadata for generation.
type RuntimeGeneration struct {
	BuildManifestJson json.RawMessage `sqlx:"build_manifest_json,enc=JSON"`
	DiagnosticsJson json.RawMessage `sqlx:"diagnostics_json,enc=JSON"`
	GenerationNo *int `sqlx:"generation_no"`
	SourceRevision *string `sqlx:"source_revision"`
	Status *string `sqlx:"status"`
	ReportCount *int `sqlx:"report_count"`
	RequestedBy *string `sqlx:"requested_by"`
	RequestedAt *time.Time `sqlx:"requested_at"`
	ActivatedAt *time.Time `sqlx:"activated_at"`
	RetiredAt *time.Time `sqlx:"retired_at"`
}
