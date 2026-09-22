package writer

func (entity *ReportResourceFolder) GetReportId() *string {
	return entity.ReportId
}
func (entity *ReportResourceFolder) SetReportId(value *string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFolderHas{}
	}
	entity.Has.ReportId = true
}
func (entity *ReportResourceFolder) GetVersionNo() *int {
	return entity.VersionNo
}
func (entity *ReportResourceFolder) SetVersionNo(value *int) {
	entity.VersionNo = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFolderHas{}
	}
	entity.Has.VersionNo = true
}
func (entity *ReportResourceFolder) GetFolderId() *string {
	return entity.FolderId
}
func (entity *ReportResourceFolder) SetFolderId(value *string) {
	entity.FolderId = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFolderHas{}
	}
	entity.Has.FolderId = true
}
func (entity *ReportResourceFolder) GetNamespace() *string {
	return entity.Namespace
}
func (entity *ReportResourceFolder) SetNamespace(value *string) {
	entity.Namespace = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFolderHas{}
	}
	entity.Has.Namespace = true
}
func (entity *ReportResourceFolder) GetRootPath() *string {
	return entity.RootPath
}
func (entity *ReportResourceFolder) SetRootPath(value *string) {
	entity.RootPath = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFolderHas{}
	}
	entity.Has.RootPath = true
}
func (entity *ReportResourceFolder) GetUriPrefix() *string {
	return entity.UriPrefix
}
func (entity *ReportResourceFolder) SetUriPrefix(value *string) {
	entity.UriPrefix = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFolderHas{}
	}
	entity.Has.UriPrefix = true
}
func (entity *ReportResourceFolder) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *ReportResourceFolder) GetOrdinal() *int {
	return entity.Ordinal
}
func (entity *ReportResourceFolder) SetOrdinal(value *int) {
	entity.Ordinal = value
	if entity.Has == nil {
		entity.Has = &ReportResourceFolderHas{}
	}
	entity.Has.Ordinal = true
}
