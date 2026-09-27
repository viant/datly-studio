SELECT connector."name", connector."driver", connector."dsn_template", connector."secret_ref", connector."description", connector."owner_id", connector."status", connector."options_json", connector."etag", connector."created_at", connector."updated_at" FROM  (SELECT c.name, c.driver, c.dsn_template, c.secret_ref, c.description,
       c.owner_id, c.status, c.options_json, c.etag, c.created_at, c.updated_at
FROM connectors c
WHERE c.deleted_at IS NULL
)  connector WHERE 1 = 1