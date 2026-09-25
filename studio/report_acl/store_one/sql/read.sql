SELECT acl."report_id", acl."subject_type", acl."subject_id", acl."can_view", acl."can_run", acl."can_edit", acl."can_publish", acl."can_use_dql", acl."etag" FROM  (SELECT a.report_id, a.subject_type, a.subject_id, a.can_view, a.can_run,
       a.can_edit, a.can_publish, a.can_use_dql, a.etag
FROM report_acl a
WHERE a.report_id = $ReportId
  AND a.subject_type = $SubjectType
  AND a.subject_id = $SubjectId
)  acl WHERE 1 = 1