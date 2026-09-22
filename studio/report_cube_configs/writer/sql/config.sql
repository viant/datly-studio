SELECT cube_config."report_id", cube_config."version_no", cube_config."cube_enabled", cube_config."cube_mcp_tool_enabled", cube_config."dimensions_json", cube_config."measures_json", cube_config."filters_json", cube_config."order_by_json", cube_config."input_layout_json", cube_config."linked_input_type", cube_config."compose_enabled", cube_config."compose_mcp_tool_enabled", cube_config."compose_max_cubes", cube_config."compose_max_limit", cube_config."compose_timeout_ms" FROM  (
    SELECT c.*
FROM report_cube_configs c

)  cube_config WHERE 1 = 1