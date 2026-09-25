SELECT r.id, r.namespace, r.slug, r.title, r.description, r.owner_id,
       r.status, r.default_connector_name, r.component_scope, r.component_name,
       r.current_draft_version, r.etag, r.created_at, r.updated_at
FROM reports r
WHERE r.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(0, "OR"),
    $predicate.FilterGroup(1, "AND"),
    $predicate.FilterGroup(2, "AND"),
    $predicate.FilterGroup(3, "AND")
).Build("AND")}
