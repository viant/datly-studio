SELECT connector."name", connector."status", connector."etag", connector."updated_at", connector."last_test_status", connector."deleted_at" FROM  (SELECT c.name, c.status, c.etag, c.updated_at, c.last_test_status, c.deleted_at
FROM connectors c
)  connector WHERE 1 = 1