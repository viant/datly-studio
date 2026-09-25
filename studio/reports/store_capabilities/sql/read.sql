SELECT capability."report_id", capability."owner_id", capability."can_view", capability."can_run", capability."can_edit", capability."can_publish", capability."can_use_dql" FROM  (SELECT r.id AS report_id, r.owner_id,
       COALESCE(acl.can_view, FALSE) AS can_view,
       COALESCE(acl.can_run, FALSE) AS can_run,
       COALESCE(acl.can_edit, FALSE) AS can_edit,
       COALESCE(acl.can_publish, FALSE) AS can_publish,
       COALESCE(acl.can_use_dql, FALSE) AS can_use_dql
FROM reports r
LEFT JOIN report_acl acl ON acl.report_id = r.id
  AND acl.subject_type = 'user' AND acl.subject_id = $Subject
WHERE r.id = $ReportId AND r.deleted_at IS NULL
)  capability WHERE 1 = 1