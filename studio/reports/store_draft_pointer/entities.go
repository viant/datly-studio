package store_draft_pointer

import (
	time "time"
)

func (entity *DraftPointer) GetId() string {
	return entity.Id
}
func (entity *DraftPointer) SetId(value string) {
	entity.Id = value
	if entity.Has == nil {
		entity.Has = &DraftPointerHas{}
	}
	entity.Has.Id = true
}
func (entity *DraftPointer) GetCurrentDraftVersion() *int {
	return entity.CurrentDraftVersion
}
func (entity *DraftPointer) SetCurrentDraftVersion(value *int) {
	entity.CurrentDraftVersion = value
	if entity.Has == nil {
		entity.Has = &DraftPointerHas{}
	}
	entity.Has.CurrentDraftVersion = true
}
func (entity *DraftPointer) GetEtag() *int64 {
	return entity.Etag
}
func (entity *DraftPointer) SetEtag(value *int64) {
	entity.Etag = value
	if entity.Has == nil {
		entity.Has = &DraftPointerHas{}
	}
	entity.Has.Etag = true
}
func (entity *DraftPointer) GetUpdatedAt() *time.Time {
	return entity.UpdatedAt
}
func (entity *DraftPointer) SetUpdatedAt(value *time.Time) {
	entity.UpdatedAt = value
	if entity.Has == nil {
		entity.Has = &DraftPointerHas{}
	}
	entity.Has.UpdatedAt = true
}
