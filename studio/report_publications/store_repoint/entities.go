package store_repoint

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
func (entity *StoredPublication) GetActiveGeneration() *int64 {
	return entity.ActiveGeneration
}
func (entity *StoredPublication) SetActiveGeneration(value *int64) {
	entity.ActiveGeneration = value
	if entity.Has == nil {
		entity.Has = &StoredPublicationHas{}
	}
	entity.Has.ActiveGeneration = true
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
