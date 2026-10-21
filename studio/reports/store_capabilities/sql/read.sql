SELECT capability.* FROM  (SELECT r.id AS report_id, r.owner_id, CAST(n.allowed_roles_json AS CHAR) AS namespace_roles,
       COALESCE(acl.can_view, FALSE) AS can_view,
       COALESCE(acl.can_run, FALSE) AS can_run,
       COALESCE(acl.can_edit, FALSE) AS can_edit,
       COALESCE(acl.can_publish, FALSE) AS can_publish,
       COALESCE(acl.can_use_dql, FALSE) AS can_use_dql
FROM components r
LEFT JOIN namespaces n ON n.owner_id=r.owner_id AND n.name=r.namespace AND n.deleted_at IS NULL AND n.status='active'
LEFT JOIN component_acl acl ON acl.report_id = r.id
  AND acl.subject_type = 'user' AND acl.subject_id = $Subject
WHERE r.id = $ReportId AND r.deleted_at IS NULL
)  capability WHERE 1 = 1