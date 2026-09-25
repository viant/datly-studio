SELECT c.name, c.driver, c.dsn_template, c.secret_ref, c.description,
       c.options_json, c.status, c.last_test_status, c.last_test_error_code,
       c.last_tested_at, c.etag, c.updated_at, c.deleted_at
FROM connectors c
