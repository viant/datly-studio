SELECT r.id AS report_id, v.version_no, r.component_scope, r.component_name,
       r.default_connector_name, c.driver, c.dsn_template,
       COALESCE(c.secret_ref, '') AS secret_ref,
       COALESCE(v.generated_dql, '') AS generated_dql,
       COALESCE(v.authored_dql, '') AS authored_dql
FROM components r
JOIN report_versions v ON v.report_id = r.id AND v.version_no = $VersionNo
JOIN connectors c ON c.name = r.default_connector_name
WHERE r.id = $ReportID
  AND v.compile_status = 'valid'
  AND v.state IN ('published', 'validated')
  AND r.deleted_at IS NULL AND c.deleted_at IS NULL
  AND ($NamespaceID = '' OR EXISTS (SELECT 1 FROM namespaces n
    WHERE n.namespace_id=$NamespaceID AND n.owner_id=r.owner_id AND n.name=r.namespace
      AND n.status='active' AND n.deleted_at IS NULL))
