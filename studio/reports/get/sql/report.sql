SELECT report."id", report."namespace", report."slug", report."title", report."description", report."owner_id", report."owner_package", report."status", report."default_connector_name", report."component_scope", report."component_name", report."current_draft_version", report."etag", report."created_at", report."updated_at" FROM  (SELECT r.id, r.namespace, r.slug, r.title, r.description, r.owner_id,
       '' AS owner_package, r.status, r.default_connector_name,
       r.component_scope, r.component_name, r.current_draft_version,
       r.etag, r.created_at, r.updated_at
FROM reports r
WHERE r.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(2, "AND"),
    $predicate.FilterGroup(3, "AND")
).Build("AND")}
)  report WHERE 1 = 1