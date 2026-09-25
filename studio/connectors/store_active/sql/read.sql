SELECT connector."name", connector."driver", connector."dsn_template", connector."secret_ref", connector."options_json" FROM  (SELECT c.name, c.driver, c.dsn_template, COALESCE(c.secret_ref, '') AS secret_ref, c.options_json
FROM connectors c
WHERE c.deleted_at IS NULL AND c.status = 'active'
)  connector WHERE 1 = 1