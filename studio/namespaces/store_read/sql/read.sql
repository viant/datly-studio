SELECT namespace."owner_id", namespace."name", namespace."title", namespace."description", namespace."status", namespace."etag", namespace."created_at", namespace."updated_at" FROM  (SELECT n.owner_id, n.name, n.title, n.description, n.status, n.etag,
       n.created_at, n.updated_at
FROM namespaces n
WHERE n.deleted_at IS NULL
  AND ($OwnerId = '' OR n.owner_id = $OwnerId)
  AND ($Name = '' OR n.name = $Name)
  AND ($Query = '' OR LOWER(n.name) LIKE $Query OR LOWER(n.title) LIKE $Query OR LOWER(n.description) LIKE $Query)
  AND ($Status = '' OR n.status = $Status)
  AND ($Scoped = FALSE OR n.owner_id = $Subject OR EXISTS (
      SELECT 1 FROM reports namespace_report
      JOIN report_acl namespace_acl ON namespace_acl.report_id = namespace_report.id
      WHERE namespace_report.owner_id = n.owner_id
        AND namespace_report.namespace = n.name
        AND namespace_report.deleted_at IS NULL
        AND namespace_acl.subject_type = 'user'
        AND namespace_acl.subject_id = $Subject
        AND namespace_acl.can_view = TRUE
  ))
ORDER BY n.updated_at DESC, n.name ASC
LIMIT $PageLimit OFFSET $PageOffset
)  namespace WHERE 1 = 1