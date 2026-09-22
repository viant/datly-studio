package writer

func (entity *ReportSkillRoot) GetReportId() *string {
	return entity.ReportId
}
func (entity *ReportSkillRoot) SetReportId(value *string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &ReportSkillRootHas{}
	}
	entity.Has.ReportId = true
}
func (entity *ReportSkillRoot) GetVersionNo() *int {
	return entity.VersionNo
}
func (entity *ReportSkillRoot) SetVersionNo(value *int) {
	entity.VersionNo = value
	if entity.Has == nil {
		entity.Has = &ReportSkillRootHas{}
	}
	entity.Has.VersionNo = true
}
func (entity *ReportSkillRoot) GetSkillId() *string {
	return entity.SkillId
}
func (entity *ReportSkillRoot) SetSkillId(value *string) {
	entity.SkillId = value
	if entity.Has == nil {
		entity.Has = &ReportSkillRootHas{}
	}
	entity.Has.SkillId = true
}
func (entity *ReportSkillRoot) GetFolderId() *string {
	return entity.FolderId
}
func (entity *ReportSkillRoot) SetFolderId(value *string) {
	entity.FolderId = value
	if entity.Has == nil {
		entity.Has = &ReportSkillRootHas{}
	}
	entity.Has.FolderId = true
}
func (entity *ReportSkillRoot) GetSkillRoot() *string {
	return entity.SkillRoot
}
func (entity *ReportSkillRoot) SetSkillRoot(value *string) {
	entity.SkillRoot = value
	if entity.Has == nil {
		entity.Has = &ReportSkillRootHas{}
	}
	entity.Has.SkillRoot = true
}
func (entity *ReportSkillRoot) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *ReportSkillRoot) GetOrdinal() *int {
	return entity.Ordinal
}
func (entity *ReportSkillRoot) SetOrdinal(value *int) {
	entity.Ordinal = value
	if entity.Has == nil {
		entity.Has = &ReportSkillRootHas{}
	}
	entity.Has.Ordinal = true
}
