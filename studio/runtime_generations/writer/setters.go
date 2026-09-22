package writer

import (
	json "encoding/json"
	time "time"
)

func (entity *RuntimeGeneration) GetBuildManifestJson() json.RawMessage {
	return entity.BuildManifestJson
}
func (entity *RuntimeGeneration) SetBuildManifestJson(value json.RawMessage) {
	entity.BuildManifestJson = value
	if entity.Has == nil {
		entity.Has = &RuntimeGenerationHas{}
	}
	entity.Has.BuildManifestJson = true
}
func (entity *RuntimeGeneration) GetDiagnosticsJson() json.RawMessage {
	return entity.DiagnosticsJson
}
func (entity *RuntimeGeneration) SetDiagnosticsJson(value json.RawMessage) {
	entity.DiagnosticsJson = value
	if entity.Has == nil {
		entity.Has = &RuntimeGenerationHas{}
	}
	entity.Has.DiagnosticsJson = true
}
func (entity *RuntimeGeneration) GetGenerationNo() *int {
	return entity.GenerationNo
}
func (entity *RuntimeGeneration) SetGenerationNo(value *int) {
	entity.GenerationNo = value
	if entity.Has == nil {
		entity.Has = &RuntimeGenerationHas{}
	}
	entity.Has.GenerationNo = true
}
func (entity *RuntimeGeneration) GetSourceRevision() *string {
	return entity.SourceRevision
}
func (entity *RuntimeGeneration) SetSourceRevision(value *string) {
	entity.SourceRevision = value
	if entity.Has == nil {
		entity.Has = &RuntimeGenerationHas{}
	}
	entity.Has.SourceRevision = true
}
func (entity *RuntimeGeneration) GetStatus() *string {
	return entity.Status
}
func (entity *RuntimeGeneration) SetStatus(value *string) {
	entity.Status = value
	if entity.Has == nil {
		entity.Has = &RuntimeGenerationHas{}
	}
	entity.Has.Status = true
}
func (entity *RuntimeGeneration) GetRequestedBy() *string {
	return entity.RequestedBy
}
func (entity *RuntimeGeneration) SetRequestedBy(value *string) {
	entity.RequestedBy = value
	if entity.Has == nil {
		entity.Has = &RuntimeGenerationHas{}
	}
	entity.Has.RequestedBy = true
}
func (entity *RuntimeGeneration) GetRequestedAt() *time.Time {
	return entity.RequestedAt
}
func (entity *RuntimeGeneration) SetRequestedAt(value *time.Time) {
	entity.RequestedAt = value
	if entity.Has == nil {
		entity.Has = &RuntimeGenerationHas{}
	}
	entity.Has.RequestedAt = true
}
func (entity *RuntimeGeneration) GetReportCount() *int {
	return entity.ReportCount
}
func (entity *RuntimeGeneration) SetReportCount(value *int) {
	entity.ReportCount = value
	if entity.Has == nil {
		entity.Has = &RuntimeGenerationHas{}
	}
	entity.Has.ReportCount = true
}
func (entity *RuntimeGeneration) GetActivatedAt() *time.Time {
	return entity.ActivatedAt
}
func (entity *RuntimeGeneration) SetActivatedAt(value *time.Time) {
	entity.ActivatedAt = value
	if entity.Has == nil {
		entity.Has = &RuntimeGenerationHas{}
	}
	entity.Has.ActivatedAt = true
}
func (entity *RuntimeGeneration) GetRetiredAt() *time.Time {
	return entity.RetiredAt
}
func (entity *RuntimeGeneration) SetRetiredAt(value *time.Time) {
	entity.RetiredAt = value
	if entity.Has == nil {
		entity.Has = &RuntimeGenerationHas{}
	}
	entity.Has.RetiredAt = true
}
