package store_write

import (
	time "time"
)

func (entity *StoredSession) GetSessionIdHash() string {
	return entity.SessionIdHash
}
func (entity *StoredSession) SetSessionIdHash(value string) {
	entity.SessionIdHash = value
	if entity.Has == nil {
		entity.Has = &StoredSessionHas{}
	}
	entity.Has.SessionIdHash = true
}
func (entity *StoredSession) GetSubjectId() string {
	return entity.SubjectId
}
func (entity *StoredSession) SetSubjectId(value string) {
	entity.SubjectId = value
	if entity.Has == nil {
		entity.Has = &StoredSessionHas{}
	}
	entity.Has.SubjectId = true
}
func (entity *StoredSession) GetPayloadCiphertext() []byte {
	return entity.PayloadCiphertext
}
func (entity *StoredSession) SetPayloadCiphertext(value []byte) {
	entity.PayloadCiphertext = value
	if entity.Has == nil {
		entity.Has = &StoredSessionHas{}
	}
	entity.Has.PayloadCiphertext = true
}
func (entity *StoredSession) GetExpiresAtUnix() int64 {
	return entity.ExpiresAtUnix
}
func (entity *StoredSession) SetExpiresAtUnix(value int64) {
	entity.ExpiresAtUnix = value
	if entity.Has == nil {
		entity.Has = &StoredSessionHas{}
	}
	entity.Has.ExpiresAtUnix = true
}
func (entity *StoredSession) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *StoredSession) SetShouldDelete(value bool) {
	entity.ShouldDelete = value
	if entity.Has == nil {
		entity.Has = &StoredSessionHas{}
	}
	entity.Has.ShouldDelete = true
}
func (entity *StoredSession) GetCreatedAt() *time.Time {
	return entity.CreatedAt
}
func (entity *StoredSession) SetCreatedAt(value *time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredSessionHas{}
	}
	entity.Has.CreatedAt = true
}
