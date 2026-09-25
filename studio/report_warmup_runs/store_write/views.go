package store_write

import (
	json "encoding/json"
	time "time"
)

// StoredWarmupRun is generated canonical view metadata for warmup_run.
type StoredWarmupRun struct {
	RunId           string              `sqlx:"run_id,primaryKey"`
	ReportId        string              `sqlx:"report_id"`
	VersionNo       int                 `sqlx:"version_no"`
	SourceRevision  int64               `sqlx:"source_revision"`
	SpecHash        string              `sqlx:"spec_hash"`
	PlanKey         string              `sqlx:"plan_key"`
	ActiveKey       *string             `sqlx:"active_key"`
	Status          string              `sqlx:"status"`
	RequestedBy     string              `sqlx:"requested_by"`
	CacheName       *string             `sqlx:"cache_name"`
	CacheProvider   *string             `sqlx:"cache_provider"`
	ConnectorName   *string             `sqlx:"connector_name"`
	IndexColumn     *string             `sqlx:"index_column"`
	PlannedCases    int                 `sqlx:"planned_cases"`
	CompletedCases  int                 `sqlx:"completed_cases"`
	MaxCases        *int                `sqlx:"max_cases"`
	RowLimit        *int                `sqlx:"row_limit"`
	Entries         int                 `sqlx:"entries"`
	DurationNs      int64               `sqlx:"duration_ns"`
	DiagnosticsJson *json.RawMessage    `sqlx:"diagnostics_json,enc=JSON"`
	TargetJson      json.RawMessage     `sqlx:"target_json,enc=JSON"`
	RequestedAt     time.Time           `sqlx:"requested_at"`
	CreatedAt       *time.Time          `sqlx:"created_at"`
	CreatedBy       *string             `sqlx:"created_by"`
	UpdatedAt       *time.Time          `writer:"concurrency" sqlx:"updated_at"`
	UpdatedBy       *string             `sqlx:"updated_by"`
	StartedAt       *time.Time          `sqlx:"started_at"`
	CompletedAt     *time.Time          `sqlx:"completed_at"`
	Has             *StoredWarmupRunHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredWarmupRunHas"`
}

type StoredWarmupRunHas struct {
	RunId           bool
	ReportId        bool
	VersionNo       bool
	SourceRevision  bool
	SpecHash        bool
	PlanKey         bool
	ActiveKey       bool
	Status          bool
	RequestedBy     bool
	CacheName       bool
	CacheProvider   bool
	ConnectorName   bool
	IndexColumn     bool
	PlannedCases    bool
	CompletedCases  bool
	MaxCases        bool
	RowLimit        bool
	Entries         bool
	DurationNs      bool
	DiagnosticsJson bool
	TargetJson      bool
	RequestedAt     bool
	CreatedAt       bool
	CreatedBy       bool
	UpdatedAt       bool
	UpdatedBy       bool
	StartedAt       bool
	CompletedAt     bool
}

// CurrentWarmupRunView is generated canonical view metadata for warmup_run.
type CurrentWarmupRunView struct {
	RunId           string           `sqlx:"run_id,primaryKey"`
	ReportId        string           `sqlx:"report_id"`
	VersionNo       int              `sqlx:"version_no"`
	SourceRevision  int64            `sqlx:"source_revision"`
	SpecHash        string           `sqlx:"spec_hash"`
	PlanKey         string           `sqlx:"plan_key"`
	ActiveKey       *string          `sqlx:"active_key"`
	Status          string           `sqlx:"status"`
	RequestedBy     string           `sqlx:"requested_by"`
	CacheName       *string          `sqlx:"cache_name"`
	CacheProvider   *string          `sqlx:"cache_provider"`
	ConnectorName   *string          `sqlx:"connector_name"`
	IndexColumn     *string          `sqlx:"index_column"`
	PlannedCases    int              `sqlx:"planned_cases"`
	CompletedCases  int              `sqlx:"completed_cases"`
	MaxCases        *int             `sqlx:"max_cases"`
	RowLimit        *int             `sqlx:"row_limit"`
	Entries         int              `sqlx:"entries"`
	DurationNs      int64            `sqlx:"duration_ns"`
	DiagnosticsJson *json.RawMessage `sqlx:"diagnostics_json,enc=JSON"`
	TargetJson      json.RawMessage  `sqlx:"target_json,enc=JSON"`
	RequestedAt     time.Time        `sqlx:"requested_at"`
	CreatedAt       *time.Time       `sqlx:"created_at"`
	CreatedBy       *string          `sqlx:"created_by"`
	UpdatedAt       *time.Time       `sqlx:"updated_at"`
	UpdatedBy       *string          `sqlx:"updated_by"`
	StartedAt       *time.Time       `sqlx:"started_at"`
	CompletedAt     *time.Time       `sqlx:"completed_at"`
}

type WarmupRunKeysRow struct {
	RunId string `sqlx:"run_id"`
}
