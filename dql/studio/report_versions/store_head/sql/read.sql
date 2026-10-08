SELECT COALESCE(MAX(v.version_no), 0) AS max_version_no
FROM component_versions v
WHERE v.report_id = $ReportId
