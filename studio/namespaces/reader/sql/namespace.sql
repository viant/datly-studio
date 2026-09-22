SELECT namespaces."owner_id", namespaces."name", namespaces."title", namespaces."description", namespaces."status", namespaces."etag", namespaces."created_at", namespaces."updated_at" FROM  (
    SELECT n.owner_id, n.name, n.title, n.description, n.status,
       n.etag, n.created_at, n.updated_at
FROM namespaces n
WHERE n.deleted_at IS NULL
  AND (n.owner_id = $Auth.Auth.Subject OR EXISTS (
      SELECT 1 FROM reports namespace_report
      JOIN report_acl namespace_acl ON namespace_acl.report_id = namespace_report.id
      WHERE namespace_report.owner_id = n.owner_id
        AND namespace_report.namespace = n.name
        AND namespace_report.deleted_at IS NULL
        AND namespace_acl.subject_type = 'user'
        AND namespace_acl.subject_id = $Auth.Auth.Subject
        AND namespace_acl.can_view = TRUE
  ))
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(0, "OR"),
    $predicate.FilterGroup(1, "AND"),
    $predicate.FilterGroup(2, "AND")
).Build("AND")}

)  namespaces WHERE 1 = 1