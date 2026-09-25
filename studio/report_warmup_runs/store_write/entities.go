package store_write

import (
	json "encoding/json"
	time "time"
)

func (entity *StoredWarmupRun) GetRunId() string {
	return entity.RunId
}
func (entity *StoredWarmupRun) SetRunId(value string) {
	entity.RunId = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.RunId = true
}
func (entity *StoredWarmupRun) GetReportId() string {
	return entity.ReportId
}
func (entity *StoredWarmupRun) SetReportId(value string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.ReportId = true
}
func (entity *StoredWarmupRun) GetVersionNo() int {
	return entity.VersionNo
}
func (entity *StoredWarmupRun) SetVersionNo(value int) {
	entity.VersionNo = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.VersionNo = true
}
func (entity *StoredWarmupRun) GetSourceRevision() int64 {
	return entity.SourceRevision
}
func (entity *StoredWarmupRun) SetSourceRevision(value int64) {
	entity.SourceRevision = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.SourceRevision = true
}
func (entity *StoredWarmupRun) GetSpecHash() string {
	return entity.SpecHash
}
func (entity *StoredWarmupRun) SetSpecHash(value string) {
	entity.SpecHash = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.SpecHash = true
}
func (entity *StoredWarmupRun) GetPlanKey() string {
	return entity.PlanKey
}
func (entity *StoredWarmupRun) SetPlanKey(value string) {
	entity.PlanKey = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.PlanKey = true
}
func (entity *StoredWarmupRun) GetActiveKey() *string {
	return entity.ActiveKey
}
func (entity *StoredWarmupRun) SetActiveKey(value *string) {
	entity.ActiveKey = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.ActiveKey = true
}
func (entity *StoredWarmupRun) GetStatus() string {
	return entity.Status
}
func (entity *StoredWarmupRun) SetStatus(value string) {
	entity.Status = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.Status = true
}
func (entity *StoredWarmupRun) GetRequestedBy() string {
	return entity.RequestedBy
}
func (entity *StoredWarmupRun) SetRequestedBy(value string) {
	entity.RequestedBy = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.RequestedBy = true
}
func (entity *StoredWarmupRun) GetCacheName() *string {
	return entity.CacheName
}
func (entity *StoredWarmupRun) SetCacheName(value *string) {
	entity.CacheName = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.CacheName = true
}
func (entity *StoredWarmupRun) GetCacheProvider() *string {
	return entity.CacheProvider
}
func (entity *StoredWarmupRun) SetCacheProvider(value *string) {
	entity.CacheProvider = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.CacheProvider = true
}
func (entity *StoredWarmupRun) GetConnectorName() *string {
	return entity.ConnectorName
}
func (entity *StoredWarmupRun) SetConnectorName(value *string) {
	entity.ConnectorName = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.ConnectorName = true
}
func (entity *StoredWarmupRun) GetIndexColumn() *string {
	return entity.IndexColumn
}
func (entity *StoredWarmupRun) SetIndexColumn(value *string) {
	entity.IndexColumn = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.IndexColumn = true
}
func (entity *StoredWarmupRun) GetPlannedCases() int {
	return entity.PlannedCases
}
func (entity *StoredWarmupRun) SetPlannedCases(value int) {
	entity.PlannedCases = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.PlannedCases = true
}
func (entity *StoredWarmupRun) GetCompletedCases() int {
	return entity.CompletedCases
}
func (entity *StoredWarmupRun) SetCompletedCases(value int) {
	entity.CompletedCases = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.CompletedCases = true
}
func (entity *StoredWarmupRun) GetMaxCases() *int {
	return entity.MaxCases
}
func (entity *StoredWarmupRun) SetMaxCases(value *int) {
	entity.MaxCases = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.MaxCases = true
}
func (entity *StoredWarmupRun) GetRowLimit() *int {
	return entity.RowLimit
}
func (entity *StoredWarmupRun) SetRowLimit(value *int) {
	entity.RowLimit = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.RowLimit = true
}
func (entity *StoredWarmupRun) GetEntries() int {
	return entity.Entries
}
func (entity *StoredWarmupRun) SetEntries(value int) {
	entity.Entries = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.Entries = true
}
func (entity *StoredWarmupRun) GetDurationNs() int64 {
	return entity.DurationNs
}
func (entity *StoredWarmupRun) SetDurationNs(value int64) {
	entity.DurationNs = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.DurationNs = true
}
func (entity *StoredWarmupRun) GetDiagnosticsJson() *json.RawMessage {
	return entity.DiagnosticsJson
}
func (entity *StoredWarmupRun) SetDiagnosticsJson(value *json.RawMessage) {
	entity.DiagnosticsJson = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.DiagnosticsJson = true
}
func (entity *StoredWarmupRun) GetTargetJson() json.RawMessage {
	return entity.TargetJson
}
func (entity *StoredWarmupRun) SetTargetJson(value json.RawMessage) {
	entity.TargetJson = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.TargetJson = true
}
func (entity *StoredWarmupRun) GetRequestedAt() time.Time {
	return entity.RequestedAt
}
func (entity *StoredWarmupRun) SetRequestedAt(value time.Time) {
	entity.RequestedAt = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.RequestedAt = true
}
func (entity *StoredWarmupRun) GetCreatedAt() *time.Time {
	return entity.CreatedAt
}
func (entity *StoredWarmupRun) SetCreatedAt(value *time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *StoredWarmupRun) GetCreatedBy() *string {
	return entity.CreatedBy
}
func (entity *StoredWarmupRun) SetCreatedBy(value *string) {
	entity.CreatedBy = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.CreatedBy = true
}
func (entity *StoredWarmupRun) GetUpdatedAt() *time.Time {
	return entity.UpdatedAt
}
func (entity *StoredWarmupRun) SetUpdatedAt(value *time.Time) {
	entity.UpdatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.UpdatedAt = true
}
func (entity *StoredWarmupRun) GetUpdatedBy() *string {
	return entity.UpdatedBy
}
func (entity *StoredWarmupRun) SetUpdatedBy(value *string) {
	entity.UpdatedBy = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.UpdatedBy = true
}
func (entity *StoredWarmupRun) GetStartedAt() *time.Time {
	return entity.StartedAt
}
func (entity *StoredWarmupRun) SetStartedAt(value *time.Time) {
	entity.StartedAt = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.StartedAt = true
}
func (entity *StoredWarmupRun) GetCompletedAt() *time.Time {
	return entity.CompletedAt
}
func (entity *StoredWarmupRun) SetCompletedAt(value *time.Time) {
	entity.CompletedAt = value
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.CompletedAt = true
}
