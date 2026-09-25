package store_insert

import (
	time "time"
)

func (entity *StoredEvent) GetEventId() string {
	return entity.EventId
}
func (entity *StoredEvent) SetEventId(value string) {
	entity.EventId = value
	if entity.Has == nil {
		entity.Has = &StoredEventHas{}
	}
	entity.Has.EventId = true
}
func (entity *StoredEvent) GetReportId() string {
	return entity.ReportId
}
func (entity *StoredEvent) SetReportId(value string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &StoredEventHas{}
	}
	entity.Has.ReportId = true
}
func (entity *StoredEvent) GetOwnerId() string {
	return entity.OwnerId
}
func (entity *StoredEvent) SetOwnerId(value string) {
	entity.OwnerId = value
	if entity.Has == nil {
		entity.Has = &StoredEventHas{}
	}
	entity.Has.OwnerId = true
}
func (entity *StoredEvent) GetOperation() string {
	return entity.Operation
}
func (entity *StoredEvent) SetOperation(value string) {
	entity.Operation = value
	if entity.Has == nil {
		entity.Has = &StoredEventHas{}
	}
	entity.Has.Operation = true
}
func (entity *StoredEvent) GetVersionNo() *int {
	return entity.VersionNo
}
func (entity *StoredEvent) SetVersionNo(value *int) {
	entity.VersionNo = value
	if entity.Has == nil {
		entity.Has = &StoredEventHas{}
	}
	entity.Has.VersionNo = true
}
func (entity *StoredEvent) GetGenerationNo() *int64 {
	return entity.GenerationNo
}
func (entity *StoredEvent) SetGenerationNo(value *int64) {
	entity.GenerationNo = value
	if entity.Has == nil {
		entity.Has = &StoredEventHas{}
	}
	entity.Has.GenerationNo = true
}
func (entity *StoredEvent) GetStatus() string {
	return entity.Status
}
func (entity *StoredEvent) SetStatus(value string) {
	entity.Status = value
	if entity.Has == nil {
		entity.Has = &StoredEventHas{}
	}
	entity.Has.Status = true
}
func (entity *StoredEvent) GetRequestedBy() string {
	return entity.RequestedBy
}
func (entity *StoredEvent) SetRequestedBy(value string) {
	entity.RequestedBy = value
	if entity.Has == nil {
		entity.Has = &StoredEventHas{}
	}
	entity.Has.RequestedBy = true
}
func (entity *StoredEvent) GetReason() *string {
	return entity.Reason
}
func (entity *StoredEvent) SetReason(value *string) {
	entity.Reason = value
	if entity.Has == nil {
		entity.Has = &StoredEventHas{}
	}
	entity.Has.Reason = true
}
func (entity *StoredEvent) GetFailureCode() *string {
	return entity.FailureCode
}
func (entity *StoredEvent) SetFailureCode(value *string) {
	entity.FailureCode = value
	if entity.Has == nil {
		entity.Has = &StoredEventHas{}
	}
	entity.Has.FailureCode = true
}
func (entity *StoredEvent) GetFailureMessage() *string {
	return entity.FailureMessage
}
func (entity *StoredEvent) SetFailureMessage(value *string) {
	entity.FailureMessage = value
	if entity.Has == nil {
		entity.Has = &StoredEventHas{}
	}
	entity.Has.FailureMessage = true
}
func (entity *StoredEvent) GetOccurredAt() time.Time {
	return entity.OccurredAt
}
func (entity *StoredEvent) SetOccurredAt(value time.Time) {
	entity.OccurredAt = value
	if entity.Has == nil {
		entity.Has = &StoredEventHas{}
	}
	entity.Has.OccurredAt = true
}
