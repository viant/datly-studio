SELECT access."report_id" FROM  (SELECT r.id AS report_id
FROM reports r
WHERE r.id = $ReportId
  AND r.deleted_at IS NULL
  AND (r.owner_id = $Subject OR EXISTS (
    SELECT 1 FROM report_acl acl
    WHERE acl.report_id = r.id
      AND acl.subject_type = 'user'
      AND acl.subject_id = $Subject
      AND acl.can_run = TRUE
  ))
)  access WHERE 1 = 1