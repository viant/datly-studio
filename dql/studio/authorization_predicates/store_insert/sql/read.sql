SELECT p.name, p.title, p.description, p.package_path, p.type_name,
       p.sql_scope_json, p.owner_id, p.status, p.etag, p.created_at, p.updated_at
FROM authorization_predicates p
WHERE 1=1
