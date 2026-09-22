package writer

func (entity *ReportACL) GetReportId() *string {
	return entity.ReportId
}
func (entity *ReportACL) SetReportId(value *string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &ReportACLHas{}
	}
	entity.Has.ReportId = true
}
func (entity *ReportACL) GetSubjectType() *string {
	return entity.SubjectType
}
func (entity *ReportACL) SetSubjectType(value *string) {
	entity.SubjectType = value
	if entity.Has == nil {
		entity.Has = &ReportACLHas{}
	}
	entity.Has.SubjectType = true
}
func (entity *ReportACL) GetSubjectId() *string {
	return entity.SubjectId
}
func (entity *ReportACL) SetSubjectId(value *string) {
	entity.SubjectId = value
	if entity.Has == nil {
		entity.Has = &ReportACLHas{}
	}
	entity.Has.SubjectId = true
}
func (entity *ReportACL) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *ReportACL) GetEtag() *int {
	return entity.Etag
}
func (entity *ReportACL) SetEtag(value *int) {
	entity.Etag = value
	if entity.Has == nil {
		entity.Has = &ReportACLHas{}
	}
	entity.Has.Etag = true
}
func (entity *ReportACL) GetCanView() *int {
	return entity.CanView
}
func (entity *ReportACL) SetCanView(value *int) {
	entity.CanView = value
	if entity.Has == nil {
		entity.Has = &ReportACLHas{}
	}
	entity.Has.CanView = true
}
func (entity *ReportACL) GetCanRun() *int {
	return entity.CanRun
}
func (entity *ReportACL) SetCanRun(value *int) {
	entity.CanRun = value
	if entity.Has == nil {
		entity.Has = &ReportACLHas{}
	}
	entity.Has.CanRun = true
}
func (entity *ReportACL) GetCanEdit() *int {
	return entity.CanEdit
}
func (entity *ReportACL) SetCanEdit(value *int) {
	entity.CanEdit = value
	if entity.Has == nil {
		entity.Has = &ReportACLHas{}
	}
	entity.Has.CanEdit = true
}
func (entity *ReportACL) GetCanPublish() *int {
	return entity.CanPublish
}
func (entity *ReportACL) SetCanPublish(value *int) {
	entity.CanPublish = value
	if entity.Has == nil {
		entity.Has = &ReportACLHas{}
	}
	entity.Has.CanPublish = true
}
func (entity *ReportACL) GetCanUseDql() *int {
	return entity.CanUseDql
}
func (entity *ReportACL) SetCanUseDql(value *int) {
	entity.CanUseDql = value
	if entity.Has == nil {
		entity.Has = &ReportACLHas{}
	}
	entity.Has.CanUseDql = true
}
