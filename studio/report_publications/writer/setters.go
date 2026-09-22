package writer

import (
	json "encoding/json"
	time "time"
)

func (entity *ReportPublication) GetFailureJson() json.RawMessage {
	return entity.FailureJson
}
func (entity *ReportPublication) SetFailureJson(value json.RawMessage) {
	entity.FailureJson = value
	if entity.Has == nil {
		entity.Has = &ReportPublicationHas{}
	}
	entity.Has.FailureJson = true
}
func (entity *ReportPublication) GetReportId() *string {
	return entity.ReportId
}
func (entity *ReportPublication) SetReportId(value *string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &ReportPublicationHas{}
	}
	entity.Has.ReportId = true
}
func (entity *ReportPublication) GetActiveVersionNo() *int {
	return entity.ActiveVersionNo
}
func (entity *ReportPublication) SetActiveVersionNo(value *int) {
	entity.ActiveVersionNo = value
	if entity.Has == nil {
		entity.Has = &ReportPublicationHas{}
	}
	entity.Has.ActiveVersionNo = true
}
func (entity *ReportPublication) GetDesiredGeneration() *int {
	return entity.DesiredGeneration
}
func (entity *ReportPublication) SetDesiredGeneration(value *int) {
	entity.DesiredGeneration = value
	if entity.Has == nil {
		entity.Has = &ReportPublicationHas{}
	}
	entity.Has.DesiredGeneration = true
}
func (entity *ReportPublication) GetPublicationStatus() *string {
	return entity.PublicationStatus
}
func (entity *ReportPublication) SetPublicationStatus(value *string) {
	entity.PublicationStatus = value
	if entity.Has == nil {
		entity.Has = &ReportPublicationHas{}
	}
	entity.Has.PublicationStatus = true
}
func (entity *ReportPublication) GetSpecHash() *string {
	return entity.SpecHash
}
func (entity *ReportPublication) SetSpecHash(value *string) {
	entity.SpecHash = value
	if entity.Has == nil {
		entity.Has = &ReportPublicationHas{}
	}
	entity.Has.SpecHash = true
}
func (entity *ReportPublication) GetPublishedBy() *string {
	return entity.PublishedBy
}
func (entity *ReportPublication) SetPublishedBy(value *string) {
	entity.PublishedBy = value
	if entity.Has == nil {
		entity.Has = &ReportPublicationHas{}
	}
	entity.Has.PublishedBy = true
}
func (entity *ReportPublication) GetPublishedAt() *time.Time {
	return entity.PublishedAt
}
func (entity *ReportPublication) SetPublishedAt(value *time.Time) {
	entity.PublishedAt = value
	if entity.Has == nil {
		entity.Has = &ReportPublicationHas{}
	}
	entity.Has.PublishedAt = true
}
func (entity *ReportPublication) GetActiveGeneration() *int {
	return entity.ActiveGeneration
}
func (entity *ReportPublication) SetActiveGeneration(value *int) {
	entity.ActiveGeneration = value
	if entity.Has == nil {
		entity.Has = &ReportPublicationHas{}
	}
	entity.Has.ActiveGeneration = true
}
func (entity *ReportPublication) GetRuntimeRevision() *string {
	return entity.RuntimeRevision
}
func (entity *ReportPublication) SetRuntimeRevision(value *string) {
	entity.RuntimeRevision = value
	if entity.Has == nil {
		entity.Has = &ReportPublicationHas{}
	}
	entity.Has.RuntimeRevision = true
}
func (entity *ReportPublication) GetActivatedAt() *time.Time {
	return entity.ActivatedAt
}
func (entity *ReportPublication) SetActivatedAt(value *time.Time) {
	entity.ActivatedAt = value
	if entity.Has == nil {
		entity.Has = &ReportPublicationHas{}
	}
	entity.Has.ActivatedAt = true
}
func (entity *ReportPublication) GetDesiredVersionNo() *int {
	return entity.DesiredVersionNo
}
func (entity *ReportPublication) SetDesiredVersionNo(value *int) {
	entity.DesiredVersionNo = value
	if entity.Has == nil {
		entity.Has = &ReportPublicationHas{}
	}
	entity.Has.DesiredVersionNo = true
}
