SELECT r."report_id", r."version_no", r."exposure_id", r."route_id", r."route_method", r."route_path", r."kind", r."name", r."description", r."description_path", r."mime_type", r."enabled", r."ordinal" FROM (SELECT exposure."report_id", exposure."version_no", exposure."exposure_id", exposure."route_id", exposure."route_method", exposure."route_path", exposure."kind", exposure."name", exposure."description", exposure."description_path", exposure."mime_type", exposure."enabled", exposure."ordinal", exposure."should_delete" FROM  (
    SELECT e.*, '' AS should_delete
FROM report_mcp_exposures e

)  exposure WHERE 1 = 1) r WHERE $criteria.CompositeIn("r", $ExposureKeys)