package writer

import (
	time "time"
)

func (entity *NamespaceMutationRecord) GetOwnerId() string {
	return entity.OwnerId
}
func (entity *NamespaceMutationRecord) SetOwnerId(value string) {
	entity.OwnerId = value
	if entity.Has == nil {
		entity.Has = &NamespaceMutationRecordHas{}
	}
	entity.Has.OwnerId = true
}
func (entity *NamespaceMutationRecord) GetName() string {
	return entity.Name
}
func (entity *NamespaceMutationRecord) SetName(value string) {
	entity.Name = value
	if entity.Has == nil {
		entity.Has = &NamespaceMutationRecordHas{}
	}
	entity.Has.Name = true
}
func (entity *NamespaceMutationRecord) GetTitle() string {
	return entity.Title
}
func (entity *NamespaceMutationRecord) SetTitle(value string) {
	entity.Title = value
	if entity.Has == nil {
		entity.Has = &NamespaceMutationRecordHas{}
	}
	entity.Has.Title = true
}
func (entity *NamespaceMutationRecord) GetStatus() string {
	return entity.Status
}
func (entity *NamespaceMutationRecord) SetStatus(value string) {
	entity.Status = value
	if entity.Has == nil {
		entity.Has = &NamespaceMutationRecordHas{}
	}
	entity.Has.Status = true
}
func (entity *NamespaceMutationRecord) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *NamespaceMutationRecord) GetEtag() *int {
	return entity.Etag
}
func (entity *NamespaceMutationRecord) SetEtag(value *int) {
	entity.Etag = value
	if entity.Has == nil {
		entity.Has = &NamespaceMutationRecordHas{}
	}
	entity.Has.Etag = true
}
func (entity *NamespaceMutationRecord) GetDescription() *string {
	return entity.Description
}
func (entity *NamespaceMutationRecord) SetDescription(value *string) {
	entity.Description = value
	if entity.Has == nil {
		entity.Has = &NamespaceMutationRecordHas{}
	}
	entity.Has.Description = true
}
func (entity *NamespaceMutationRecord) GetCreatedAt() *time.Time {
	return entity.CreatedAt
}
func (entity *NamespaceMutationRecord) SetCreatedAt(value *time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &NamespaceMutationRecordHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *NamespaceMutationRecord) GetUpdatedAt() *time.Time {
	return entity.UpdatedAt
}
func (entity *NamespaceMutationRecord) SetUpdatedAt(value *time.Time) {
	entity.UpdatedAt = value
	if entity.Has == nil {
		entity.Has = &NamespaceMutationRecordHas{}
	}
	entity.Has.UpdatedAt = true
}
func (entity *NamespaceMutationRecord) GetDeletedAt() *time.Time {
	return entity.DeletedAt
}
func (entity *NamespaceMutationRecord) SetDeletedAt(value *time.Time) {
	entity.DeletedAt = value
	if entity.Has == nil {
		entity.Has = &NamespaceMutationRecordHas{}
	}
	entity.Has.DeletedAt = true
}
