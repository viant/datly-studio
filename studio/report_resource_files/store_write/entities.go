package store_write

import (
	time "time"
)

func (entity *StoredFile) GetReportId() string {
	return entity.ReportId
}
func (entity *StoredFile) SetReportId(value string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &StoredFileHas{}
	}
	entity.Has.ReportId = true
}
func (entity *StoredFile) GetVersionNo() int {
	return entity.VersionNo
}
func (entity *StoredFile) SetVersionNo(value int) {
	entity.VersionNo = value
	if entity.Has == nil {
		entity.Has = &StoredFileHas{}
	}
	entity.Has.VersionNo = true
}
func (entity *StoredFile) GetResourceId() string {
	return entity.ResourceId
}
func (entity *StoredFile) SetResourceId(value string) {
	entity.ResourceId = value
	if entity.Has == nil {
		entity.Has = &StoredFileHas{}
	}
	entity.Has.ResourceId = true
}
func (entity *StoredFile) GetNamespace() string {
	return entity.Namespace
}
func (entity *StoredFile) SetNamespace(value string) {
	entity.Namespace = value
	if entity.Has == nil {
		entity.Has = &StoredFileHas{}
	}
	entity.Has.Namespace = true
}
func (entity *StoredFile) GetResourcePath() string {
	return entity.ResourcePath
}
func (entity *StoredFile) SetResourcePath(value string) {
	entity.ResourcePath = value
	if entity.Has == nil {
		entity.Has = &StoredFileHas{}
	}
	entity.Has.ResourcePath = true
}
func (entity *StoredFile) GetMediaType() *string {
	return entity.MediaType
}
func (entity *StoredFile) SetMediaType(value *string) {
	entity.MediaType = value
	if entity.Has == nil {
		entity.Has = &StoredFileHas{}
	}
	entity.Has.MediaType = true
}
func (entity *StoredFile) GetContent() []byte {
	return entity.Content
}
func (entity *StoredFile) SetContent(value []byte) {
	entity.Content = value
	if entity.Has == nil {
		entity.Has = &StoredFileHas{}
	}
	entity.Has.Content = true
}
func (entity *StoredFile) GetContentSize() int64 {
	return entity.ContentSize
}
func (entity *StoredFile) SetContentSize(value int64) {
	entity.ContentSize = value
	if entity.Has == nil {
		entity.Has = &StoredFileHas{}
	}
	entity.Has.ContentSize = true
}
func (entity *StoredFile) GetContentSha256() string {
	return entity.ContentSha256
}
func (entity *StoredFile) SetContentSha256(value string) {
	entity.ContentSha256 = value
	if entity.Has == nil {
		entity.Has = &StoredFileHas{}
	}
	entity.Has.ContentSha256 = true
}
func (entity *StoredFile) GetIsBinary() bool {
	return entity.IsBinary
}
func (entity *StoredFile) SetIsBinary(value bool) {
	entity.IsBinary = value
	if entity.Has == nil {
		entity.Has = &StoredFileHas{}
	}
	entity.Has.IsBinary = true
}
func (entity *StoredFile) GetCreatedAt() time.Time {
	return entity.CreatedAt
}
func (entity *StoredFile) SetCreatedAt(value time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredFileHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *StoredFile) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *StoredFile) SetShouldDelete(value bool) {
	entity.ShouldDelete = value
	if entity.Has == nil {
		entity.Has = &StoredFileHas{}
	}
	entity.Has.ShouldDelete = true
}
