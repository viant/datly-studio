SELECT r."name", r."status", r."etag", r."updated_at", r."last_test_status", r."last_test_error_code", r."last_tested_at", r."deleted_at" FROM (SELECT connector."name", connector."status", connector."etag", connector."updated_at", connector."last_test_status", connector."last_test_error_code", connector."last_tested_at", connector."deleted_at" FROM  (SELECT c.name, c.status, c.etag, c.updated_at, c.last_test_status,
       c.last_test_error_code, c.last_tested_at, c.deleted_at
FROM connectors c
)  connector WHERE 1 = 1) r WHERE $criteria.CompositeIn("r", $ConnectorKeys)