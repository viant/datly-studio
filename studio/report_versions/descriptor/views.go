package descriptor

import (
	json "encoding/json"
)

// VersionDescriptor is generated canonical view metadata for version.
type VersionDescriptor struct {
	ReportId             string          `sqlx:"report_id"`
	VersionNo            int             `sqlx:"version_no"`
	ComponentSpecJson    json.RawMessage `sqlx:"component_spec_json,enc=JSON"`
	TypeManifestJson     json.RawMessage `sqlx:"type_manifest_json,enc=JSON"`
	ResourceManifestJson json.RawMessage `sqlx:"resource_manifest_json,enc=JSON"`
}
