SELECT r.id, r.namespace, r.slug, r.title, r.description, r.owner_id, r.status,
       r.default_connector_name, r.component_scope, r.component_name,
       r.current_draft_version, r.etag, r.created_at, r.updated_at
FROM reports r
WHERE r.deleted_at IS NULL
  AND (r.owner_id = $Auth.Auth.Subject OR EXISTS (
      SELECT 1 FROM report_acl auth_acl
      WHERE auth_acl.report_id = r.id
        AND auth_acl.subject_type = 'user'
        AND auth_acl.subject_id = $Auth.Auth.Subject
        AND auth_acl.can_view = TRUE
  ))
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(0, "OR"),
    $predicate.FilterGroup(1, "AND"),
    $predicate.FilterGroup(2, "AND")
).Build("AND")}
