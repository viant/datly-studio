package store_write

func (entity *StoredACL) GetReportId() *string {
	return entity.ReportId
}
func (entity *StoredACL) SetReportId(value *string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &StoredACLHas{}
	}
	entity.Has.ReportId = true
}
func (entity *StoredACL) GetSubjectType() *string {
	return entity.SubjectType
}
func (entity *StoredACL) SetSubjectType(value *string) {
	entity.SubjectType = value
	if entity.Has == nil {
		entity.Has = &StoredACLHas{}
	}
	entity.Has.SubjectType = true
}
func (entity *StoredACL) GetSubjectId() *string {
	return entity.SubjectId
}
func (entity *StoredACL) SetSubjectId(value *string) {
	entity.SubjectId = value
	if entity.Has == nil {
		entity.Has = &StoredACLHas{}
	}
	entity.Has.SubjectId = true
}
func (entity *StoredACL) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *StoredACL) SetShouldDelete(value bool) {
	entity.ShouldDelete = value
	if entity.Has == nil {
		entity.Has = &StoredACLHas{}
	}
	entity.Has.ShouldDelete = true
}
func (entity *StoredACL) GetEtag() *int {
	return entity.Etag
}
func (entity *StoredACL) SetEtag(value *int) {
	entity.Etag = value
	if entity.Has == nil {
		entity.Has = &StoredACLHas{}
	}
	entity.Has.Etag = true
}
func (entity *StoredACL) GetCanView() *int {
	return entity.CanView
}
func (entity *StoredACL) SetCanView(value *int) {
	entity.CanView = value
	if entity.Has == nil {
		entity.Has = &StoredACLHas{}
	}
	entity.Has.CanView = true
}
func (entity *StoredACL) GetCanRun() *int {
	return entity.CanRun
}
func (entity *StoredACL) SetCanRun(value *int) {
	entity.CanRun = value
	if entity.Has == nil {
		entity.Has = &StoredACLHas{}
	}
	entity.Has.CanRun = true
}
func (entity *StoredACL) GetCanEdit() *int {
	return entity.CanEdit
}
func (entity *StoredACL) SetCanEdit(value *int) {
	entity.CanEdit = value
	if entity.Has == nil {
		entity.Has = &StoredACLHas{}
	}
	entity.Has.CanEdit = true
}
func (entity *StoredACL) GetCanPublish() *int {
	return entity.CanPublish
}
func (entity *StoredACL) SetCanPublish(value *int) {
	entity.CanPublish = value
	if entity.Has == nil {
		entity.Has = &StoredACLHas{}
	}
	entity.Has.CanPublish = true
}
func (entity *StoredACL) GetCanUseDql() *int {
	return entity.CanUseDql
}
func (entity *StoredACL) SetCanUseDql(value *int) {
	entity.CanUseDql = value
	if entity.Has == nil {
		entity.Has = &StoredACLHas{}
	}
	entity.Has.CanUseDql = true
}
