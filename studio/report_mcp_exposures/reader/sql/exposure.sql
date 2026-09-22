SELECT mcp_exposure."report_id", mcp_exposure."version_no", mcp_exposure."exposure_id", mcp_exposure."route_id", mcp_exposure."route_method", mcp_exposure."route_path", mcp_exposure."kind", mcp_exposure."name", mcp_exposure."description", mcp_exposure."description_path", mcp_exposure."mime_type", mcp_exposure."enabled", mcp_exposure."ordinal" FROM  (
    SELECT e.report_id, e.version_no, e.exposure_id, e.route_id,
       e.route_method, e.route_path, e.kind, e.name, e.description,
       e.description_path, e.mime_type, e.enabled, e.ordinal
FROM report_mcp_exposures e
WHERE e.report_id = $ReportId
  AND e.version_no = $VersionNo
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(1, "AND"),
    $predicate.FilterGroup(2, "AND"),
    $predicate.FilterGroup(3, "AND")
).Build("AND")}

)  mcp_exposure WHERE 1 = 1