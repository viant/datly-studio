SELECT cube_config."report_id", cube_config."version_no", cube_config."cube_enabled", cube_config."cube_mcp_tool_enabled", cube_config."dimensions_json", cube_config."measures_json", cube_config."filters_json", cube_config."order_by_json", cube_config."input_layout_json", cube_config."linked_input_type", cube_config."compose_enabled", cube_config."compose_mcp_tool_enabled", cube_config."compose_max_cubes", cube_config."compose_max_limit", cube_config."compose_timeout_ms" FROM  (
    SELECT c.report_id, c.version_no, c.cube_enabled,
       c.cube_mcp_tool_enabled, c.dimensions_json, c.measures_json,
       c.filters_json, c.order_by_json, c.input_layout_json,
       c.linked_input_type, c.compose_enabled,
       c.compose_mcp_tool_enabled, c.compose_max_cubes,
       c.compose_max_limit, c.compose_timeout_ms
FROM report_cube_configs c
WHERE c.report_id = $ReportId
  AND c.version_no = $VersionNo
${predicate.Builder().CombineAnd($predicate.FilterGroup(3, "AND")).Build("AND")}

)  cube_config WHERE 1 = 1