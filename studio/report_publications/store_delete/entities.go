package store_delete

func (entity *StoredPublication) GetReportId() string {
	return entity.ReportId
}
func (entity *StoredPublication) SetReportId(value string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &StoredPublicationHas{}
	}
	entity.Has.ReportId = true
}
func (entity *StoredPublication) GetDesiredGeneration() *int64 {
	return entity.DesiredGeneration
}
func (entity *StoredPublication) SetDesiredGeneration(value *int64) {
	entity.DesiredGeneration = value
	if entity.Has == nil {
		entity.Has = &StoredPublicationHas{}
	}
	entity.Has.DesiredGeneration = true
}
func (entity *StoredPublication) GetPublicationStatus() string {
	return entity.PublicationStatus
}
func (entity *StoredPublication) SetPublicationStatus(value string) {
	entity.PublicationStatus = value
	if entity.Has == nil {
		entity.Has = &StoredPublicationHas{}
	}
	entity.Has.PublicationStatus = true
}
func (entity *StoredPublication) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *StoredPublication) SetShouldDelete(value bool) {
	entity.ShouldDelete = value
	if entity.Has == nil {
		entity.Has = &StoredPublicationHas{}
	}
	entity.Has.ShouldDelete = true
}
