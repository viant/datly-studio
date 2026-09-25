SELECT c.name, c.driver, c.dsn_template,
       COALESCE(c.secret_ref, '') AS secret_ref,
       COALESCE(c.options_json, '{}') AS options_json
FROM connectors c
WHERE c.deleted_at IS NULL AND c.status = 'active'
  AND (c.owner_id = $Subject OR EXISTS (
    SELECT 1 FROM reports r
    JOIN report_acl acl ON acl.report_id = r.id
    WHERE r.default_connector_name = c.name
      AND r.deleted_at IS NULL
      AND acl.subject_type = 'user'
      AND acl.subject_id = $Subject
      AND acl.can_view = TRUE
  ))
ORDER BY c.name
