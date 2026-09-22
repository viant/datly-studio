package writer

import (
	time "time"
)

func (entity *ReportResourceFile) GetReportId() *string {
	return entity.ReportId
}
func (entity *ReportResourceFile) SetReportId(value *string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFileHas{}
	}
	entity.Has.ReportId = true
}
func (entity *ReportResourceFile) GetVersionNo() *int {
	return entity.VersionNo
}
func (entity *ReportResourceFile) SetVersionNo(value *int) {
	entity.VersionNo = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFileHas{}
	}
	entity.Has.VersionNo = true
}
func (entity *ReportResourceFile) GetResourceId() *string {
	return entity.ResourceId
}
func (entity *ReportResourceFile) SetResourceId(value *string) {
	entity.ResourceId = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFileHas{}
	}
	entity.Has.ResourceId = true
}
func (entity *ReportResourceFile) GetNamespace() *string {
	return entity.Namespace
}
func (entity *ReportResourceFile) SetNamespace(value *string) {
	entity.Namespace = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFileHas{}
	}
	entity.Has.Namespace = true
}
func (entity *ReportResourceFile) GetResourcePath() *string {
	return entity.ResourcePath
}
func (entity *ReportResourceFile) SetResourcePath(value *string) {
	entity.ResourcePath = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFileHas{}
	}
	entity.Has.ResourcePath = true
}
func (entity *ReportResourceFile) GetContent() *string {
	return entity.Content
}
func (entity *ReportResourceFile) SetContent(value *string) {
	entity.Content = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFileHas{}
	}
	entity.Has.Content = true
}
func (entity *ReportResourceFile) GetContentSize() *int {
	return entity.ContentSize
}
func (entity *ReportResourceFile) SetContentSize(value *int) {
	entity.ContentSize = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFileHas{}
	}
	entity.Has.ContentSize = true
}
func (entity *ReportResourceFile) GetContentSha256() *string {
	return entity.ContentSha256
}
func (entity *ReportResourceFile) SetContentSha256(value *string) {
	entity.ContentSha256 = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFileHas{}
	}
	entity.Has.ContentSha256 = true
}
func (entity *ReportResourceFile) GetCreatedAt() *time.Time {
	return entity.CreatedAt
}
func (entity *ReportResourceFile) SetCreatedAt(value *time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFileHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *ReportResourceFile) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *ReportResourceFile) GetMediaType() *string {
	return entity.MediaType
}
func (entity *ReportResourceFile) SetMediaType(value *string) {
	entity.MediaType = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFileHas{}
	}
	entity.Has.MediaType = true
}
func (entity *ReportResourceFile) GetIsBinary() *int {
	return entity.IsBinary
}
func (entity *ReportResourceFile) SetIsBinary(value *int) {
	entity.IsBinary = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFileHas{}
	}
	entity.Has.IsBinary = true
}
