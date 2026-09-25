package store_read

import (
	json "encoding/json"
	time "time"
)

// StoredWarmupRun is generated canonical view metadata for warmup_run.
type StoredWarmupRun struct {
	RunId           string          `sqlx:"run_id"`
	ReportId        string          `sqlx:"report_id"`
	VersionNo       int             `sqlx:"version_no"`
	SourceRevision  int64           `sqlx:"source_revision"`
	SpecHash        string          `sqlx:"spec_hash"`
	PlanKey         string          `sqlx:"plan_key"`
	Status          string          `sqlx:"status"`
	RequestedBy     string          `sqlx:"requested_by"`
	RequestedAt     time.Time       `sqlx:"requested_at"`
	CreatedAt       *time.Time      `sqlx:"created_at"`
	CreatedBy       *string         `sqlx:"created_by"`
	UpdatedAt       *time.Time      `sqlx:"updated_at"`
	UpdatedBy       *string         `sqlx:"updated_by"`
	StartedAt       *time.Time      `sqlx:"started_at"`
	CompletedAt     *time.Time      `sqlx:"completed_at"`
	PlannedCases    int             `sqlx:"planned_cases"`
	CompletedCases  int             `sqlx:"completed_cases"`
	MaxCases        *int            `sqlx:"max_cases"`
	RowLimit        *int            `sqlx:"row_limit"`
	Entries         int             `sqlx:"entries"`
	DurationNs      int64           `sqlx:"duration_ns"`
	TargetJson      json.RawMessage `sqlx:"target_json,enc=JSON"`
	DiagnosticsJson json.RawMessage `sqlx:"diagnostics_json,enc=JSON"`
}
