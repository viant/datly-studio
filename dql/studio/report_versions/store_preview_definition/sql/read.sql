SELECT r.id AS report_id, v.version_no, r.component_scope, r.component_name,
       r.default_connector_name, c.driver, c.dsn_template,
       COALESCE(c.secret_ref, '') AS secret_ref,
       COALESCE(c.options_json, '{}') AS options_json,
       v.source_revision, v.spec_hash,
       COALESCE(v.generated_dql, '') AS generated_dql,
       COALESCE(v.authored_dql, '') AS authored_dql
FROM reports r
JOIN connectors c ON c.name = r.default_connector_name
JOIN report_versions v ON v.report_id = r.id AND v.version_no = $VersionNo
WHERE r.id = $ReportId
  AND r.deleted_at IS NULL
  AND c.deleted_at IS NULL
