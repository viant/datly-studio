SELECT predicate."name", predicate."title", predicate."description", predicate."package_path", predicate."type_name", predicate."sql_scope_json", predicate."owner_id", predicate."status", predicate."etag", predicate."created_at", predicate."updated_at" FROM  (SELECT p.name, p.title, p.description, p.package_path, p.type_name,
       p.sql_scope_json, p.owner_id, p.status, p.etag, p.created_at, p.updated_at
FROM authorization_predicates p
WHERE p.deleted_at IS NULL
  AND ($SearchPattern = '' OR (
       LOWER(p.name) LIKE $SearchPattern OR LOWER(p.title) LIKE $SearchPattern OR
       LOWER(p.description) LIKE $SearchPattern OR LOWER(p.package_path) LIKE $SearchPattern OR
       LOWER(p.type_name) LIKE $SearchPattern))
  AND ($Status = '' OR p.status = $Status)
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("AND")}
)  predicate WHERE 1 = 1