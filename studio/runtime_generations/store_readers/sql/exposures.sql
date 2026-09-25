SELECT exposure."report_id", exposure."version_no", exposure."exposure_id", exposure."kind", exposure."name", exposure."route_method", exposure."route_path", exposure."description", exposure."mime_type", exposure."enabled" FROM (SELECT e.report_id, e.version_no, e.exposure_id, e.kind, e.name,
       e.route_method, e.route_path, e.description, e.mime_type, e.enabled
FROM report_mcp_exposures e
ORDER BY e.report_id, e.ordinal, e.exposure_id
) exposure