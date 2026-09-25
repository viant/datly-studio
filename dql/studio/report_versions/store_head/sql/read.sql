SELECT COALESCE(MAX(v.version_no), 0) AS max_version_no
FROM report_versions v
WHERE v.report_id = $ReportId
