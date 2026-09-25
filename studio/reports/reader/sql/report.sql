SELECT reports."id", reports."namespace", reports."slug", reports."title", reports."description", reports."owner_id", reports."owner_package", reports."status", reports."default_connector_name", reports."component_scope", reports."component_name", reports."current_draft_version", reports."etag", reports."created_at", reports."updated_at" FROM  (
    SELECT r.id, r.namespace, r.slug, r.title, r.description, r.owner_id, '' AS owner_package, r.status,
       r.default_connector_name, r.component_scope, r.component_name,
       r.current_draft_version, r.etag, r.created_at, r.updated_at
FROM reports r
WHERE r.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(0, "OR"),
    $predicate.FilterGroup(1, "AND"),
    $predicate.FilterGroup(3, "AND")
).Build("AND")}

)  reports WHERE 1 = 1 ORDER BY reports.updated_at DESC, reports.id ASC