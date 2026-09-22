SELECT exposure."report_id", exposure."version_no", exposure."exposure_id", exposure."route_id", exposure."route_method", exposure."route_path", exposure."kind", exposure."name", exposure."description", exposure."description_path", exposure."mime_type", exposure."enabled", exposure."ordinal", exposure."should_delete" FROM  (
    SELECT e.*, '' AS should_delete
FROM report_mcp_exposures e

)  exposure WHERE 1 = 1