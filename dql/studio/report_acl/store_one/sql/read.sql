SELECT a.report_id, a.subject_type, a.subject_id, a.can_view, a.can_run,
       a.can_edit, a.can_publish, a.can_use_dql, a.etag
FROM report_acl a
WHERE a.report_id = $ReportId
  AND a.subject_type = $SubjectType
  AND a.subject_id = $SubjectId
