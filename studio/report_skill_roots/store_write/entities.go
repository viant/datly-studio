package store_write

func (entity *StoredSkill) GetReportId() string {
	return entity.ReportId
}
func (entity *StoredSkill) SetReportId(value string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &StoredSkillHas{}
	}
	entity.Has.ReportId = true
}
func (entity *StoredSkill) GetVersionNo() int {
	return entity.VersionNo
}
func (entity *StoredSkill) SetVersionNo(value int) {
	entity.VersionNo = value
	if entity.Has == nil {
		entity.Has = &StoredSkillHas{}
	}
	entity.Has.VersionNo = true
}
func (entity *StoredSkill) GetSkillId() string {
	return entity.SkillId
}
func (entity *StoredSkill) SetSkillId(value string) {
	entity.SkillId = value
	if entity.Has == nil {
		entity.Has = &StoredSkillHas{}
	}
	entity.Has.SkillId = true
}
func (entity *StoredSkill) GetFolderId() string {
	return entity.FolderId
}
func (entity *StoredSkill) SetFolderId(value string) {
	entity.FolderId = value
	if entity.Has == nil {
		entity.Has = &StoredSkillHas{}
	}
	entity.Has.FolderId = true
}
func (entity *StoredSkill) GetSkillRoot() string {
	return entity.SkillRoot
}
func (entity *StoredSkill) SetSkillRoot(value string) {
	entity.SkillRoot = value
	if entity.Has == nil {
		entity.Has = &StoredSkillHas{}
	}
	entity.Has.SkillRoot = true
}
func (entity *StoredSkill) GetOrdinal() int {
	return entity.Ordinal
}
func (entity *StoredSkill) SetOrdinal(value int) {
	entity.Ordinal = value
	if entity.Has == nil {
		entity.Has = &StoredSkillHas{}
	}
	entity.Has.Ordinal = true
}
func (entity *StoredSkill) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *StoredSkill) SetShouldDelete(value bool) {
	entity.ShouldDelete = value
	if entity.Has == nil {
		entity.Has = &StoredSkillHas{}
	}
	entity.Has.ShouldDelete = true
}
