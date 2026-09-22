package writer

import (
	json "encoding/json"
	time "time"
)

// RuntimeGeneration is generated canonical view metadata for generation.
type RuntimeGeneration struct {
	BuildManifestJson json.RawMessage `sqlx:"build_manifest_json,enc=JSON,required=true" validate:"required"`
	DiagnosticsJson json.RawMessage `sqlx:"diagnostics_json,enc=JSON"`
	GenerationNo *int `sqlx:"generation_no,primaryKey"`
	SourceRevision *string `validate:"required" sqlx:"source_revision,required=true"`
	Status *string `validate:"required,choice(building,active,failed,retired)" sqlx:"status,required=true"`
	RequestedBy *string `validate:"required" sqlx:"requested_by,required=true"`
	RequestedAt *time.Time `validate:"required" sqlx:"requested_at,required=true"`
	ReportCount *int `sqlx:"report_count,required=true"`
	ActivatedAt *time.Time `sqlx:"activated_at"`
	RetiredAt *time.Time `sqlx:"retired_at"`
	Has *RuntimeGenerationHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"RuntimeGenerationHas"`
}

type RuntimeGenerationHas struct {
	BuildManifestJson bool
	DiagnosticsJson bool
	GenerationNo bool
	SourceRevision bool
	Status bool
	RequestedBy bool
	RequestedAt bool
	ReportCount bool
	ActivatedAt bool
	RetiredAt bool
}

// CurrentGenerationView is generated canonical view metadata for generation.
type CurrentGenerationView struct {
	BuildManifestJson json.RawMessage `sqlx:"build_manifest_json,enc=JSON,required=true" validate:"required"`
	DiagnosticsJson json.RawMessage `sqlx:"diagnostics_json,enc=JSON"`
	GenerationNo *int `sqlx:"generation_no,primaryKey"`
	SourceRevision *string `validate:"required" sqlx:"source_revision,required=true"`
	Status *string `validate:"required,choice(building,active,failed,retired)" sqlx:"status,required=true"`
	RequestedBy *string `validate:"required" sqlx:"requested_by,required=true"`
	RequestedAt *time.Time `validate:"required" sqlx:"requested_at,required=true"`
	ReportCount *int `sqlx:"report_count,required=true"`
	ActivatedAt *time.Time `sqlx:"activated_at"`
	RetiredAt *time.Time `sqlx:"retired_at"`
}

type GenerationKeysRow struct {
	GenerationNo *int `sqlx:"generation_no"`
}
