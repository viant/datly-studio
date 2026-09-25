package reader

import (
	json "encoding/json"
	time "time"
)

// WarmupRun is generated canonical view metadata for warmup_run.
type WarmupRun struct {
	TargetJson      json.RawMessage `sqlx:"target_json,enc=JSON"`
	DiagnosticsJson json.RawMessage `sqlx:"diagnostics_json,enc=JSON"`
	RunId           any             `sqlx:"run_id"`
	ReportId        any             `sqlx:"report_id"`
	VersionNo       *int            `sqlx:"version_no"`
	SourceRevision  *int            `sqlx:"source_revision"`
	SpecHash        any             `sqlx:"spec_hash"`
	PlanKey         any             `sqlx:"plan_key"`
	Status          any             `sqlx:"status"`
	RequestedBy     any             `sqlx:"requested_by"`
	CacheName       any             `sqlx:"cache_name"`
	CacheProvider   any             `sqlx:"cache_provider"`
	ConnectorName   any             `sqlx:"connector_name"`
	IndexColumn     any             `sqlx:"index_column"`
	PlannedCases    *int            `sqlx:"planned_cases"`
	CompletedCases  *int            `sqlx:"completed_cases"`
	MaxCases        *int            `sqlx:"max_cases"`
	RowLimit        *int            `sqlx:"row_limit"`
	Entries         *int            `sqlx:"entries"`
	DurationNs      *int            `sqlx:"duration_ns"`
	RequestedAt     *time.Time      `sqlx:"requested_at"`
	StartedAt       *time.Time      `sqlx:"started_at"`
	CompletedAt     *time.Time      `sqlx:"completed_at"`
}
