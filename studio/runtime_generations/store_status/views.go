package store_status

import (
	json "encoding/json"
	time "time"
)

// StoredGeneration is generated canonical view metadata for generation.
type StoredGeneration struct {
	GenerationNo    int64           `sqlx:"generation_no"`
	Status          string          `sqlx:"status"`
	ReportCount     int             `sqlx:"report_count"`
	ActivatedAt     *time.Time      `sqlx:"activated_at"`
	DiagnosticsJson json.RawMessage `sqlx:"diagnostics_json,enc=JSON"`
}
