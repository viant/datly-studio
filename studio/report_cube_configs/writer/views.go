package writer

import (
	json "encoding/json"
)

// ReportCubeConfig is generated canonical view metadata for config.
type ReportCubeConfig struct {
	DimensionsJson json.RawMessage `sqlx:"dimensions_json,enc=JSON"`
	MeasuresJson json.RawMessage `sqlx:"measures_json,enc=JSON"`
	FiltersJson json.RawMessage `sqlx:"filters_json,enc=JSON"`
	OrderByJson json.RawMessage `sqlx:"order_by_json,enc=JSON"`
	InputLayoutJson json.RawMessage `sqlx:"input_layout_json,enc=JSON"`
	ReportId *string `sqlx:"report_id,primaryKey,refTable=report_versions,refColumn=report_id,required=true" validate:"required"`
	VersionNo *int `sqlx:"version_no,primaryKey,refTable=report_versions,refColumn=version_no,required=true"`
	ComposeMaxCubes *int `validate:"gt=0" sqlx:"compose_max_cubes"`
	ComposeMaxLimit *int `validate:"gt=0" sqlx:"compose_max_limit"`
	ComposeTimeoutMs *int `validate:"gt=0" sqlx:"compose_timeout_ms"`
	CubeEnabled *int `sqlx:"cube_enabled,required=true"`
	CubeMcpToolEnabled *int `sqlx:"cube_mcp_tool_enabled"`
	LinkedInputType *string `sqlx:"linked_input_type"`
	ComposeEnabled *int `sqlx:"compose_enabled,required=true"`
	ComposeMcpToolEnabled *int `sqlx:"compose_mcp_tool_enabled"`
	Has *ReportCubeConfigHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"ReportCubeConfigHas"`
}

type ReportCubeConfigHas struct {
	DimensionsJson bool
	MeasuresJson bool
	FiltersJson bool
	OrderByJson bool
	InputLayoutJson bool
	ReportId bool
	VersionNo bool
	ComposeMaxCubes bool
	ComposeMaxLimit bool
	ComposeTimeoutMs bool
	CubeEnabled bool
	CubeMcpToolEnabled bool
	LinkedInputType bool
	ComposeEnabled bool
	ComposeMcpToolEnabled bool
}

// CurrentConfigView is generated canonical view metadata for config.
type CurrentConfigView struct {
	DimensionsJson json.RawMessage `sqlx:"dimensions_json,enc=JSON"`
	MeasuresJson json.RawMessage `sqlx:"measures_json,enc=JSON"`
	FiltersJson json.RawMessage `sqlx:"filters_json,enc=JSON"`
	OrderByJson json.RawMessage `sqlx:"order_by_json,enc=JSON"`
	InputLayoutJson json.RawMessage `sqlx:"input_layout_json,enc=JSON"`
	ReportId *string `sqlx:"report_id,primaryKey,refTable=report_versions,refColumn=report_id,required=true" validate:"required"`
	VersionNo *int `sqlx:"version_no,primaryKey,refTable=report_versions,refColumn=version_no,required=true"`
	ComposeMaxCubes *int `validate:"gt=0" sqlx:"compose_max_cubes"`
	ComposeMaxLimit *int `validate:"gt=0" sqlx:"compose_max_limit"`
	ComposeTimeoutMs *int `validate:"gt=0" sqlx:"compose_timeout_ms"`
	CubeEnabled *int `sqlx:"cube_enabled,required=true"`
	CubeMcpToolEnabled *int `sqlx:"cube_mcp_tool_enabled"`
	LinkedInputType *string `sqlx:"linked_input_type"`
	ComposeEnabled *int `sqlx:"compose_enabled,required=true"`
	ComposeMcpToolEnabled *int `sqlx:"compose_mcp_tool_enabled"`
}

type ConfigKeysRow struct {
	ReportId *string `sqlx:"report_id"`
	VersionNo *int `sqlx:"version_no"`
}
