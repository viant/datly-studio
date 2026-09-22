package writer

func (entity *ReportMCPExposure) GetReportId() *string {
	return entity.ReportId
}
func (entity *ReportMCPExposure) SetReportId(value *string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &ReportMCPExposureHas{}
	}
	entity.Has.ReportId = true
}
func (entity *ReportMCPExposure) GetVersionNo() *int {
	return entity.VersionNo
}
func (entity *ReportMCPExposure) SetVersionNo(value *int) {
	entity.VersionNo = value
	if entity.Has == nil {
		entity.Has = &ReportMCPExposureHas{}
	}
	entity.Has.VersionNo = true
}
func (entity *ReportMCPExposure) GetExposureId() *string {
	return entity.ExposureId
}
func (entity *ReportMCPExposure) SetExposureId(value *string) {
	entity.ExposureId = value
	if entity.Has == nil {
		entity.Has = &ReportMCPExposureHas{}
	}
	entity.Has.ExposureId = true
}
func (entity *ReportMCPExposure) GetRouteId() *string {
	return entity.RouteId
}
func (entity *ReportMCPExposure) SetRouteId(value *string) {
	entity.RouteId = value
	if entity.Has == nil {
		entity.Has = &ReportMCPExposureHas{}
	}
	entity.Has.RouteId = true
}
func (entity *ReportMCPExposure) GetRouteMethod() *string {
	return entity.RouteMethod
}
func (entity *ReportMCPExposure) SetRouteMethod(value *string) {
	entity.RouteMethod = value
	if entity.Has == nil {
		entity.Has = &ReportMCPExposureHas{}
	}
	entity.Has.RouteMethod = true
}
func (entity *ReportMCPExposure) GetRoutePath() *string {
	return entity.RoutePath
}
func (entity *ReportMCPExposure) SetRoutePath(value *string) {
	entity.RoutePath = value
	if entity.Has == nil {
		entity.Has = &ReportMCPExposureHas{}
	}
	entity.Has.RoutePath = true
}
func (entity *ReportMCPExposure) GetKind() *string {
	return entity.Kind
}
func (entity *ReportMCPExposure) SetKind(value *string) {
	entity.Kind = value
	if entity.Has == nil {
		entity.Has = &ReportMCPExposureHas{}
	}
	entity.Has.Kind = true
}
func (entity *ReportMCPExposure) GetName() *string {
	return entity.Name
}
func (entity *ReportMCPExposure) SetName(value *string) {
	entity.Name = value
	if entity.Has == nil {
		entity.Has = &ReportMCPExposureHas{}
	}
	entity.Has.Name = true
}
func (entity *ReportMCPExposure) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *ReportMCPExposure) GetDescription() *string {
	return entity.Description
}
func (entity *ReportMCPExposure) SetDescription(value *string) {
	entity.Description = value
	if entity.Has == nil {
		entity.Has = &ReportMCPExposureHas{}
	}
	entity.Has.Description = true
}
func (entity *ReportMCPExposure) GetDescriptionPath() *string {
	return entity.DescriptionPath
}
func (entity *ReportMCPExposure) SetDescriptionPath(value *string) {
	entity.DescriptionPath = value
	if entity.Has == nil {
		entity.Has = &ReportMCPExposureHas{}
	}
	entity.Has.DescriptionPath = true
}
func (entity *ReportMCPExposure) GetMimeType() *string {
	return entity.MimeType
}
func (entity *ReportMCPExposure) SetMimeType(value *string) {
	entity.MimeType = value
	if entity.Has == nil {
		entity.Has = &ReportMCPExposureHas{}
	}
	entity.Has.MimeType = true
}
func (entity *ReportMCPExposure) GetEnabled() *int {
	return entity.Enabled
}
func (entity *ReportMCPExposure) SetEnabled(value *int) {
	entity.Enabled = value
	if entity.Has == nil {
		entity.Has = &ReportMCPExposureHas{}
	}
	entity.Has.Enabled = true
}
func (entity *ReportMCPExposure) GetOrdinal() *int {
	return entity.Ordinal
}
func (entity *ReportMCPExposure) SetOrdinal(value *int) {
	entity.Ordinal = value
	if entity.Has == nil {
		entity.Has = &ReportMCPExposureHas{}
	}
	entity.Has.Ordinal = true
}
