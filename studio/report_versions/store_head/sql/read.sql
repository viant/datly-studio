SELECT head."max_version_no" FROM  (SELECT COALESCE(MAX(v.version_no), 0) AS max_version_no
FROM report_versions v
WHERE v.report_id = $ReportId
)  head WHERE 1 = 1