SELECT predicate."name", predicate."title", predicate."description", predicate."package_path", predicate."type_name", predicate."sql_scope_json", predicate."owner_id", predicate."status", predicate."etag", predicate."created_at", predicate."updated_at", predicate."deleted_at", predicate."should_delete" FROM  (SELECT p.*, '' AS should_delete
FROM authorization_predicates p
)  predicate WHERE 1 = 1