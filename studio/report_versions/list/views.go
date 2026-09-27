package list

import (
	json "encoding/json"
	time "time"
)

// Version is generated canonical view metadata for version.
type Version struct {
	ReportId                string          `sqlx:"report_id"`
	VersionNo               int             `sqlx:"version_no"`
	State                   string          `sqlx:"state"`
	AuthoringMode           string          `sqlx:"authoring_mode"`
	AuthoredSql             *string         `json:"authoredSql,omitempty" sqlx:"authored_sql"`
	AuthoredDql             *string         `json:"authoredDql,omitempty" sqlx:"authored_dql"`
	ComponentSpecJson       json.RawMessage `sqlx:"component_spec_json,enc=JSON" json:"componentSpec,omitempty"`
	DqlExportLimitsJson     json.RawMessage `sqlx:"dql_export_limits_json,enc=JSON" json:"dqlExportLimits,omitempty"`
	TypeManifestJson        json.RawMessage `sqlx:"type_manifest_json,enc=JSON" json:"typeManifest,omitempty"`
	ResourceManifestJson    json.RawMessage `sqlx:"resource_manifest_json,enc=JSON" json:"resourceManifest,omitempty"`
	ComponentDescriptorJson json.RawMessage `sqlx:"component_descriptor_json,enc=JSON" json:"componentDescriptor,omitempty"`
	SpecFormatVersion       string          `sqlx:"spec_format_version"`
	SpecHash                string          `sqlx:"spec_hash"`
	GeneratedDql            *string         `json:"generatedDql,omitempty" sqlx:"generated_dql"`
	CompileStatus           string          `sqlx:"compile_status"`
	CompileDiagnosticsJson  json.RawMessage `sqlx:"compile_diagnostics_json,enc=JSON" json:"compileDiagnostics,omitempty"`
	DatlyVersion            string          `sqlx:"datly_version"`
	CompilerVersion         string          `sqlx:"compiler_version"`
	SourceRevision          int64           `sqlx:"source_revision"`
	Notes                   *string         `json:"notes,omitempty" sqlx:"notes"`
	CreatedBy               string          `sqlx:"created_by"`
	CreatedAt               time.Time       `sqlx:"created_at"`
	ValidatedAt             *time.Time      `json:"validatedAt,omitempty" sqlx:"validated_at"`
	PublishedAt             *time.Time      `json:"publishedAt,omitempty" sqlx:"published_at"`
}
