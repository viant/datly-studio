SELECT c.name, c.status, c.etag, c.updated_at, c.last_test_status,
       c.last_test_error_code, c.last_tested_at, c.deleted_at
FROM connectors c
