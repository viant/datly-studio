package store_import

import (
	json "encoding/json"
	time "time"
)

func (entity *ImportedVersion) GetReportId() string {
	return entity.ReportId
}
func (entity *ImportedVersion) SetReportId(value string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.ReportId = true
}
func (entity *ImportedVersion) GetVersionNo() int {
	return entity.VersionNo
}
func (entity *ImportedVersion) SetVersionNo(value int) {
	entity.VersionNo = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.VersionNo = true
}
func (entity *ImportedVersion) GetState() string {
	return entity.State
}
func (entity *ImportedVersion) SetState(value string) {
	entity.State = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.State = true
}
func (entity *ImportedVersion) GetAuthoringMode() string {
	return entity.AuthoringMode
}
func (entity *ImportedVersion) SetAuthoringMode(value string) {
	entity.AuthoringMode = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.AuthoringMode = true
}
func (entity *ImportedVersion) GetAuthoredDql() *string {
	return entity.AuthoredDql
}
func (entity *ImportedVersion) SetAuthoredDql(value *string) {
	entity.AuthoredDql = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.AuthoredDql = true
}
func (entity *ImportedVersion) GetGeneratedDql() *string {
	return entity.GeneratedDql
}
func (entity *ImportedVersion) SetGeneratedDql(value *string) {
	entity.GeneratedDql = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.GeneratedDql = true
}
func (entity *ImportedVersion) GetComponentSpecJson() json.RawMessage {
	return entity.ComponentSpecJson
}
func (entity *ImportedVersion) SetComponentSpecJson(value json.RawMessage) {
	entity.ComponentSpecJson = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.ComponentSpecJson = true
}
func (entity *ImportedVersion) GetSpecFormatVersion() string {
	return entity.SpecFormatVersion
}
func (entity *ImportedVersion) SetSpecFormatVersion(value string) {
	entity.SpecFormatVersion = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.SpecFormatVersion = true
}
func (entity *ImportedVersion) GetSpecHash() string {
	return entity.SpecHash
}
func (entity *ImportedVersion) SetSpecHash(value string) {
	entity.SpecHash = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.SpecHash = true
}
func (entity *ImportedVersion) GetTypeManifestJson() json.RawMessage {
	return entity.TypeManifestJson
}
func (entity *ImportedVersion) SetTypeManifestJson(value json.RawMessage) {
	entity.TypeManifestJson = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.TypeManifestJson = true
}
func (entity *ImportedVersion) GetCompileStatus() string {
	return entity.CompileStatus
}
func (entity *ImportedVersion) SetCompileStatus(value string) {
	entity.CompileStatus = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.CompileStatus = true
}
func (entity *ImportedVersion) GetDatlyVersion() string {
	return entity.DatlyVersion
}
func (entity *ImportedVersion) SetDatlyVersion(value string) {
	entity.DatlyVersion = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.DatlyVersion = true
}
func (entity *ImportedVersion) GetCompilerVersion() string {
	return entity.CompilerVersion
}
func (entity *ImportedVersion) SetCompilerVersion(value string) {
	entity.CompilerVersion = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.CompilerVersion = true
}
func (entity *ImportedVersion) GetSourceRevision() int64 {
	return entity.SourceRevision
}
func (entity *ImportedVersion) SetSourceRevision(value int64) {
	entity.SourceRevision = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.SourceRevision = true
}
func (entity *ImportedVersion) GetNotes() *string {
	return entity.Notes
}
func (entity *ImportedVersion) SetNotes(value *string) {
	entity.Notes = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.Notes = true
}
func (entity *ImportedVersion) GetCreatedBy() string {
	return entity.CreatedBy
}
func (entity *ImportedVersion) SetCreatedBy(value string) {
	entity.CreatedBy = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.CreatedBy = true
}
func (entity *ImportedVersion) GetCreatedAt() time.Time {
	return entity.CreatedAt
}
func (entity *ImportedVersion) SetCreatedAt(value time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *ImportedVersion) GetFile() []*ImportedResourceFile {
	return entity.File
}
func (entity *ImportedVersion) SetFile(value []*ImportedResourceFile) {
	entity.File = value
	if entity.Has == nil {
		entity.Has = &ImportedVersionHas{}
	}
	entity.Has.File = true
}
func (entity *ImportedResourceFile) GetReportId() string {
	return entity.ReportId
}
func (entity *ImportedResourceFile) SetReportId(value string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &ImportedResourceFileHas{}
	}
	entity.Has.ReportId = true
}
func (entity *ImportedResourceFile) GetVersionNo() int {
	return entity.VersionNo
}
func (entity *ImportedResourceFile) SetVersionNo(value int) {
	entity.VersionNo = value
	if entity.Has == nil {
		entity.Has = &ImportedResourceFileHas{}
	}
	entity.Has.VersionNo = true
}
func (entity *ImportedResourceFile) GetResourceId() string {
	return entity.ResourceId
}
func (entity *ImportedResourceFile) SetResourceId(value string) {
	entity.ResourceId = value
	if entity.Has == nil {
		entity.Has = &ImportedResourceFileHas{}
	}
	entity.Has.ResourceId = true
}
func (entity *ImportedResourceFile) GetNamespace() string {
	return entity.Namespace
}
func (entity *ImportedResourceFile) SetNamespace(value string) {
	entity.Namespace = value
	if entity.Has == nil {
		entity.Has = &ImportedResourceFileHas{}
	}
	entity.Has.Namespace = true
}
func (entity *ImportedResourceFile) GetResourcePath() string {
	return entity.ResourcePath
}
func (entity *ImportedResourceFile) SetResourcePath(value string) {
	entity.ResourcePath = value
	if entity.Has == nil {
		entity.Has = &ImportedResourceFileHas{}
	}
	entity.Has.ResourcePath = true
}
func (entity *ImportedResourceFile) GetContent() []byte {
	return entity.Content
}
func (entity *ImportedResourceFile) SetContent(value []byte) {
	entity.Content = value
	if entity.Has == nil {
		entity.Has = &ImportedResourceFileHas{}
	}
	entity.Has.Content = true
}
func (entity *ImportedResourceFile) GetContentSize() int64 {
	return entity.ContentSize
}
func (entity *ImportedResourceFile) SetContentSize(value int64) {
	entity.ContentSize = value
	if entity.Has == nil {
		entity.Has = &ImportedResourceFileHas{}
	}
	entity.Has.ContentSize = true
}
func (entity *ImportedResourceFile) GetContentSha256() string {
	return entity.ContentSha256
}
func (entity *ImportedResourceFile) SetContentSha256(value string) {
	entity.ContentSha256 = value
	if entity.Has == nil {
		entity.Has = &ImportedResourceFileHas{}
	}
	entity.Has.ContentSha256 = true
}
func (entity *ImportedResourceFile) GetIsBinary() bool {
	return entity.IsBinary
}
func (entity *ImportedResourceFile) SetIsBinary(value bool) {
	entity.IsBinary = value
	if entity.Has == nil {
		entity.Has = &ImportedResourceFileHas{}
	}
	entity.Has.IsBinary = true
}
func (entity *ImportedResourceFile) GetCreatedAt() time.Time {
	return entity.CreatedAt
}
func (entity *ImportedResourceFile) SetCreatedAt(value time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &ImportedResourceFileHas{}
	}
	entity.Has.CreatedAt = true
}
