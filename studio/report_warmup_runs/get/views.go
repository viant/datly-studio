package get

import (
	json "encoding/json"
	time "time"
)

// WarmupRow is generated canonical view metadata for warmup_run.
type WarmupRow struct {
	RunId           string          `sqlx:"run_id,required=true,primaryKey=true"`
	ReportId        string          `sqlx:"report_id,refTable=component_versions,refColumn=report_id,required=true"`
	VersionNo       int             `sqlx:"version_no,refTable=component_versions,refColumn=version_no,required=true"`
	SourceRevision  int64           `sqlx:"source_revision,required=true"`
	SpecHash        string          `sqlx:"spec_hash,required=true"`
	PlanKey         string          `sqlx:"plan_key,required=true"`
	Status          string          `sqlx:"status,required=true"`
	RequestedBy     string          `sqlx:"requested_by,required=true"`
	RequestedAt     time.Time       `sqlx:"requested_at,required=true"`
	CreatedAt       *time.Time      `sqlx:"created_at"`
	CreatedBy       *string         `sqlx:"created_by"`
	UpdatedAt       *time.Time      `sqlx:"updated_at"`
	UpdatedBy       *string         `sqlx:"updated_by"`
	StartedAt       *time.Time      `sqlx:"started_at"`
	CompletedAt     *time.Time      `sqlx:"completed_at"`
	PlannedCases    int             `sqlx:"planned_cases,required=true"`
	CompletedCases  int             `sqlx:"completed_cases,required=true"`
	MaxCases        *int            `sqlx:"max_cases"`
	RowLimit        *int            `sqlx:"row_limit"`
	Entries         int             `sqlx:"entries,required=true"`
	DurationNs      int64           `sqlx:"duration_ns,required=true"`
	TargetJson      json.RawMessage `sqlx:"target_json,enc=JSON,required=true"`
	DiagnosticsJson json.RawMessage `sqlx:"diagnostics_json,enc=JSON"`
}
