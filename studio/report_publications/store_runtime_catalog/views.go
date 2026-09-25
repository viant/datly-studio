package store_runtime_catalog

import (
	json "encoding/json"
	time "time"
)

// PublishedComponent is generated canonical view metadata for publication.
type PublishedComponent struct {
	ReportId                       string          `sqlx:"report_id"`
	ReportNamespace                string          `sqlx:"report_namespace"`
	ReportSlug                     string          `sqlx:"report_slug"`
	ReportTitle                    string          `sqlx:"report_title"`
	ReportDescription              *string         `sqlx:"report_description"`
	ReportOwnerId                  string          `sqlx:"report_owner_id"`
	ReportStatus                   string          `sqlx:"report_status"`
	ReportConnector                string          `sqlx:"report_connector"`
	ReportComponentScope           string          `sqlx:"report_component_scope"`
	ReportComponentName            string          `sqlx:"report_component_name"`
	ReportDraftVersion             *int            `sqlx:"report_draft_version"`
	ReportEtag                     int64           `sqlx:"report_etag"`
	ReportCreatedAt                time.Time       `sqlx:"report_created_at"`
	ReportUpdatedAt                time.Time       `sqlx:"report_updated_at"`
	PublicationActiveVersionNo     int             `sqlx:"publication_active_version_no"`
	PublicationDesiredGeneration   int64           `sqlx:"publication_desired_generation"`
	PublicationActiveGeneration    *int64          `sqlx:"publication_active_generation"`
	PublicationStatus              string          `sqlx:"publication_status"`
	PublicationRuntimeRevision     *string         `sqlx:"publication_runtime_revision"`
	PublicationPublishedAt         *time.Time      `sqlx:"publication_published_at"`
	VersionState                   string          `sqlx:"version_state"`
	VersionAuthoringMode           string          `sqlx:"version_authoring_mode"`
	VersionAuthoredSql             *string         `sqlx:"version_authored_sql"`
	VersionAuthoredDql             *string         `sqlx:"version_authored_dql"`
	VersionComponentSpecJson       json.RawMessage `sqlx:"version_component_spec_json,enc=JSON"`
	VersionSpecFormatVersion       string          `sqlx:"version_spec_format_version"`
	VersionSpecHash                string          `sqlx:"version_spec_hash"`
	VersionGeneratedDql            *string         `sqlx:"version_generated_dql"`
	VersionDqlExportLimitsJson     json.RawMessage `sqlx:"version_dql_export_limits_json,enc=JSON"`
	VersionTypeManifestJson        json.RawMessage `sqlx:"version_type_manifest_json,enc=JSON"`
	VersionResourceManifestJson    json.RawMessage `sqlx:"version_resource_manifest_json,enc=JSON"`
	VersionComponentDescriptorJson json.RawMessage `sqlx:"version_component_descriptor_json,enc=JSON"`
	VersionCompileStatus           string          `sqlx:"version_compile_status"`
	VersionCompileDiagnosticsJson  json.RawMessage `sqlx:"version_compile_diagnostics_json,enc=JSON"`
	VersionDatlyVersion            string          `sqlx:"version_datly_version"`
	VersionCompilerVersion         string          `sqlx:"version_compiler_version"`
	VersionSourceRevision          int64           `sqlx:"version_source_revision"`
	VersionNotes                   *string         `sqlx:"version_notes"`
	VersionCreatedBy               string          `sqlx:"version_created_by"`
	VersionCreatedAt               time.Time       `sqlx:"version_created_at"`
	VersionValidatedAt             *time.Time      `sqlx:"version_validated_at"`
	VersionPublishedAt             *time.Time      `sqlx:"version_published_at"`
	ReportLive                     bool            `sqlx:"report_live"`
}
