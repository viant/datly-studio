SELECT c.name, c.driver, c.dsn_template, COALESCE(c.secret_ref, '') AS secret_ref, c.options_json
FROM connectors c
WHERE c.deleted_at IS NULL AND c.status = 'active'
