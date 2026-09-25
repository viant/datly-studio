package store_preview_definition

import (
	json "encoding/json"
)

// PreviewDefinition is generated canonical view metadata for definition.
type PreviewDefinition struct {
	ReportId             string          `sqlx:"report_id"`
	VersionNo            int             `sqlx:"version_no"`
	ComponentScope       string          `sqlx:"component_scope"`
	ComponentName        string          `sqlx:"component_name"`
	DefaultConnectorName string          `sqlx:"default_connector_name"`
	Driver               string          `sqlx:"driver"`
	SecretRef            string          `sqlx:"secret_ref"`
	OptionsJson          json.RawMessage `sqlx:"options_json,enc=JSON"`
	SourceRevision       int64           `sqlx:"source_revision"`
	SpecHash             string          `sqlx:"spec_hash"`
	GeneratedDql         string          `sqlx:"generated_dql"`
	AuthoredDql          string          `sqlx:"authored_dql"`
	DsnTemplate          *string         `sqlx:"dsn_template"`
}
