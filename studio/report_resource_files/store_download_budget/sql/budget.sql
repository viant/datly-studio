SELECT budget."file_count", budget."content_bytes" FROM  (SELECT COUNT(*) AS file_count,
       COALESCE(SUM(LENGTH(f.content)), 0) AS content_bytes
FROM component_resource_files f
WHERE f.report_id = $ReportId AND f.version_no = $VersionNo
)  budget WHERE 1 = 1