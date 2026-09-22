package writer

import (
	json "encoding/json"
)

func (entity *ReportCubeConfig) GetDimensionsJson() json.RawMessage {
	return entity.DimensionsJson
}
func (entity *ReportCubeConfig) SetDimensionsJson(value json.RawMessage) {
	entity.DimensionsJson = value
	if entity.Has == nil {
		entity.Has = &ReportCubeConfigHas{}
	}
	entity.Has.DimensionsJson = true
}
func (entity *ReportCubeConfig) GetMeasuresJson() json.RawMessage {
	return entity.MeasuresJson
}
func (entity *ReportCubeConfig) SetMeasuresJson(value json.RawMessage) {
	entity.MeasuresJson = value
	if entity.Has == nil {
		entity.Has = &ReportCubeConfigHas{}
	}
	entity.Has.MeasuresJson = true
}
func (entity *ReportCubeConfig) GetFiltersJson() json.RawMessage {
	return entity.FiltersJson
}
func (entity *ReportCubeConfig) SetFiltersJson(value json.RawMessage) {
	entity.FiltersJson = value
	if entity.Has == nil {
		entity.Has = &ReportCubeConfigHas{}
	}
	entity.Has.FiltersJson = true
}
func (entity *ReportCubeConfig) GetOrderByJson() json.RawMessage {
	return entity.OrderByJson
}
func (entity *ReportCubeConfig) SetOrderByJson(value json.RawMessage) {
	entity.OrderByJson = value
	if entity.Has == nil {
		entity.Has = &ReportCubeConfigHas{}
	}
	entity.Has.OrderByJson = true
}
func (entity *ReportCubeConfig) GetInputLayoutJson() json.RawMessage {
	return entity.InputLayoutJson
}
func (entity *ReportCubeConfig) SetInputLayoutJson(value json.RawMessage) {
	entity.InputLayoutJson = value
	if entity.Has == nil {
		entity.Has = &ReportCubeConfigHas{}
	}
	entity.Has.InputLayoutJson = true
}
func (entity *ReportCubeConfig) GetReportId() *string {
	return entity.ReportId
}
func (entity *ReportCubeConfig) SetReportId(value *string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &ReportCubeConfigHas{}
	}
	entity.Has.ReportId = true
}
func (entity *ReportCubeConfig) GetVersionNo() *int {
	return entity.VersionNo
}
func (entity *ReportCubeConfig) SetVersionNo(value *int) {
	entity.VersionNo = value
	if entity.Has == nil {
		entity.Has = &ReportCubeConfigHas{}
	}
	entity.Has.VersionNo = true
}
func (entity *ReportCubeConfig) GetComposeMaxCubes() *int {
	return entity.ComposeMaxCubes
}
func (entity *ReportCubeConfig) SetComposeMaxCubes(value *int) {
	entity.ComposeMaxCubes = value
	if entity.Has == nil {
		entity.Has = &ReportCubeConfigHas{}
	}
	entity.Has.ComposeMaxCubes = true
}
func (entity *ReportCubeConfig) GetComposeMaxLimit() *int {
	return entity.ComposeMaxLimit
}
func (entity *ReportCubeConfig) SetComposeMaxLimit(value *int) {
	entity.ComposeMaxLimit = value
	if entity.Has == nil {
		entity.Has = &ReportCubeConfigHas{}
	}
	entity.Has.ComposeMaxLimit = true
}
func (entity *ReportCubeConfig) GetComposeTimeoutMs() *int {
	return entity.ComposeTimeoutMs
}
func (entity *ReportCubeConfig) SetComposeTimeoutMs(value *int) {
	entity.ComposeTimeoutMs = value
	if entity.Has == nil {
		entity.Has = &ReportCubeConfigHas{}
	}
	entity.Has.ComposeTimeoutMs = true
}
func (entity *ReportCubeConfig) GetCubeEnabled() *int {
	return entity.CubeEnabled
}
func (entity *ReportCubeConfig) SetCubeEnabled(value *int) {
	entity.CubeEnabled = value
	if entity.Has == nil {
		entity.Has = &ReportCubeConfigHas{}
	}
	entity.Has.CubeEnabled = true
}
func (entity *ReportCubeConfig) GetCubeMcpToolEnabled() *int {
	return entity.CubeMcpToolEnabled
}
func (entity *ReportCubeConfig) SetCubeMcpToolEnabled(value *int) {
	entity.CubeMcpToolEnabled = value
	if entity.Has == nil {
		entity.Has = &ReportCubeConfigHas{}
	}
	entity.Has.CubeMcpToolEnabled = true
}
func (entity *ReportCubeConfig) GetLinkedInputType() *string {
	return entity.LinkedInputType
}
func (entity *ReportCubeConfig) SetLinkedInputType(value *string) {
	entity.LinkedInputType = value
	if entity.Has == nil {
		entity.Has = &ReportCubeConfigHas{}
	}
	entity.Has.LinkedInputType = true
}
func (entity *ReportCubeConfig) GetComposeEnabled() *int {
	return entity.ComposeEnabled
}
func (entity *ReportCubeConfig) SetComposeEnabled(value *int) {
	entity.ComposeEnabled = value
	if entity.Has == nil {
		entity.Has = &ReportCubeConfigHas{}
	}
	entity.Has.ComposeEnabled = true
}
func (entity *ReportCubeConfig) GetComposeMcpToolEnabled() *int {
	return entity.ComposeMcpToolEnabled
}
func (entity *ReportCubeConfig) SetComposeMcpToolEnabled(value *int) {
	entity.ComposeMcpToolEnabled = value
	if entity.Has == nil {
		entity.Has = &ReportCubeConfigHas{}
	}
	entity.Has.ComposeMcpToolEnabled = true
}
