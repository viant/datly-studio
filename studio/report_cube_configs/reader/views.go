package reader

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
	ReportId *string `sqlx:"report_id"`
	VersionNo *int `sqlx:"version_no"`
	CubeEnabled *int `sqlx:"cube_enabled"`
	CubeMcpToolEnabled *int `sqlx:"cube_mcp_tool_enabled"`
	LinkedInputType *string `sqlx:"linked_input_type"`
	ComposeEnabled *int `sqlx:"compose_enabled"`
	ComposeMcpToolEnabled *int `sqlx:"compose_mcp_tool_enabled"`
	ComposeMaxCubes *int `sqlx:"compose_max_cubes"`
	ComposeMaxLimit *int `sqlx:"compose_max_limit"`
	ComposeTimeoutMs *int `sqlx:"compose_timeout_ms"`
}
