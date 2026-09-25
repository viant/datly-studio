package store_write

func (entity *StoredFolder) GetReportId() string {
	return entity.ReportId
}
func (entity *StoredFolder) SetReportId(value string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &StoredFolderHas{}
	}
	entity.Has.ReportId = true
}
func (entity *StoredFolder) GetVersionNo() int {
	return entity.VersionNo
}
func (entity *StoredFolder) SetVersionNo(value int) {
	entity.VersionNo = value
	if entity.Has == nil {
		entity.Has = &StoredFolderHas{}
	}
	entity.Has.VersionNo = true
}
func (entity *StoredFolder) GetFolderId() string {
	return entity.FolderId
}
func (entity *StoredFolder) SetFolderId(value string) {
	entity.FolderId = value
	if entity.Has == nil {
		entity.Has = &StoredFolderHas{}
	}
	entity.Has.FolderId = true
}
func (entity *StoredFolder) GetNamespace() string {
	return entity.Namespace
}
func (entity *StoredFolder) SetNamespace(value string) {
	entity.Namespace = value
	if entity.Has == nil {
		entity.Has = &StoredFolderHas{}
	}
	entity.Has.Namespace = true
}
func (entity *StoredFolder) GetRootPath() string {
	return entity.RootPath
}
func (entity *StoredFolder) SetRootPath(value string) {
	entity.RootPath = value
	if entity.Has == nil {
		entity.Has = &StoredFolderHas{}
	}
	entity.Has.RootPath = true
}
func (entity *StoredFolder) GetUriPrefix() string {
	return entity.UriPrefix
}
func (entity *StoredFolder) SetUriPrefix(value string) {
	entity.UriPrefix = value
	if entity.Has == nil {
		entity.Has = &StoredFolderHas{}
	}
	entity.Has.UriPrefix = true
}
func (entity *StoredFolder) GetOrdinal() int {
	return entity.Ordinal
}
func (entity *StoredFolder) SetOrdinal(value int) {
	entity.Ordinal = value
	if entity.Has == nil {
		entity.Has = &StoredFolderHas{}
	}
	entity.Has.Ordinal = true
}
func (entity *StoredFolder) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *StoredFolder) SetShouldDelete(value bool) {
	entity.ShouldDelete = value
	if entity.Has == nil {
		entity.Has = &StoredFolderHas{}
	}
	entity.Has.ShouldDelete = true
}
