package store_insert

import (
	json "encoding/json"
	time "time"
)

// StoredGeneration is generated canonical view metadata for generation.
type StoredGeneration struct {
	GenerationNo      int64                `sqlx:"generation_no,primaryKey"`
	SourceRevision    string               `sqlx:"source_revision"`
	Status            string               `sqlx:"status"`
	ReportCount       int                  `sqlx:"report_count"`
	BuildManifestJson json.RawMessage      `sqlx:"build_manifest_json,enc=JSON"`
	RequestedBy       string               `sqlx:"requested_by"`
	RequestedAt       time.Time            `sqlx:"requested_at"`
	Has               *StoredGenerationHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredGenerationHas"`
}

type StoredGenerationHas struct {
	GenerationNo      bool
	SourceRevision    bool
	Status            bool
	ReportCount       bool
	BuildManifestJson bool
	RequestedBy       bool
	RequestedAt       bool
}
