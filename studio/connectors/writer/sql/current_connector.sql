SELECT r."options_json", r."name", r."driver", r."dsn_template", r."secret_ref", r."etag", r."description", r."owner_id", r."status", r."last_test_status", r."last_test_error_code", r."last_tested_at", r."created_at", r."updated_at", r."deleted_at" FROM (SELECT connector."name", connector."driver", connector."dsn_template", connector."secret_ref", connector."description", connector."owner_id", connector."status", connector."options_json", connector."last_test_status", connector."last_test_error_code", connector."last_tested_at", connector."etag", connector."created_at", connector."updated_at", connector."deleted_at", connector."should_delete" FROM  (
    SELECT c.*, '' AS should_delete
FROM connectors c

)  connector WHERE 1 = 1) r WHERE $criteria.CompositeIn("r", $ConnectorKeys)