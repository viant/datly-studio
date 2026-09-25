SELECT c.name, c.driver, c.dsn_template, c.secret_ref, c.description,
       c.owner_id, c.status, c.options_json, c.etag, c.created_at, c.updated_at
FROM connectors c
WHERE c.deleted_at IS NULL
